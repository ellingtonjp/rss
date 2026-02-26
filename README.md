🖐 I'm completely vibe coded 🤖

# rss-generator

A Go web app that generates RSS feeds from any website by scraping with CSS selectors, or by proxying existing RSS feeds. Feeds are stored in SQLite and refreshed on a configurable schedule.

## Usage

```
RSS_API_KEY=<secret> go run .
```

Server runs on `:8080`. The web UI lets you create/preview/edit feeds. RSS endpoints and OPML export require `?key=<RSS_API_KEY>`.

## Security

It's vibe coded and I haven't looked at the code so there are likely lots of vulnerabilities. It's OK with me due to how it's hosted in my homelab and protected at the perimeter.

Claude gave me the following security review.

```
  Critical/High                                                                                                                                                                                                                                                                                                       
                                                                                                                                                                                                                                                                                                                      
  #2 — API key embedded in OPML (handlers.go:362)                                                                                                                                                                                                                                                                     
  The key is baked into every xmlUrl in the OPML. If you import that into a cloud-based RSS reader (Feedly, Inoreader, etc.), the key is stored in their system. Consider using a separate read-only token for feed consumption.                                                                                      
                                                                                                                                                                                                                                                                                                                      
  Medium

  #3 — API key in query string — will appear in Caddy access logs, Referer headers, etc. A path-based token (/feeds/{token}/1/rss) would be safer.

  #4 — SSRF (fetcher.go:28) — http.Get(sourceURL) fetches any URL with no restrictions. A scraped feed could target 169.254.169.254 or internal services. Mitigated by VPN-only web UI, but stored URLs are re-fetched by the scheduler.

  #5 — No CSRF protection — A malicious page could auto-submit forms to create feeds (injecting SSRF URLs) or delete feeds if the user is on VPN.

  #7 — Host header injection (handlers.go:362) — r.Host is used directly in OPML output. A spoofed Host header would generate URLs pointing to an attacker's domain with the real API key.

  #9 — Unbounded response body (fetcher.go:28) — No size limit on fetched pages. A malicious URL returning gigabytes will OOM the server. Fix: io.LimitReader(resp.Body, 10<<20).

  #10 — No HTTP client timeout (fetcher.go:28) — http.DefaultClient has no timeout. A slow-responding server hangs the scheduler forever. Fix: &http.Client{Timeout: 30 * time.Second}.

  Low

  #8 — OPML link broken (feeds.html:5) — The "Export OPML" link doesn't include ?key=, so it always returns 401.

  #12 — Error messages leak internals — err.Error() returned to clients on internet-exposed endpoints.

  #14 — OPML uses http:// not https:// — API key sent in plaintext over the wire.

  What's done well

  - All SQL queries use parameterized ? placeholders — no SQL injection
  - html/template auto-escapes output — no XSS
  - crypto/rand for key generation
  - .env in .gitignore
  - No command execution, no path traversal
```
