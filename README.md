# brewbot

A Discord bot for homebrew clubs. Handles brew day scheduling, brewer rotation, recipe storage, ratings, and ABV calculation.

## Features

- **Scheduling** — Members propose dates, bot runs a reaction poll, majority vote auto-closes it and creates a brew channel
- **Rotation** — Tracks whose turn it is to brew, supports skipping with a reason
- **Recipes** — Submit and store recipes per brew; channel renames to the brew name on submit
- **Ratings** — Rate brews 1–5 with tasting notes after the session
- **Blackboard** — Archive of all past brews with ratings, styles, ABV
- **ABV Calculator** — Quick OG/FG → ABV utility

## Commands

| Command | Description |
|---|---|
| `/propose <dates>` | Propose one or more comma-separated dates |
| `/startpoll` | Start a reaction vote from all proposed dates |
| `/closepoll` | Manually close the poll (picks highest-voted) |
| `/rotation list` | Show the rotation order |
| `/rotation add @user` | Add a brewer to the rotation |
| `/rotation skip @user [reason]` | Skip someone this round |
| `/rotation next` | Show who is brewing next |
| `/recipe submit <name> [options]` | Submit recipe, renames the channel |
| `/recipe view` | View the recipe in this brew channel |
| `/rate <1-5> [notes]` | Rate the current brew |
| `/complete` | Mark brew done, archive to blackboard |
| `/abv <og> <fg>` | Calculate ABV and attenuation |
| `/blackboard` | Show all past brews and ratings |

## Typical Session Flow

```
1. Everyone runs /propose with their available dates
2. Someone runs /startpoll — bot posts a reaction vote
3. Members react — first option to hit majority auto-closes the poll
4. Bot creates a brew channel, pings the brewer, posts a pinned planning message
5. Brewer runs /recipe submit in their channel (channel renames to brew name)
6. After the session, everyone runs /rate
7. Brewer runs /complete to archive to the blackboard
```

## Tech Stack

- **Language**: Go
- **Discord library**: [discordgo](https://github.com/bwmarrin/discordgo)
- **Database**: SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (no CGO, no system deps)
- **Config**: `.env` file via [godotenv](https://github.com/joho/godotenv)

## Setup

### 1. Discord Application

1. Go to [discord.com/developers/applications](https://discord.com/developers/applications)
2. Create a new application → Bot tab → Add Bot → copy the token
3. Invite the bot to your server using the OAuth URL:

```
https://discord.com/oauth2/authorize?client_id=<APP_ID>&scope=bot+applications.commands&permissions=395137242192
```

Required permissions: Send Messages, Send in Threads, Create Public/Private Threads, Manage Threads, Manage Channels, Add Reactions, Pin Messages, Read Message History.

### 2. Configuration

Create a `.env` file in the project root:

```env
DISCORD_TOKEN=your-bot-token-here
DISCORD_APP_ID=your-application-id-here
DB_PATH=brewbot.db        # optional, defaults to brewbot.db
```

### 3. Build & Run

Requires Go 1.25+.

```bash
# Build
make build

# Run
./brewbot

# Or build + run in one step
make run
```

### 4. Help Channel

After first run, set up the `#brewbot-commands` channel by calling `SetupHelpChannel` or using the Discord API. The bot posts a full command reference automatically.

## Development

```bash
# Build with version stamp
make build

# Cut a release
make release TAG=v1.0.0

# Deploy a specific version
git checkout v1.0.0
make build
./brewbot

# Clean build artifacts
make clean
```

### Project Structure

```
brewbot/
├── main.go              # Entry point, signal handling
├── Makefile             # Build, release, clean targets
├── config/
│   └── config.go        # Env var loading
├── db/
│   └── db.go            # SQLite schema, migrations, all queries
└── bot/
    ├── bot.go           # Discord session, start/stop
    ├── commands.go      # Slash command definitions + registration
    ├── handlers.go      # Interaction router + reaction handler
    ├── autocomplete.go  # Autocomplete responses (e.g. /rate dropdown)
    ├── help.go          # Help channel setup
    ├── help.md          # Embedded command reference
    ├── propose.go       # /propose
    ├── poll.go          # /startpoll, /closepoll, majority detection
    ├── rotation.go      # /rotation subcommands
    ├── recipe.go        # /recipe submit + view
    ├── rate.go          # /rate, /complete
    ├── abv.go           # /abv
    └── blackboard.go    # /blackboard
```

### Database Schema

| Table | Purpose |
|---|---|
| `rotation` | Ordered brewer list with position and active flag |
| `proposed_dates` | Dates proposed since last poll |
| `polls` | Poll state, message ID, winning date |
| `poll_options` | Emoji→date mapping per poll |
| `brews` | Each brew session (brewer, date, channel, status) |
| `recipes` | Recipe details linked to a brew |
| `ratings` | Per-user ratings linked to a brew |

### Versioning

Versions follow [semver](https://semver.org). The version is embedded in the binary at build time via `-ldflags`:

```
v0.x — initial feature development
v1.0 — stable, all core features working
```

## Deployment (systemd)

See `brewbot.service` for a ready-to-use systemd unit file (coming soon).
