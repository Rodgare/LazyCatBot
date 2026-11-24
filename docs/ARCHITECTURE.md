# LazyCatBot — architecture map

Detailed internals. Read on demand; `AGENTS.md` is the always-loaded summary.
Source of truth is the code — update this file when structure changes.

## 1. Startup (`cmd/bot/main.go`)
1. `godotenv.Load()` → `.env`; logs to `bot.log` + Axiom (`slog` multi-handler); debug adds console.
2. `config.LoadConfig()` builds `*config.Config` from env.
3. `sql.Open("sqlite", "bot.db")`; `PRAGMA journal_mode=WAL`, `busy_timeout=10000`, `SetMaxOpenConns(1)`.
4. goose applies embedded migrations (`internal/storage/migrations`) with `goose.Up`.
5. Construct stores, `sirus.NewClient`, `discord.NewDiscordReporter`, `worker.NewWorker`; wire optional stores (`SetMythicLeaderboardStore`, `SetPremiumGate`, `SetGuildResolver`, `SetKillHistoryStore`).
6. Load channel→guild ids from `subStore.GetChannelDiscordIDs()` into the reporter.
7. Start cron scheduler + goroutines (see §4); start Tribute webhook if keys set.
8. Register slash commands (`ApplicationCommandBulkOverwrite`) and open the Discord gateway.

Shutdown on SIGINT/SIGTERM cancels `ctx`, sleeps 5s.

## 2. Packages
| Package | Responsibility | Key files |
|---|---|---|
| `internal/config` | env → `Config`; realm list helpers | `config.go` |
| `internal/sirus` | HTTP scraping/parsing of sirus.su | `client.go`, `raids.go`, `helpers.go`, `itemsets.go` |
| `internal/models` | API DTOs + report structs | `bossfight.go`, `boss_kill_report.go`, `mythic.go`, `leaderboard.go`, `digest.go`, `player_profile.go` |
| `internal/storage` | SQLite repos, one type per domain | `leaderboard.go`, `subscribe.go`, `guild_members.go`, `premium.go`, `kill_history.go`, `character_cache.go`, `meta.go` |
| `internal/storage/migrations` | embedded goose SQL + `EmbedFS` | `migrations.go`, `00001..00017_*.sql` |
| `internal/worker` | business logic & scheduling | `killmonitor.go`, `queue.go`, `leaderboard.go`, `mythic_leaderboard.go`, `guild_members.go`, `character_cache.go`, `digest.go`, `announce.go`, `cleanup.go`, `ratelimit.go` |
| `internal/discord` | handlers, embeds, PNG rendering | `handlers.go`, `reporter.go`, `renderer.go`, `image*.go`, `ranking.go`, `premium.go`, `donors.go`, `limits.go`, `analytics.go` |
| `internal/webhook` | Tribute webhook → premium | `tribute.go` |
| `internal/news` | embedded announcement, hash-versioned | `news.go`, `news.md` |
| `cmd/genmythicbg` | generates mythic background PNG | `main.go` |

## 3. Data flow
```
sirus.su pages ──> internal/sirus (parse) ──> internal/models
        │                                          │
        ▼                                          ▼
   worker monitors  ──> job queue (premium-first) ──> discord.Reporter
        │                                          │
        ▼                                          ▼
   internal/storage (SQLite)                 embed + gg PNG image
```
- Kill/leaderboard data is scraped from sirus HTML/JSON; parsing is the main
  latency source (paged requests with sleeps).
- `worker` decides what to post and to which channel, consulting `PremiumGate`
  and free limits; `discord.Reporter` sends/edits messages and renders images.
- `itemsets.go` lazily loads `itemsets.json` once to map item set id → tier.

## 4. Worker goroutines
Started from `main.go`:
- `StartProcessor(ctx)` — single consumer of the job queue (300ms spacing).
- `GuildKillMonitor`, `PlayerKillMonitor`, `MythicRunsMonitor` — poll Sirus for new kills/runs.
- `GuildMembersUpdater`, `CharacterCacheUpdater` — refresh rosters/character data.
- `CleanupRoutine` — prune processed kills / stale rows.
- `StartLeaderboardSyncOnStart` (only if `SyncOnStart`), `StartMythicLeaderboardSync`.
- `StartCronScheduler` — scheduled syncs (`robfig/cron`).
- `AnnounceOnce(ctx)` — deliver news once per version.

`queue.go`: two FIFO queues (premium, free). Premium drains first; a free job
older than `freeJobMaxWait` (45s) is forced next to prevent starvation.
`ratelimit.go` throttles Sirus requests.

## 5. Discord commands (registered in `main.go`)
`/menu`, `/top`, `/topm`, `/premium`, `/attendance`, `/digest`, `/panel`,
`/profile`, `/help`, `/list`, `/listcats`, `/removechannel`, `/resync`,
`/grantpremium`. Handlers in `discord/handlers.go` (`InteractionCreate`);
`GuildCreate` auto-setup is skipped in debug mode.

## 6. Storage / migrations
- One SQLite file `bot.db`, single connection (serialized).
- Repos: `LeaderboardStorage`, `SubscribeStorage`, `GuildMembersStorage`,
  `PlayerSubscribeStorage`, `CharacterCacheStorage`, `ActualRaidsStorage`,
  `MythicLeaderboardStorage`, `PremiumStore`, `KillHistoryStorage`.
- Migrations `00001_init` → `00017_bot_meta`; **append-only**. Exact schema lives
  in the `.sql` files — read those, not memory. Notable: realm columns, mythic
  leaderboard, character cache, donors, premium + guild scope, leaderboard/kill
  history, `bot_meta` key-value.
- `leaderboard` is a snapshot (DELETE+INSERT per combo); historical ranges use
  dedicated history tables, not the live snapshot.

## 7. Config (env, `internal/config/config.go`)
| Env | Meaning / default |
|---|---|
| `DISCORD_TOKEN` | bot token (required) |
| `DEBUG` | `"true"` enables debug + console logs; disables `GuildCreate` |
| `DEBUG_CHANNEL_ID`, `ADMIN_USER_ID` | debug channel, admin commands |
| `SIRUS_BASE_URLS` | csv, default `https://sirus.su` |
| `SIRUS_REALM` | default realm, `x3` |
| `SIRUS_REALMS` | tracked realms, default `x3,x5` |
| `SIRUS_SECONDARY_REALMS` | realms parsed for current raids only, default `x5,x3` |
| `SYNC_LEADERBOARD_ON_START` | **inverted**: `"false"` → `SyncOnStart=true` |
| `SYNC_LEADERBOARD_REALMS`, `SYNC_LEADERBOARD_FORCE` | sync scope/force |
| `SIRUS_MYTHIC_SEASON`, `SIRUS_MYTHIC_WEEK_ID` | mythic window |
| `PREMIUM_URL` / `DONATE_URL` | premium purchase link |
| `PREMIUM_MONTHLY_USD`, `FREE_GUILD_LIMIT`, `FREE_PLAYER_LIMIT` | pricing/limits |
| `TRIBUTE_API_KEY`, `TRIBUTE_WEBHOOK_ADDR`, `TRIBUTE_WEBHOOK_PATH` | Tribute webhook (disabled if unset) |
| `AXIOM_TOKEN`, `AXIOM_DATASET` | Axiom log sink |

## 8. Premium & payments
- Premium is **guild-wide**: channel → guild resolved via `ChannelGuildResolver`,
  then `PremiumGate.IsPremium(guildID)`.
- `internal/webhook/tribute.go` verifies `trbt-signature` (HMAC-SHA256), parses
  the donation code, and activates/extends premium in `PremiumStore`.

## 9. Rendering
`discord/image*.go` build PNGs with `fogleman/gg` using fonts in
`internal/discord/assets/fonts` and class/spec/raid images in
`internal/discord/assets/images`. Avoid reading those asset dirs.

## 10. Where to look
- New slash command → `internal/discord/handlers.go` + register in `cmd/bot/main.go`.
- New DB field → new migration + repo in `internal/storage`.
- New scheduled sync → `internal/worker` + schedule in `StartCronScheduler`.
- Report text/layout → `discord/reporter.go` (embeds) or `discord/image_*.go`.
- Scraping change → `internal/sirus` (keep `ratelimit.go` throttling).
