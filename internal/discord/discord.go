package discord

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type Embed struct {
	Image       *EmbedImage  `json:"image,omitempty"`
	Title       string       `json:"title,omitempty"`
	URL         string       `json:"url,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedFooter struct {
	Text string `json:"text"`
}

type EmbedPayload struct {
	Embeds []Embed `json:"embeds"`
}

type EmbedImage struct {
	URL string `json:"url"`
}

func SendEmbed(webhook string, title string, price string, url string, imageURL string) error {
	embed := Embed{
		Title:       title,
		URL:         url,
		Description: "A new item has been added to eBay.",
		Color:       5763719,
		Fields: []EmbedField{
			{Name: "Price", Value: price, Inline: true}},
		Footer:    &EmbedFooter{Text: "Monitoring Agent"},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if imageURL != "" {
		embed.Image = &EmbedImage{URL: imageURL}
	}

	body, _ := json.Marshal(EmbedPayload{Embeds: []Embed{embed}})
	resp, err := http.Post(webhook, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
