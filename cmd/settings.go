package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	_ "time/tzdata"
)

type settings struct {
	QuietHours *quietHours `json:"quietHours"`
	Searches   []search    `json:"searches"`
}

type search struct {
	Name     string   `json:"name"`
	Query    string   `json:"query"`
	MinPrice int      `json:"minPrice"`
	MaxPrice int      `json:"maxPrice"`
	Interval duration `json:"interval"`
}

type quietHours struct {
	Start      clock  `json:"start"`
	End        clock  `json:"end"`
	Timezone   string `json:"timezone"`
	Multiplier int    `json:"multiplier"`

	loc *time.Location
}

type duration time.Duration

func (d *duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("interval must be a duration string like \"60s\": %w", err)
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = duration(v)
	return nil
}

type clock int

func (c *clock) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("time must be a string like \"02:00\": %w", err)
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return fmt.Errorf("invalid time %q, expected HH:MM: %w", s, err)
	}
	*c = clock(t.Hour()*60 + t.Minute())
	return nil
}

func (q *quietHours) contains(t time.Time) bool {
	if q == nil {
		return false
	}
	lt := t.In(q.loc)
	m := clock(lt.Hour()*60 + lt.Minute())
	if q.Start < q.End {
		return m >= q.Start && m < q.End
	}
	return m >= q.Start || m < q.End
}

func (q *quietHours) validate() error {
	if q.Start == q.End {
		return fmt.Errorf("quietHours: start and end must differ")
	}
	if q.Multiplier < 1 {
		return fmt.Errorf("quietHours: multiplier must be at least 1")
	}
	tz := q.Timezone
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("quietHours: invalid timezone %q: %w", q.Timezone, err)
	}
	q.loc = loc
	return nil
}

func loadSettings(path string) (*settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var s settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if s.QuietHours != nil {
		if err := s.QuietHours.validate(); err != nil {
			return nil, err
		}
	}
	if len(s.Searches) == 0 {
		return nil, fmt.Errorf("no searches defined in %s", path)
	}
	for i, sr := range s.Searches {
		if sr.Query == "" {
			return nil, fmt.Errorf("search %d (%q): query is required", i, sr.Name)
		}
		if sr.Name == "" {
			s.Searches[i].Name = sr.Query
		}
		if sr.Interval <= 0 {
			return nil, fmt.Errorf("search %q: interval must be positive", s.Searches[i].Name)
		}
	}
	return &s, nil
}

func effectiveInterval(now time.Time, interval time.Duration, q *quietHours) time.Duration {
	if q.contains(now) {
		return interval * time.Duration(q.Multiplier)
	}
	return interval
}
