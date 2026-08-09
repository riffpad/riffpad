# Riffpad waitlist email toolchain

Operator tool for sending waitlist announcements with one-click
unsubscribe. Pure Go standard library (no runtime dependencies beyond a
binary build).

## Commands

```sh
# 1. Pull and dedupe waitlist emails from Formspree (Professional/Business
#    plan API; on the free plan export the CSV from the dashboard instead)
FORMSPREE_API_KEY=xxx FORMSPREE_FORM_ID=xjgqddar \
  go run . fetch -out waitlist.csv

# 2. Preview the rendered emails without sending
SMTP_PASS=xxx UNSUBSCRIBE_SECRET=yyy \
  go run . send -recipients waitlist.csv -template templates/announcement.txt.example \
    -subject "Riffpad is in beta" -dry-run

# 3. Send for real (Spacemail free tier: ~20 emails/hour, 50 recipients per
#    message; add -interval 180s to stay under the limit)
SMTP_PASS=xxx UNSUBSCRIBE_SECRET=yyy WAITLIST_ADMIN_KEY=zzz \
  go run . send -recipients waitlist.csv -template templates/announcement.txt.example \
    -subject "Riffpad is in beta" -interval 180s

# Helper: print the unsubscribe URL for one address
UNSUBSCRIBE_SECRET=yyy go run . token -email you@example.com
```

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `SMTP_HOST` | `mail.spacemail.com` | SMTP server (implicit TLS) |
| `SMTP_PORT` | `465` | SMTP port |
| `SMTP_USER` | `hi@riffpad.ai` | SMTP login |
| `SMTP_PASS` | — | SMTP password (required unless `--dry-run`) |
| `FROM` | `SMTP_USER` | From address |
| `FROM_NAME` | `Riffpad` | From display name shown in mail clients |
| `UNSUBSCRIBE_SECRET` | — | HMAC secret shared with the relay (required) |
| `UNSUBSCRIBE_BASE_URL` | `https://riffpad.ai/unsubscribe` | unsubscribe page |
| `RIFFPAD_API_URL` | `https://api.riffpad.ai` | relay base URL |
| `WAITLIST_ADMIN_KEY` | — | relay admin key for fetching opt-outs |
| `FORMSPREE_API_KEY` / `FORMSPREE_FORM_ID` | — | Formspree credentials for `fetch` |

## How unsubscribe works

Each email gets `https://riffpad.ai/unsubscribe?email=...&token=...` where
the token is `HMAC-SHA256(UNSUBSCRIBE_SECRET, email)`. The landing page
posts it to `POST /api/waitlist/unsubscribe` on the relay, which verifies
the signature and stores the address in the `email_optouts` table. The
`send` command fetches that list (`GET /api/waitlist/optouts` with
`X-Admin-Key`) and skips opted-out addresses.

If a template does not include `{{.UnsubscribeURL}}`, a footer with the
unsubscribe link is appended automatically so every sent message stays
compliant.
