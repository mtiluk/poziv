package config

import (
	"encoding/base64"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	AppID             string
	CertID            string
	EbayURL           string
	BasicCreds        string
	DiscordWebhookURL string
}

var (
	once sync.Once
	cfg  *Config
)

func GetConfig() *Config {
	once.Do(func() {
		_ = godotenv.Load()

		appID := os.Getenv("AppID")
		certID := os.Getenv("CertID")

		cfg = &Config{
			AppID:             appID,
			CertID:            certID,
			EbayURL:           os.Getenv("EbayURL"),
			BasicCreds:        basicCreds(appID, certID),
			DiscordWebhookURL: os.Getenv("discordWebhookURL"),
		}
	})
	return cfg
}

func basicCreds(appID, certID string) string {
	return base64.StdEncoding.EncodeToString([]byte(appID + ":" + certID))
}
