package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/mtiluk/poziv/internal/config"
	"github.com/mtiluk/poziv/internal/discord"
	"github.com/mtiluk/poziv/internal/ebay"
	"github.com/mtiluk/poziv/internal/seen"
)

const configPath = "config.json"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ec := ebay.NewClient(cfg.EbayURL, cfg.BasicCreds)

	st, err := loadSettings(configPath)
	if err != nil {
		log.Fatal(err)
	}
	searches := st.Searches

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
			nextPoll[i] = now.Add(effectiveInterval(now, time.Duration(s.Interval), st.QuietHours))
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
