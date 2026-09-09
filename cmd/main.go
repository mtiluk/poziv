package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mtiluk/poziv/internal/config"
	"github.com/mtiluk/poziv/internal/discord"
	"github.com/mtiluk/poziv/internal/ebay"
	"github.com/mtiluk/poziv/internal/seen"
)

// searches lists every eBay search this bot tracks. Add an entry here to
// track another item; Query is a free-text eBay keyword search, so extra
// specs (RAM, size, etc.) are just appended keywords. MinPrice is in GBP
// and excludes listings priced below it.
var searches = []struct {
	Name     string
	Query    string
	MinPrice int
}{
	{Name: "Bambu Lab", Query: "bambu lab", MinPrice: 20},
	{Name: "Lenovo Legion 5 16GB RAM", Query: "lenovo legion 5 16gb ram", MinPrice: 200},
}

func main() {
	cfg := config.GetConfig()
	ec := ebay.NewClient(cfg.EbayURL, cfg.BasicCreds)

	store, err := seen.Open("data/seen.json", 30*24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	for {
		first := store.IsEmpty()
		for _, s := range searches {
			if err := poll(ec, store, cfg.DiscordWebhookURL, s.Name, s.Query, s.MinPrice, first); err != nil {
				log.Println("poll:", s.Name, err)
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func poll(ec *ebay.Client, store *seen.Seen, webhook, name, query string, minPrice int, first bool) error {
	var filter string
	if minPrice > 0 {
		filter = fmt.Sprintf("price:[%d..],priceCurrency:GBP", minPrice)
	}

	res, err := ec.Search(context.Background(), ebay.SearchParams{
		Query:  query,
		Limit:  50,
		Sort:   "newlyListed",
		Filter: filter,
	})
	if err != nil {
		return err
	}

	for _, it := range res.ItemSummaries {
		if store.Has(it.ItemID) {
			continue
		}
		if !first {
			err := discord.SendEmbed(webhook, name, it.Title, it.Price.Value, it.ItemWebURL, it.Image.ImageURL)
			if err != nil {
				log.Println("notify:", err)
				continue
			}
		}
		store.Add(it.ItemID)
	}

	return store.Save()
}
