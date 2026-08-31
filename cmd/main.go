package main

import (
	"context"
	"log"
	"time"

	"github.com/mtiluk/poziv/internal/config"
	"github.com/mtiluk/poziv/internal/discord"
	"github.com/mtiluk/poziv/internal/ebay"
	"github.com/mtiluk/poziv/internal/seen"
)

func main() {
	cfg := config.GetConfig()
	ec := ebay.NewClient(cfg.EbayURL, cfg.BasicCreds)

	store, err := seen.Open("data/seen.json", 30*24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	for {
		if err := poll(ec, store, cfg.DiscordWebhookURL); err != nil {
			log.Println("poll:", err)
		}
		time.Sleep(30 * time.Second)
	}
}

func poll(ec *ebay.Client, store *seen.Seen, webhook string) error {
	res, err := ec.Search(context.Background(), ebay.SearchParams{
		Query: "bambu lab",
		Limit: 50,
		Sort:  "newlyListed",
	})
	if err != nil {
		return err
	}

	first := store.IsEmpty()

	for _, it := range res.ItemSummaries {
		if store.Has(it.ItemID) {
			continue
		}
		if !first {
			err := discord.SendEmbed(webhook, it.Title, it.Price.Value, it.ItemWebURL, it.Image.ImageURL)
			if err != nil {
				log.Println("notify:", err)
				continue
			}
		}
		store.Add(it.ItemID)
	}

	return store.Save()
}
