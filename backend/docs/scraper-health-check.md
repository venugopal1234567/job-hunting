# Scraper Weekly Health Check

AI agent checklist. Run every Monday. Goal: catch broken scrapers before they go stale.

## 1. Quick Status Overview

**One-liner check** — DB row freshness by board (target: ≤ 2 days stale):

```bash
docker exec remotehunter-db psql -U hunter -d remotehunter -c "
SELECT board_name, COUNT(*) as jobs, MAX(scraped_at) as last_scrape FROM jobs GROUP BY board_name ORDER BY last_scrape ASC;
"
```

**Red flags** (`last_scrape` > 3 days old):
- `googlejobs`, `googlejobscompanylist` — stale until SerpAPI quota renews (Oct 4) or proxy added
- Any board showing 0 rows or unchanged count week-over-week

## 2. Per-Board Verification

### Stable Boards (expect fresh data)
| Board | Source | What to check |
|-------|--------|--------------|
| `hnhiring` | HN Algolia API | 10-15 rows, newest ≤ 7 days |
| `weworkremotely` | weworkremotely.com RSS | 80-110 rows, newest ≤ 7 days |
| `remotive` | remotive.io API (free) | ≥ 40 rows, status 200 |
| `remoteok` | remoteok.com API | ≥ 50 rows |
| `arbeitnow` | arbeitnow.com API | ≥ 30 rows (monitor upstream — can return 0 sometimes) |

### Quota/Blocked Boards (expect intermittent staleness)
| Board | Known issues | Expected state |
|-------|-------------|----------------|
| `googlejobs` | SerpAPI quota + Google CAPTCHA | Stale until Oct 4 or proxy added |
| `googlejobscompanylist` | Same as above | Same as above |
| `flexboard` | Cloudflare challenge | Currently broken — monitor upstream |
| `ziprecruiter` | User said ignore for now | Not monitored |

## 3. SerpAPI Quota Check (GoogleJobs only)

Check before next Oct 4:

```bash
# Verify SerpAPI live status
curl -s "https://serpapi.com/search.json?engine=google_jobs&q=test&api_key=$SERPAPI_API_KEY" | head -c 200
# If "error: Your account has run out of searches" → still blocked
# If JSON job results → quota restored, remove the 5-calls/day cap
```

**When quota restores:** remove `googleJobsMaxCallsPerDay` logic from `scheduler.go` and run a manual scrape to backfill.

## 4. Container & Logs

```bash
# Restart backend (picks up any code changes)
docker restart remotehunter-backend

# Watch live scrape activity
docker logs -f remotehunter-backend 2>&1 | grep -E "(Scheduler|Scraper).*saved|error|Skipping|captcha|0 jobs"
```

Expected log patterns:
- `[Scheduler] Running scraper 'X'` — all boards except googlejobs after cap
- `[Scheduler] Skipping 'GoogleJobs' — daily SerpAPI cap (5) reached`
- `[Scraper] GoogleJobs: SerpAPI returned status 429 — falling back to chromedp`
- `[Scraper] GoogleJobs: parsed 0 jobs` (CAPTCHA)

## 5. Manual Trigger Test (if something looks off)

```bash
# Force a single board scrape via curl
curl -X POST http://localhost:8080/api/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"board": "hnhiring", "target_url": ""}'
```

## 6. Fix Checklist

When a board is stale or 0-row:

1. **Check logs** — `docker logs remotehunter-backend | grep <board>`
2. **Check source** — is the upstream API/HTML still valid?
3. **Check SerpAPI** — if googlejobs, is quota exhausted?
4. **Check DB** — `SELECT * FROM scraper_configs WHERE board_name = '<Board>'` — is it enabled?
5. **Code change?** — Fix + commit + `git push` + `docker restart remotehunter-backend`
6. **Verify** — Re-run DB row-count query, confirm fresh rows appear

## 7. Weekly Action Items

- [ ] Run freshness query — confirm no red flags
- [ ] Check SerpAPI quota status (googlejobs only)
- [ ] Spot-check 2-3 boards' latest rows in DB
- [ ] Review any new scraper errors in logs (last 24h)
- [ ] If googlejobs still blocked, confirm no proxy added

## 8. Emergency Reset (if rate limiter hangs)

If the 5-calls/day counter seems stuck:

```bash
docker restart remotehunter-backend
# Scheduler resets in-memory counters on restart
```
