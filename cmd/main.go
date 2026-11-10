package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/mtiluk/poziv/internal/config"
	"github.com/mtiluk/poziv/internal/discord"
	"github.com/mtiluk/poziv/internal/ebay"
	"github.com/mtiluk/poziv/internal/seen"
)

const (
	quietHourStart  = 2
	quietHourEnd    = 6
	quietMultiplier = 5
	searchesPath    = "searches.json"
)

type search struct {
	Name     string   `json:"name"`
	Query    string   `json:"query"`
	MinPrice int      `json:"minPrice"`
	MaxPrice int      `json:"maxPrice"`
	Interval duration `json:"interval"`
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

func loadSearches(path string) ([]search, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read searches: %w", err)
	}
	var searches []search
	if err := json.Unmarshal(data, &searches); err != nil {
		return nil, fmt.Errorf("parse searches: %w", err)
	}
	if len(searches) == 0 {
		return nil, fmt.Errorf("no searches defined in %s", path)
	}
	for i, s := range searches {
		if s.Query == "" {
			return nil, fmt.Errorf("search %d (%q): query is required", i, s.Name)
		}
		if s.Name == "" {
			searches[i].Name = s.Query
		}
		if s.Interval <= 0 {
			return nil, fmt.Errorf("search %q: interval must be positive", searches[i].Name)
		}
	}
	return searches, nil
}

func isQuietHours(t time.Time) bool {
	h := t.UTC().Hour()
	return h >= quietHourStart && h < quietHourEnd
}

func effectiveInterval(base time.Time, interval time.Duration) time.Duration {
	if isQuietHours(base) {
		return interval * quietMultiplier
	}
	return interval
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ec := ebay.NewClient(cfg.EbayURL, cfg.BasicCreds)

	searches, err := loadSearches(searchesPath)
	if err != nil {
		log.Fatal(err)
	}

	store, err := seen.Open("data/seen.json", 30*24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	nextPoll := make([]time.Time, len(searches))

	for {
		now := time.Now()
		for i, s := range searches {
			if now.Before(nextPoll[i]) {
				continue
			}
			if err := poll(ec, store, cfg.DiscordWebhookURL, s); err != nil {
				log.Println("poll:", s.Name, err)
			}
			nextPoll[i] = now.Add(effectiveInterval(now, time.Duration(s.Interval)))
		}
		time.Sleep(time.Second)
	}
}

func primedKey(s search) string {
	return fmt.Sprintf("search:%s|%d|%d", s.Query, s.MinPrice, s.MaxPrice)
}

func priceFilter(minPrice, maxPrice int) string {
	if minPrice <= 0 && maxPrice <= 0 {
		return ""
	}
	return fmt.Sprintf("price:[%s..%s],priceCurrency:GBP", priceBound(minPrice), priceBound(maxPrice))
}

func poll(ec *ebay.Client, store *seen.Seen, webhook string, s search) error {
	res, err := ec.Search(context.Background(), ebay.SearchParams{
		Query:  s.Query,
		Limit:  50,
		Sort:   "newlyListed",
		Filter: priceFilter(s.MinPrice, s.MaxPrice),
	})
	if err != nil {
		return err
	}

	key := primedKey(s)
	silent := !store.Has(key)
	if silent {
		log.Printf("poll: %s: first run, recording %d existing listings without notifying", s.Name, len(res.ItemSummaries))
	}

	for _, it := range res.ItemSummaries {
		if store.Has(it.ItemID) {
			continue
		}
		if !silent {
			err := discord.SendEmbed(webhook, s.Name, it.Title, it.Price.Value, it.ItemWebURL, it.Image.ImageURL)
			if errors.Is(err, discord.ErrRateLimited) {
				log.Println("notify:", err)
				break
			}
			if err != nil {
				log.Println("notify:", err)
				continue
			}
		}
		store.Add(it.ItemID)
	}
	store.Add(key)

	return store.Save()
}

func priceBound(v int) string {
	if v <= 0 {
		return ""
	}
	return strconv.Itoa(v)
}
