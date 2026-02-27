# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the server (requires RSS_API_KEY in .env or environment)
go run .

# Build binary
go build -o rss-generator .

# Run with explicit key
RSS_API_KEY=mysecret go run .
```

There are no tests in this codebase.

## Architecture

Single-package Go web app (`package main`) with SQLite persistence via `feeds.db`.

**Feed types:**
- `scrape` — fetches a URL, extracts items with CSS selectors (via goquery), and caches the generated RSS XML in SQLite
- `rss` — proxies an existing RSS feed URL directly (no scraping; scheduler skips these)

**Data flow for scrape feeds:**
1. User submits CSS selectors via the web UI → `handlePreviewTest` calls `FetchAndParse` → returns live preview
2. On save (`handleCreateFeed`/`handleUpdateFeed`) → `FetchAndParse` + `GenerateRSS` → stored in `feeds.cached_rss`
3. Scheduler (`scheduler.go`) ticks every minute, queries `FeedsDueForRefresh()` (uses SQLite datetime arithmetic on `last_refreshed + refresh_minutes`), and re-runs fetch+generate+cache for each due feed
4. RSS consumers hit `GET /feeds/{id}/rss?key=<RSS_API_KEY>` → returns `cached_rss` directly

**Key files:**
- `main.go` — HTTP mux, API key auth middleware, server startup
- `db.go` — `Feed` struct, all SQL (CREATE, CRUD, `FeedsDueForRefresh`); migrations are additive `ALTER TABLE` statements that silently fail if the column exists
- `fetcher.go` — `FetchAndParse` (HTTP GET + goquery), `ExtractItems` (CSS selector extraction with optional regex capture groups via `applyRegex`), relative URL resolution
- `rss.go` — `GenerateRSS` (gorilla/feeds), multi-format date parser
- `scheduler.go` — goroutine ticker calling `refreshDueFeeds`
- `handlers.go` — all HTTP handlers, OPML export
- `env.go` — `.env` file loader (only sets vars not already in environment)
- `templates/` — `html/template` files; `layout.html` wraps all full pages; `preview_results.html` is a partial (no layout) returned via HTMX
- `static/style.css` — served under `/static/`

**Auth:** `RSS_API_KEY` (from `.env` or env var) is required as `?key=` query param on `/feeds/{id}/rss` and `/feeds/opml` endpoints. The web UI itself has no authentication.

**Regex extraction:** Each selector field has an optional companion regex field. If provided, `applyRegex` runs it against the extracted text — if there's a capture group, group 1 is returned; otherwise the full match is returned.
