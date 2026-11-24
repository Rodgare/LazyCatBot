---
description: Refresh docs/ARCHITECTURE.md from the current code
---
Re-scan the repository (packages under `internal/`, `cmd/`, migrations, registered slash commands, env vars in `internal/config`) and update `docs/ARCHITECTURE.md` to match reality. Keep it under ~200 lines, use tables and `file_path:line` references, and do not include code bodies. Do not read `itemsets.json`, databases, binaries, logs, images, `.env`, or `.kilo/worktrees`.
