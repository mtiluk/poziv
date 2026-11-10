# poziv

A small Go bot that watches eBay (UK) for new listings and posts them to a Discord channel via webhook.

It polls the eBay Browse API for each configured search, sorted by newest listings, and sends a Discord embed (title, price, image, link) for every item it hasn't seen before.

## How it works

- Each search in `config.json` is polled on its own interval.
- During optional quiet hours (for example 02:00 to 06:00 in your timezone), every interval is multiplied to reduce API usage overnight.
- Seen item IDs are stored in `data/seen.json` and expire after 30 days.
- The first time a search runs (or after its `query`, `minPrice` or `maxPrice` changes), its current listings are recorded without notifying, so you don't get flooded with existing items.
- If a Discord notification fails, the item is not marked as seen and will be retried on the next poll. On a Discord rate limit, the rest of that poll's notifications are deferred.
- The bot exits at startup if any required environment variable is missing.

## Requirements

- Go 1.27+
- An eBay developer application (App ID and Cert ID) with access to the Browse API
- A Discord webhook URL

## Setup

1. Copy the example env file and fill in your credentials:

   ```sh
   cp .env.example .env
   ```

   | Variable            | Description                                   |
   | ------------------- | --------------------------------------------- |
   | `AppID`             | eBay application (client) ID                  |
   | `CertID`            | eBay certificate (client secret) ID           |
   | `EbayURL`           | eBay API base URL, e.g. `https://api.ebay.com` |
   | `discordWebhookURL` | Discord webhook to post new listings to       |

2. Edit `config.json` to define what to watch and when to slow down.

3. Run it:

   ```sh
   go run ./cmd
   ```

   Or build a binary:

   ```sh
   go build -o poziv ./cmd
   ./poziv
   ```

   Run from the repository root, since `config.json` and `data/` are resolved relative to the working directory.

## Configuration

`config.json` holds the quiet-hours settings and the list of searches:

```json
{
  "quietHours": {
    "start": "02:00",
    "end": "06:00",
    "timezone": "Europe/London",
    "multiplier": 5
  },
  "searches": [
    {
      "name": "Wooting",
      "query": "Wooting",
      "minPrice": 0,
      "maxPrice": 0,
      "interval": "60s"
    },
    {
      "name": "MacBook Air M2",
      "query": "MacBook Air M2 16GB",
      "minPrice": 300,
      "maxPrice": 700,
      "interval": "2m"
    }
  ]
}
```

Restart the bot after changing `config.json`.

### Quiet hours

While the current time is inside the window, every search's interval is multiplied by `multiplier`. Leave out `quietHours` entirely to poll at the same rate all day.

| Field        | Required | Description                                                                                                    |
| ------------ | -------- | -------------------------------------------------------------------------------------------------------------- |
| `start`      | yes      | Start of the window, `HH:MM` in 24-hour time.                                                                  |
| `end`        | yes      | End of the window (exclusive), `HH:MM`. Can be earlier than `start` for windows that cross midnight, e.g. `22:00` to `06:00`. |
| `timezone`   | no       | [IANA timezone name](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) such as `Europe/London`, `America/New_York` or `Asia/Tokyo`. Daylight saving is handled automatically. Defaults to `UTC`. |
| `multiplier` | yes      | Whole number of 1 or more to multiply intervals by during the window.                                         |

### Searches

| Field      | Required | Description                                                                                       |
| ---------- | -------- | ------------------------------------------------------------------------------------------------- |
| `name`     | no       | Label shown in the Discord embed. Defaults to `query`.                                            |
| `query`    | yes      | Free-text eBay keyword search. Add specs (RAM, size, etc.) as extra keywords.                     |
| `minPrice` | no       | Minimum item price in GBP. `0` means no lower bound. Postage is not included.                     |
| `maxPrice` | no       | Maximum item price in GBP. `0` means no upper bound. Postage is not included.                     |
| `interval` | yes      | How often to poll, as a Go duration string (`30s`, `5m`, `1h`).                                   |

## Testing

```sh
go test ./...
```

## Project layout

```
cmd/main.go               entry point and polling loop
cmd/settings.go           config.json loading and quiet-hours logic
internal/config           loads credentials from .env / environment
internal/ebay             eBay Browse API client with OAuth token caching
internal/discord          Discord webhook embed sender
internal/seen             persistent, TTL-pruned store of seen item IDs
config.json               quiet hours and searches to watch
data/seen.json            runtime state (git-ignored)
```

## License

[MIT](LICENSE)
