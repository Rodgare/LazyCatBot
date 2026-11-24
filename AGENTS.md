# LazyCatBot — agent guide

Discord bot (Go 1.26) for the Sirus WoW server. Tracks boss kills / mythic runs
for guilds and players, posts kill reports, DPS/HPS leaderboards, attendance and
weekly digests (as embeds + rendered PNG images), and sells guild-wide premium
via Tribute payments. Entry point: `cmd/bot/main.go`.

## Build / test / run
- Build: `go build ./...` (or `make build` → `bot.exe`)
- Test: `go test ./...` (or `make test`)
- Vet / format: `go vet ./...`, `gofmt -l .`
- Run locally: needs `.env` (at least `DISCORD_TOKEN`), then `make run`. DB
  migrations run automatically at startup (goose, embedded SQL).

## Package map
- `internal/config` — env config via `LoadConfig()` (godotenv). See env table in `docs/ARCHITECTURE.md`.
- `internal/sirus` — HTTP client that scrapes/parses sirus.su (leaderboards, raids, guild kills, players).
- `internal/models` — API DTOs and report structs (boss kill, mythic, digest, leaderboard, profile...).
- `internal/storage` — SQLite repositories (modernc.org/sqlite) + embedded goose migrations in `storage/migrations/`.
- `internal/worker` — business logic: kill monitors, leaderboard/mythic sync, guild-members updater, character cache, digest, announce, cleanup; premium-first job queue (`queue.go`), rate limiting.
- `internal/discord` — discordgo slash-command handlers, embeds reporter, gg PNG rendering (`image_*.go`), ranking, premium, donors, limits.
- `internal/webhook` — Tribute payment webhook (HMAC `trbt-signature`) → premium activation.
- `internal/news` — embedded `news.md` announcement, versioned by content hash.
- `cmd/genmythicbg` — utility that generates the mythic background image.

## Flow
`main` wires: stores → `sirus.Client` → `worker.Worker` → `discord.DiscordReporter`.
Worker goroutines (`StartProcessor`, `GuildKillMonitor`, `PlayerKillMonitor`,
`MythicRunsMonitor`, `GuildMembersUpdater`, `CharacterCacheUpdater`,
`CleanupRoutine`, leaderboard/mythic sync) run concurrently; slash commands arrive
through `discord.Handler.InteractionCreate`.

## Conventions & gotchas
- Migrations: numbered `NNNNN_name.sql`, embedded. Add new files; never edit applied ones.
- SQLite runs with `SetMaxOpenConns(1)` + WAL — all queries serialize. Keep DB work short; long Sirus page parsing is the usual bottleneck.
- Job queue is premium-first; free jobs are bounded by `freeJobMaxWait` (45s) so premium traffic cannot starve free channels.
- Premium is guild-wide: channel → guild is resolved before checking entitlement.
- Realms are `x3`/`x5`; "secondary" realms only parse current raids.
- Gotcha: `SyncOnStart` is set from `SYNC_LEADERBOARD_ON_START == "false"` (inverted name).
- All user-facing bot text is Russian; keep it Russian.
- Report images use `fogleman/gg` with fonts in `internal/discord/assets/fonts`.

## Do NOT read (huge / generated / secret)
- `itemsets.json` (~2.6 MB) — query it with `grep`/`jq` for a specific key, never load it whole.
- `bot.db`, `bot.db-shm`, `bot.db-wal`, `*.exe`, `bot_linux`, `*.log`, `leaderboard.json`
- `img/**`, `internal/discord/assets/**/*.png|*.ttf`
- `.kilo/worktrees/**` — duplicate full checkouts of this repo
- `.env` — secrets; never print values

## Docs
- `docs/ARCHITECTURE.md` — deeper map (read when you need internals).
- `plan.md` — premium-redesign notes and DB rationale.
- `README.md` — user-facing docs.
