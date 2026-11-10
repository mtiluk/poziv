package discord

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrRateLimited = errors.New("discord: rate limited")

var httpClient = &http.Client{Timeout: 15 * time.Second}

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

func SendEmbed(webhook string, searchName string, title string, price string, url string, imageURL string) error {
	embed := Embed{
		Title:       title,
		URL:         url,
		Description: "A new item has been added to eBay.",
		Color:       5763719,
		Fields: []EmbedField{
			{Name: "Price", Value: price, Inline: true},
			{Name: "Search", Value: searchName, Inline: true}},
		Footer:    &EmbedFooter{Text: "Monitoring Agent"},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if imageURL != "" {
		embed.Image = &EmbedImage{URL: imageURL}
	}

	body, err := json.Marshal(EmbedPayload{Embeds: []Embed{embed}})
	if err != nil {
		return fmt.Errorf("discord: encode payload: %w", err)
	}
	resp, err := httpClient.Post(webhook, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("%w (retry after %ss)", ErrRateLimited, resp.Header.Get("Retry-After"))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("discord: status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}
