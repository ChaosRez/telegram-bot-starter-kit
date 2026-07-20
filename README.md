# Go Telegram Bot Starter Kit

A small, production-minded foundation for Telegram bots written in Go. Long
polling is the default transport and MongoDB is the primary database. The kit
uses the standard library where practical and keeps Telegram, persistence, and
business logic behind clear boundaries.

## Features

- Long polling with bounded, configurable workers
- Telegram Bot API 10 support through
  [`go-telegram/bot`](https://github.com/go-telegram/bot)
- MongoDB persistence through the official v2 Go driver
- Atomic user upserts and a unique 64-bit Telegram user ID index
- `/start`, `/help`, `/stop`, unknown-command, and callback-query examples
- Embedded English and German translations with English fallback
- Structured logging without message text, usernames, callback data, or tokens
- Strict YAML configuration plus environment-provided secrets
- Graceful `SIGINT` and `SIGTERM` shutdown
- Unit and MongoDB integration tests, race detection, `go vet`, vulnerability
  scanning, container builds, and Dependabot
- Non-root multi-stage container image and a local MongoDB Compose service

## Architecture

```text
Telegram long polling
        │
        ▼
recovery middleware + outcome logging
        │
        ▼
update dispatcher ───► command/callback handlers ───► Telegram API
        │
        ▼
users.Repository
        │
        ▼
MongoDB
```

Handlers depend on small interfaces rather than concrete network clients. This
makes them testable without a Telegram token or running MongoDB instance.

## Requirements

- Go 1.26.5 or later
- MongoDB 8.0 or later, or MongoDB Atlas
- A bot token from [@BotFather](https://t.me/BotFather)

## Quick start with Docker Compose

```sh
cp .env.example .env
```

Add your BotFather token to `.env`, then start both services:

```sh
docker compose up --build
```

Compose binds MongoDB only to `127.0.0.1` and stores its data in the
`mongo-data` volume.

## Run locally

Start MongoDB:

```sh
docker compose up -d mongo
```

Export the required configuration and run the bot:

```sh
export TELEGRAM_BOT_TOKEN='your-token'
export MONGODB_URI='mongodb://localhost:27017'
go run ./cmd/bot
```

Use another YAML configuration file with:

```sh
go run ./cmd/bot -config path/to/config.yaml
```

## Configuration

Non-secret and structured settings live in
[`config/config.yaml`](config/config.yaml). Unknown fields cause startup to
fail, which catches misspelled settings early.

| Environment variable | Required | Default | Purpose |
|---|---:|---|---|
| `TELEGRAM_BOT_TOKEN` | yes | — | BotFather API token |
| `MONGODB_URI` | no | `mongodb://localhost:27017` | MongoDB connection string |
| `MONGODB_DATABASE` | no | YAML value | Database override |
| `TELEGRAM_WORKERS` | no | YAML value | Bounded update-processing concurrency |
| `LOG_LEVEL` | no | YAML value | `debug`, `info`, `warn`, or `error` |
| `CONFIG_FILE` | no | `config/config.yaml` | Default YAML path |

`.env` is only a local convenience file and is not inherently more secure than
any other plaintext configuration file. Production should inject secrets from
its deployment platform or secret manager. Do not commit `.env` or place the
bot token in YAML.

The default worker count is `1`, preserving update order. Increase it only when
handlers are safe to run concurrently. Handlers are deliberately not launched
as unbounded goroutines.

## MongoDB data model

The `users` collection stores only the Telegram profile fields used by this
example:

```javascript
{
  telegram_id: NumberLong("1234567890123"),
  username: "example",
  first_name: "Example",
  last_name: "User",
  language_code: "en",
  created_at: ISODate("..."),
  updated_at: ISODate("..."),
  last_seen_at: ISODate("...")
}
```

`telegram_id` has a unique index. Telegram identifiers must be represented as
64-bit integers.

## Add a command

1. Add the command case and handler in
   [`internal/telegram/dispatcher.go`](internal/telegram/dispatcher.go).
2. Add translated messages under
   [`internal/localization/locales`](internal/localization/locales).
3. Add the command to `syncCommands` in
   [`internal/app/app.go`](internal/app/app.go) so Telegram clients display it.
4. Add a focused handler test.

Commands are parsed from Telegram's `bot_command` message entity instead of a
regular-expression prefix, including commands targeted as `/start@bot_name`.

## Development commands

```sh
make build       # build bin/telegram-bot
make test        # race-enabled tests with coverage
make test-integration  # repository test against MONGODB_URI
make vet         # standard Go static analysis
make fmt         # format source files
make verify      # formatting check, vet, and tests
make tidy        # normalize module dependencies
```

## Operational notes

- Long polling needs one active bot process. Running multiple pollers with the
  same token causes update conflicts.
- Startup fails when the bot already has a webhook configured because Telegram
  does not allow webhooks and long polling at the same time. Delete the webhook
  before starting this polling-based kit.
- A polling process may lose an in-flight update if it is forcefully killed.
  Use a durable webhook/queue design when processing must be at-least-once.
- Keep message contents out of logs unless your product has an explicit privacy
  and retention policy.
- Use an authenticated, TLS-protected MongoDB deployment in production. The
  unauthenticated Compose database is for local development only.
- Rotate the bot token immediately if it is exposed.

## Project layout

```text
cmd/bot/                    executable entrypoint
config/                     non-secret application configuration
internal/app/               dependency wiring and lifecycle
internal/config/            strict configuration loading
internal/database/          MongoDB client lifecycle
internal/localization/      embedded locale files
internal/telegram/          dispatcher, handlers, and middleware
internal/users/             domain repository and MongoDB implementation
```
