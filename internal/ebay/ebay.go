package ebay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMarketplace = "EBAY_GB"
	tokenEndpoint      = "https://api.ebay.com/identity/v1/oauth2/token"
	browseScope        = "https://api.ebay.com/oauth/api_scope"
)

type Item struct {
	ItemID string `json:"itemId"`
	Title  string `json:"title"`
	Price  struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	} `json:"price"`
	ItemWebURL string `json:"itemWebUrl"`
	Image      struct {
		ImageURL string `json:"imageUrl"`
	} `json:"image"`
}

type SearchResponse struct {
	Total         int    `json:"total"`
	Href          string `json:"href"`
	Next          string `json:"next"`
	ItemSummaries []Item `json:"itemSummaries"`
}

type SearchParams struct {
	Query    string
	Limit    int
	Offset   int
	Sort     string
	Filter   string
	Category string
	Aspects  []string
}

func (p SearchParams) values() url.Values {
	v := url.Values{}
	if p.Query != "" {
		v.Set("q", p.Query)
	}
	if p.Limit > 0 {
		v.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Offset > 0 {
		v.Set("offset", strconv.Itoa(p.Offset))
	}
	if p.Sort != "" {
		v.Set("sort", p.Sort)
	}
	if p.Filter != "" {
		v.Set("filter", p.Filter)
	}
	if p.Category != "" {
		v.Set("category_ids", p.Category)
	}
	if len(p.Aspects) > 0 {
		v.Set("aspect_filter", strings.Join(p.Aspects, ","))
	}
	return v
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ebay: status %d: %s", e.StatusCode, e.Body)
}

type Client struct {
	BaseURL     string
	Marketplace string
	HTTP        *http.Client

	creds string

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func NewClient(baseURL, creds string) *Client {
	return &Client{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		Marketplace: defaultMarketplace,
		HTTP:        &http.Client{Timeout: 15 * time.Second},
		creds:       creds,
	}
}

func (c *Client) Search(ctx context.Context, p SearchParams) (SearchResponse, error) {
	var sr SearchResponse

	token, err := c.Token(ctx)
	if err != nil {
		return sr, err
	}

	endpoint := c.BaseURL + "/buy/browse/v1/item_summary/search?" + p.values().Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return sr, fmt.Errorf("build search request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-EBAY-C-MARKETPLACE-ID", c.Marketplace)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return sr, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		c.Invalidate()
	}
	if resp.StatusCode != http.StatusOK {
		return sr, readAPIError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return sr, fmt.Errorf("decode search response: %w", err)
	}
	return sr, nil
}

func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.expiresAt) {
		return c.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", browseScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+c.creds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", readAPIError(resp)
	}

	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("ebay: empty access token")
	}

	c.token = tr.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn)*time.Second - time.Minute)

	return c.token, nil
}

func (c *Client) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.expiresAt = time.Time{}
}

func readAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
}
