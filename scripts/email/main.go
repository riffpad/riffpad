// Command riffpad-email is the operator toolchain for Riffpad waitlist
// announcements: pull submissions from Formspree, render a template, send
// each recipient a personalized email over SMTP (implicit TLS), and embed a
// signed one-click unsubscribe link backed by the relay's email_optouts table.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "fetch":
		cmdFetch(os.Args[2:])
	case "send":
		cmdSend(os.Args[2:])
	case "token":
		cmdToken(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "riffpad-email: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `riffpad-email - Riffpad waitlist announcement toolchain

Commands:
  fetch    pull Formspree submissions into a recipients CSV (deduped)
  send     send a template email to every recipient with a signed
           unsubscribe link, skipping emails already opted out
  token    print the unsubscribe URL for one email (testing)

Examples:
  FORMSPREE_API_KEY=xxx FORMSPREE_FORM_ID=xjgqddar \
    riffpad-email fetch -out waitlist.csv

  SMTP_PASS=xxx UNSUBSCRIBE_SECRET=yyy \
    riffpad-email send -recipients waitlist.csv -template announcement.txt \
      -subject "Riffpad is live - try the beta"

Environment:
  SMTP_HOST              SMTP server (default mail.spacemail.com)
  SMTP_PORT              SMTP port (default 465, implicit TLS)
  SMTP_USER              SMTP login (default hi@riffpad.ai)
  SMTP_PASS              SMTP password
  FROM                   From address (default SMTP_USER)
  UNSUBSCRIBE_SECRET     HMAC secret shared with the relay (required)
  UNSUBSCRIBE_BASE_URL   unsubscribe page (default https://riffpad.ai/unsubscribe)
  RIFFPAD_API_URL        relay API (default https://api.riffpad.ai)
  WAITLIST_ADMIN_KEY     relay admin key used to fetch opt-outs
  FORMSPREE_API_KEY      Formspree API key (Professional/Business plans)
  FORMSPREE_FORM_ID      Formspree form hashid
`)
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func cmdToken(args []string) {
	fs := flag.NewFlagSet("token", flag.ExitOnError)
	email := fs.String("email", "", "email address")
	fs.Parse(args)
	if *email == "" {
		fatalf("token: -email is required")
	}
	u, err := unsubscribeURL(*email)
	if err != nil {
		fatalf("token: %v", err)
	}
	fmt.Println(u)
}

func cmdSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	recipients := fs.String("recipients", "", "recipients CSV (email[,name][,date])")
	template := fs.String("template", "", "email body template (text/template)")
	subject := fs.String("subject", "", "email subject")
	from := fs.String("from", "", "From address (default SMTP_USER)")
	dryRun := fs.Bool("dry-run", false, "render only, do not send")
	interval := fs.Duration("interval", 3*time.Second, "delay between sends")
	fs.Parse(args)

	if *recipients == "" || *template == "" || *subject == "" {
		fatalf("send: -recipients, -template and -subject are required")
	}
	if os.Getenv("UNSUBSCRIBE_SECRET") == "" {
		fatalf("send: UNSUBSCRIBE_SECRET is required")
	}

	recs, err := loadRecipients(*recipients)
	if err != nil {
		fatalf("send: %v", err)
	}
	if len(recs) == 0 {
		fatalf("send: no recipients in %s", *recipients)
	}

	optedOut := map[string]bool{}
	if key := os.Getenv("WAITLIST_ADMIN_KEY"); key != "" {
		emails, err := fetchOptOuts(key)
		if err != nil {
			fatalf("send: fetch opt-outs: %v", err)
		}
		for _, e := range emails {
			optedOut[normalizeEmail(e)] = true
		}
		fmt.Fprintf(os.Stderr, "riffpad-email: %d opted-out email(s) loaded from relay\n", len(optedOut))
	} else {
		fmt.Fprintln(os.Stderr, "riffpad-email: WAITLIST_ADMIN_KEY not set, skipping opt-out check")
	}

	bodyTmpl, err := loadTemplate(*template)
	if err != nil {
		fatalf("send: %v", err)
	}

	var sender *smtpSender
	if !*dryRun {
		sender = newSMTPSender(envOr("SMTP_HOST", "mail.spacemail.com"),
			envOrInt("SMTP_PORT", 465),
			envOr("SMTP_USER", "hi@riffpad.ai"),
			os.Getenv("SMTP_PASS"),
			*from)
		if sender.from == "" {
			sender.from = sender.user
		}
		if sender.pass == "" {
			fatalf("send: SMTP_PASS is required unless -dry-run")
		}
	}

	var sent, skipped, failed int
	for i, r := range recs {
		email := normalizeEmail(r.Email)
		if email == "" {
			skipped++
			continue
		}
		if optedOut[email] {
			fmt.Fprintf(os.Stderr, "  skip (opted out): %s\n", email)
			skipped++
			continue
		}
		u, err := unsubscribeURL(email)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  FAIL %s: %v\n", email, err)
			failed++
			continue
		}
		body, err := renderBody(bodyTmpl, r, u)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  FAIL %s: %v\n", email, err)
			failed++
			continue
		}
		if *dryRun {
			fmt.Printf("==> %s (%d/%d)\n%s\n\n", email, i+1, len(recs), body)
			sent++
			continue
		}
		if err := sender.send(email, *subject, body); err != nil {
			fmt.Fprintf(os.Stderr, "  FAIL %s: %v\n", email, err)
			failed++
			continue
		}
		fmt.Fprintf(os.Stderr, "  OK   %s\n", email)
		sent++
		if i < len(recs)-1 && *interval > 0 {
			time.Sleep(*interval)
		}
	}

	fmt.Fprintf(os.Stderr, "riffpad-email: done - sent=%d skipped=%d failed=%d\n", sent, skipped, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func cmdFetch(args []string) {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	formID := fs.String("form-id", "", "Formspree form hashid (or FORMSPREE_FORM_ID)")
	apiKey := fs.String("api-key", "", "Formspree API key (or FORMSPREE_API_KEY)")
	out := fs.String("out", "-", "output CSV path (default stdout)")
	fs.Parse(args)

	if *formID == "" {
		*formID = os.Getenv("FORMSPREE_FORM_ID")
	}
	if *apiKey == "" {
		*apiKey = os.Getenv("FORMSPREE_API_KEY")
	}
	if *formID == "" || *apiKey == "" {
		fatalf("fetch: -form-id and -api-key are required (or FORMSPREE_FORM_ID / FORMSPREE_API_KEY)")
	}
	subs, err := fetchFormspree(*formID, *apiKey)
	if err != nil {
		fatalf("fetch: %v", err)
	}
	if err := writeRecipientsCSV(*out, subs); err != nil {
		fatalf("fetch: %v", err)
	}
	fmt.Fprintf(os.Stderr, "riffpad-email: %d recipient(s) written to %s\n", len(subs), *out)
}

func envOrInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n <= 0 {
		return def
	}
	return n
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "riffpad-email: "+format+"\n", args...)
	os.Exit(1)
}
