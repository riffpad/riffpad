---
version: "1.0"
name: Riffpad-design-system
description: |
  Riffpad's own "Console-Mobile" design language. A terminal console and a
  mobile remote control fused into one system: monospace-first type, warm
  paper / deep console color pair, brand-amber accents, square geometry,
  minimal glyphs (>, //, ●) and a hero that shows the product as a live
  session on your phone. Deliberately distinct from manpage-style terminal
  marketing: no cream-only manpage canvas, no ASCII bracket bullets, no
  prompt-string clutter in chrome, no 4px-only geometry.
---

# Riffpad Design System — Console-Mobile

## 1. Concept

Riffpad is the pocket remote for AI coding agents. The design language says
that in one sentence: **a terminal console and a phone, fused**.

- The page keeps a quiet terminal tone: `//` comment labels, status dots,
  and a console hero — without prompt-string chrome in the navigation.
- The hero is a two-device scene: a macOS terminal running
  Codex/Claude Code and the Riffpad phone app, syncing the same session.
  The terminal waits for approval; the phone shows the approval card — the
  product, not a metaphor.
- The brand color is Riffpad amber (`#F7A501`), used on marketing surfaces
  as the single warm accent against cool neutral ink.
- Geometry is square. Nothing floats; every card earns its place with a
  1px hairline and a very soft shadow.

This is *not* opencode's manpage aesthetic: no cream-only canvas, no ASCII
bracket bullets (`[+]` / `[-]`), no 4px-only radius, no Apple semantic ramp
locked inside a mockup, no `:~$` / `~/` prompt clutter in chrome. Terminal
flavor is kept through type, console surfaces, and a single product mockup
— never through decorative command strings.

## 2. Colors

### Light theme

| Token | Hex | Usage |
|---|---|---|
| `--canvas` | `#FAFAF7` | Page background (warm paper) |
| `--surface` | `#FFFFFF` | Cards, header, phone frame |
| `--surface-muted` | `#F2F1EC` | Hover fills, secondary blocks |
| `--ink` | `#191917` | Headlines, primary text |
| `--body` | `#46463F` | Paragraph text |
| `--mute` | `#73736A` | Metadata, captions |
| `--hairline` | `rgba(25,25,23,0.12)` | 1px borders/dividers |
| `--hairline-strong` | `#191917` | Focused/active borders |
| `--accent` | `#F7A501` | Brand CTA, highlights |
| `--accent-ink` | `#191917` | Text on accent fills |
| `--success` | `#16A34A` | Running / connected states |
| `--warning` | `#D97706` | Approval request states |
| `--danger` | `#DC2626` | Reject / destructive states |
| `--info` | `#2563EB` | Informational states |
| `--console` | `#111110` | Terminal/console surfaces |
| `--console-elevated` | `#1B1B19` | Console inner cards |
| `--on-console` | `#F5F4F0` | Text on console |
| `--on-console-mute` | `#9C9A92` | Secondary text on console |

### Dark theme

| Token | Hex | Usage |
|---|---|---|
| `--canvas` | `#0D0D0C` | Page background |
| `--surface` | `#151514` | Cards, header |
| `--surface-muted` | `#1D1D1B` | Hover fills |
| `--ink` | `#F5F4F0` | Headlines, primary text |
| `--body` | `#B9B7B0` | Paragraph text |
| `--mute` | `#8A887F` | Metadata, captions |
| `--hairline` | `rgba(245,244,240,0.10)` | 1px borders/dividers |
| `--hairline-strong` | `#F5F4F0` | Focused/active borders |
| `--accent` | `#FFB224` | Brand CTA, highlights |
| `--accent-ink` | `#191917` | Text on accent fills |
| `--success` | `#4ADE80` | Running / connected states |
| `--warning` | `#FBBF24` | Approval request states |
| `--danger` | `#F87171` | Reject / destructive states |
| `--info` | `#60A5FA` | Informational states |
| `--console` | `#080807` | Terminal/console surfaces |
| `--console-elevated` | `#151514` | Console inner cards |
| `--on-console` | `#F5F4F0` | Text on console |
| `--on-console-mute` | `#8A887F` | Secondary text on console |

Rules:

- Amber is the only marketing accent. Use it for the primary CTA, hero
  highlights, focus rings, and status dots. Never tint whole sections.
- Status colors (success/warning/danger/info) live inside console
  surfaces and approval UI, not on marketing chrome.
- Both themes must keep text contrast ≥ 4.5:1 (WCAG AA).

## 3. Typography

One family everywhere: **Geist Mono** (variable, 100–900), self-hosted.

| Token | Size | Weight | Line Height | Tracking | Use |
|---|---|---|---|---|---|
| `display-xl` | 40px (28px mobile) | 700 | 1.15 | -0.02em | Hero headline |
| `heading-lg` | 28px | 700 | 1.25 | -0.01em | Section titles |
| `heading-sm` | 18px | 700 | 1.4 | 0 | Card titles |
| `body-md` | 16px | 400 | 1.7 | 0 | Body copy |
| `body-strong` | 16px | 600 | 1.6 | 0 | Emphasis |
| `caption` | 13px | 400 | 1.6 | 0 | Metadata, notes |
| `label` | 12px | 700 | 1.4 | 0.12em | Uppercase section labels |
| `button` | 14px | 700 | 1.4 | 0 | Button labels |

CJK fallback is explicitly sans-serif: PingFang SC → Hiragino Sans GB →
Microsoft YaHei → Noto Sans CJK SC → WenQuanYi Micro Hei → sans-serif.
Never let CJK glyphs fall through to a serif face.

## 4. Spacing, Radius, Elevation

### Spacing

Base unit 4px: `xs 4` · `sm 8` · `md 12` · `lg 16` · `xl 24` · `xxl 32` ·
`section 96` (desktop), 64 (tablet), 48 (mobile).

### Radius

| Token | Value | Use |
|---|---|---|
| `sm` | 0px | Buttons, chips, inputs, console inner cards |
| `md` | 0px | Feature cards, console window, FAQ rows |
| `lg` | 0px | Phone mockup frame |
| `full` | 999px | Status dots, avatar dots |

### Elevation

- Every card: 1px `hairline` border + `0 1px 2px rgba(0,0,0,0.04)` and
  `0 8px 24px rgba(0,0,0,0.06)` in light; dark theme uses the hairline and
  a faint amber glow on the primary CTA only.
- No heavy shadows, no glassmorphism, no gradients on chrome. The console
  surface is the deepest layer on the page.

## 5. Iconography & Glyphs

No emoji, no icon library required. The system uses mono glyphs:

| Glyph | Meaning |
|---|---|
| `>` | Prompt / feature bullet |
| `//` | Comment / section label prefix |
| `●` | Live status dot |
| `01/02/03` | Step numbers |
| `→` | CTA arrow |

Use them sparingly, as text characters inside monospace type. No
bracket-wrapped controls (`[button]`) anywhere. If an SVG is truly needed
(logo/favicon), keep it minimal and line-based.

## 6. Components

### Top bar

- Height 56px, `surface` background, 1px `hairline` bottom rule.
- Left: Riffpad logo mark + `riffpad` wordmark.
- Center: plain nav links: Features / How it works / Security / FAQ.
- Right: text-only language toggle `EN / 中`, theme toggle `Dark / Light`,
  amber CTA.
- Mobile: nav collapses to a `menu` button; CTA stays visible.

### Hero

- Desktop: two columns. Left: `//` badge, display headline, description,
  primary (amber) + secondary (outline) CTAs, and a one-line trust note.
  Right: console window.

### Device scene (hero)

- Left: macOS-style terminal window (`console` surface, square corners,
  1px hairline): traffic-light dots, `codex — riffpad` title,
  `codex exec --json` prompt, tool-call lines with `✓` / `▸` / `!` states,
  and a daemon status line.
- Right: Riffpad phone app (`surface`, square corners, hairline): status
  bar, `riffpad` header with `● synced`, session card
  (`s_9f2a · claude · running · 2 tool calls`), approval card with
  Approve / Reject chips, message input, bottom tabs.
- Between them: a sync connector — hairline line, pulsing amber dot,
  `e2ee / synced / 84ms`.
- Both devices mirror the same session: the terminal waits for approval,
  the phone shows the approval card. This is the product story.

### Feature bento

- `// features` label, then a 2×2 grid of cards (`surface`, `md` radius,
  1px hairline).
- Each card: `>` glyph, bold title, body description.
- Hover: border turns `hairline-strong`, transition 200ms.

### How it works

- Three cards with large mono numbers `01 / 02 / 03`.
- Desktop: connected by a dashed 1px line through the row.
- Each card: number, title, description.

### Security console

- Left: `// security` label, title, description.
- Right: a console log card with colored lines:
  `[ok] e2ee aes-256-gcm`, `[ok] relay zero-knowledge`,
  `[ok] local-first`, `[warn] read-only by default`.

### FAQ

- Rows with `hairline` dividers, `md` radius on the whole block.
- Toggle marker: `>` when closed, `v` when open. Plain text, no chevron
  icons.

### CTA

- Centered block on `canvas`.
- Amber CTA button `Get early access →`, then a small note.

### Footer

- 1px `hairline` top rule, `surface` background.
- `riffpad` wordmark + tagline, GitHub / Docs / Contact links, copyright
  and `local-first · e2ee · zero-knowledge` line.

## 7. Motion

- All interactive transitions: 150–300ms, ease-out.
- Hover: card border shifts to `hairline-strong`; primary CTA darkens by
  one step; no scale transforms that shift layout.
- `prefers-reduced-motion: reduce` disables all transitions.

## 8. Do / Don't

### Do

- Keep monospace as the single voice; let weight and size do hierarchy.
- Use amber as the brand accent on marketing surfaces.
- Show the product as console + phone; the approval card is the hero.
- Use rounded cards with hairline borders and soft shadows.
- Use minimal `>` / `//` / `●` glyphs only where they aid scanning.
- Keep CJK text sans-serif.

### Don't

- Don't recreate opencode's manpage: no cream-only canvas, no ASCII bracket
  bullets, no 4px-only geometry, no Berkeley-Mono-only identity.
- Don't use prompt-string chrome (`:~$`, `~/`, `$`) in the header, footer,
  or CTA.
- Don't wrap buttons or controls in brackets (`[button]`).
- Don't use Apple semantic colors on marketing chrome.
- Don't add gradients, glassmorphism, terminal grids, or decorative
  imagery.
- Don't let the console surface appear more than once per viewport.
- Don't use emoji as icons.

## 9. Responsive

| Breakpoint | Behavior |
|---|---|
| ≥1024px | Two-column hero, 3-step row with dashed connector, bento 2×2 |
| 768–1023px | Hero stacks; bento stays 2×2 |
| <768px | Single column; hero display 28px; bento 1×1; nav collapses |

Touch targets ≥ 44px; focus rings visible in accent color; no horizontal
scroll.
