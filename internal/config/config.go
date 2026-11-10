package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	EbayURL           string
	BasicCreds        string
	DiscordWebhookURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	vars := map[string]string{}
	var missing []string
	for _, name := range []string{"AppID", "CertID", "EbayURL", "discordWebhookURL"} {
		v := strings.TrimSpace(os.Getenv(name))
		if v == "" {
			missing = append(missing, name)
		}
		vars[name] = v
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return &Config{
		EbayURL:           vars["EbayURL"],
		BasicCreds:        basicCreds(vars["AppID"], vars["CertID"]),
		DiscordWebhookURL: vars["discordWebhookURL"],
	}, nil
}

func basicCreds(appID, certID string) string {
	return base64.StdEncoding.EncodeToString([]byte(appID + ":" + certID))
}
