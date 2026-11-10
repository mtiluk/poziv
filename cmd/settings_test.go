package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustQuiet(t *testing.T, start, end, tz string, mult int) *quietHours {
	t.Helper()
	q := &quietHours{Timezone: tz, Multiplier: mult}
	if err := q.Start.UnmarshalJSON([]byte(`"` + start + `"`)); err != nil {
		t.Fatal(err)
	}
	if err := q.End.UnmarshalJSON([]byte(`"` + end + `"`)); err != nil {
		t.Fatal(err)
	}
	if err := q.validate(); err != nil {
		t.Fatal(err)
	}
	return q
}

func at(h, m int, loc *time.Location) time.Time {
	return time.Date(2026, 1, 15, h, m, 0, 0, loc)
}

func TestQuietHoursSameDay(t *testing.T) {
	q := mustQuiet(t, "02:00", "06:00", "UTC", 5)
	cases := []struct {
		h, m int
		want bool
	}{
		{1, 59, false}, {2, 0, true}, {5, 59, true}, {6, 0, false}, {12, 0, false},
	}
	for _, c := range cases {
		if got := q.contains(at(c.h, c.m, time.UTC)); got != c.want {
			t.Errorf("%02d:%02d: got %v, want %v", c.h, c.m, got, c.want)
		}
	}
}

func TestQuietHoursAcrossMidnight(t *testing.T) {
	q := mustQuiet(t, "22:30", "06:00", "UTC", 3)
	cases := []struct {
		h, m int
		want bool
	}{
		{22, 29, false}, {22, 30, true}, {23, 59, true}, {0, 0, true}, {5, 59, true}, {6, 0, false}, {12, 0, false},
	}
	for _, c := range cases {
		if got := q.contains(at(c.h, c.m, time.UTC)); got != c.want {
			t.Errorf("%02d:%02d: got %v, want %v", c.h, c.m, got, c.want)
		}
	}
}

func TestQuietHoursTimezone(t *testing.T) {
	q := mustQuiet(t, "02:00", "06:00", "America/New_York", 5)
	if !q.contains(at(8, 0, time.UTC)) {
		t.Error("08:00 UTC is 03:00 in New York in January and should be quiet")
	}
	if q.contains(at(3, 0, time.UTC)) {
		t.Error("03:00 UTC is 22:00 in New York and should not be quiet")
	}
}

func TestQuietHoursFollowsDST(t *testing.T) {
	q := mustQuiet(t, "02:00", "06:00", "Europe/London", 5)
	summer := time.Date(2026, 7, 1, 1, 30, 0, 0, time.UTC)
	if !q.contains(summer) {
		t.Error("01:30 UTC in July is 02:30 BST and should be quiet")
	}
}

func TestQuietHoursDefaultTimezoneIsUTC(t *testing.T) {
	q := mustQuiet(t, "02:00", "06:00", "", 5)
	if q.loc != time.UTC {
		t.Errorf("got %v, want UTC", q.loc)
	}
}

func TestEffectiveInterval(t *testing.T) {
	q := mustQuiet(t, "02:00", "06:00", "UTC", 5)
	if got := effectiveInterval(at(3, 0, time.UTC), time.Minute, q); got != 5*time.Minute {
		t.Errorf("quiet: got %v", got)
	}
	if got := effectiveInterval(at(12, 0, time.UTC), time.Minute, q); got != time.Minute {
		t.Errorf("busy: got %v", got)
	}
	if got := effectiveInterval(at(3, 0, time.UTC), time.Minute, nil); got != time.Minute {
		t.Errorf("disabled: got %v", got)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadSettings(t *testing.T) {
	p := writeConfig(t, `{
		"quietHours": {"start": "23:00", "end": "07:30", "timezone": "Europe/Berlin", "multiplier": 4},
		"searches": [{"query": "Wooting", "maxPrice": 150, "interval": "90s"}]
	}`)
	st, err := loadSettings(p)
	if err != nil {
		t.Fatal(err)
	}
	q := st.QuietHours
	if q == nil || q.Start != 23*60 || q.End != 7*60+30 || q.Multiplier != 4 || q.loc.String() != "Europe/Berlin" {
		t.Errorf("unexpected quiet hours: %+v", q)
	}
	if len(st.Searches) != 1 {
		t.Fatalf("got %d searches", len(st.Searches))
	}
	s := st.Searches[0]
	if s.Name != "Wooting" {
		t.Errorf("name should default to query, got %q", s.Name)
	}
	if time.Duration(s.Interval) != 90*time.Second {
		t.Errorf("interval: got %v", time.Duration(s.Interval))
	}
}

func TestLoadSettingsWithoutQuietHours(t *testing.T) {
	st, err := loadSettings(writeConfig(t, `{"searches": [{"query": "a", "interval": "1m"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if st.QuietHours != nil {
		t.Error("quiet hours should be disabled when omitted")
	}
}

func TestLoadSettingsErrors(t *testing.T) {
	ok := `"searches": [{"query": "a", "interval": "1m"}]`
	cases := map[string]string{
		"no searches":       `{"searches": []}`,
		"missing query":     `{"searches": [{"name": "a", "interval": "1m"}]}`,
		"missing interval":  `{"searches": [{"query": "a"}]}`,
		"bad interval":      `{"searches": [{"query": "a", "interval": "soon"}]}`,
		"numeric interval":  `{"searches": [{"query": "a", "interval": 60}]}`,
		"bad time":          `{"quietHours": {"start": "2am", "end": "06:00", "multiplier": 5}, ` + ok + `}`,
		"hour out of range": `{"quietHours": {"start": "25:00", "end": "06:00", "multiplier": 5}, ` + ok + `}`,
		"equal start/end":   `{"quietHours": {"start": "02:00", "end": "02:00", "multiplier": 5}, ` + ok + `}`,
		"bad timezone":      `{"quietHours": {"start": "02:00", "end": "06:00", "timezone": "Mars/Base", "multiplier": 5}, ` + ok + `}`,
		"zero multiplier":   `{"quietHours": {"start": "02:00", "end": "06:00"}, ` + ok + `}`,
		"old array format":  `[{"query": "a", "interval": "1m"}]`,
		"invalid json":      `{`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := loadSettings(writeConfig(t, content)); err == nil {
				t.Error("expected error")
			}
		})
	}
	if _, err := loadSettings(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Error("expected error for missing file")
	}
}
