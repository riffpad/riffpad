---
version: "1.0"
name: Riffpad-design-system
description: |
  Riffpad's own "Console-Mobile" design language. A terminal console and a
  mobile remote control fused into one system: monospace-first type,
  Catppuccin Latte / Mocha surfaces, mauve accents, square geometry, flat
  panes with hairline rules, minimal glyphs (>, //, ●) and a hero that shows
  the product as a live session on your phone. Deliberately distinct from
  manpage-style terminal marketing: no cream-only manpage canvas, no ASCII
  bracket bullets, no prompt-string clutter in chrome, no 4px-only geometry.
---

# Riffpad Design System — Console-Mobile

## 1. Concept

Riffpad lets you watch, approve, and steer AI coding agents from your phone. The design language says
that in one sentence: **a terminal console and a phone, fused**.

- The page keeps a quiet terminal tone: `//` comment labels, status dots,
  and a console hero — without prompt-string chrome in the navigation.
- The hero is a two-device scene: a macOS terminal running
  Codex/Claude Code and the Riffpad phone app, syncing the same session.
  The terminal waits for approval; the phone shows the approval card — the
  product, not a metaphor.
- The brand accent is Catppuccin mauve (`#8839EF` on Latte, `#CBA6F7` on
  Mocha), used on marketing surfaces as the single accent against the
  neutral ramp. Green/peach/red/blue are reserved for status.
- Geometry is square. Nothing floats; every card earns its place with a
  1px hairline and a very soft shadow.

This is *not* opencode's manpage aesthetic: no cream-only canvas, no ASCII
bracket bullets (`[+]` / `[-]`), no 4px-only radius, no Apple semantic ramp
locked inside a mockup, no `:~$` / `~/` prompt clutter in chrome. Terminal
flavor is kept through type, console surfaces, and a single product mockup
— never through decorative command strings.

## 2. Colors

### Light theme — Catppuccin Latte

| Token | Hex | Usage |
|---|---|---|
| `--canvas` | `#E6E9EF` | Page background (Latte mantle) |
| `--surface` | `#EFF1F5` | Cards, header, phone frame (Latte base) |
| `--surface-muted` | `#DCE0E8` | Hover fills, secondary blocks (Latte crust) |
| `--ink` | `#4C4F69` | Headlines, primary text (Latte text) |
| `--body` | `#5C5F77` | Paragraph text (Latte subtext1) |
| `--mute` | `#5C5F77` | Metadata, captions (Latte subtext1 — one step darker than subtext0 so 12px labels stay ≥4.5:1 on the mantle canvas) |
| `--hairline` | `rgba(76,79,105,0.16)` | 1px borders/dividers |
| `--hairline-strong` | `rgba(76,79,105,0.44)` | Focused/active borders |
| `--accent` | `#8839EF` | Brand CTA, highlights (Latte mauve) |
| `--accent-hover` | `#7287FD` | CTA hover (Latte lavender) |
| `--accent-ink` | `#FFFFFF` | Text on accent fills |
| `--success` | `#40A02B` | Running / connected states (green) |
| `--warning` | `#DF8E1D` | Approval request states (yellow) |
| `--danger` | `#D20F39` | Reject / destructive states (red) |
| `--info` | `#1E66F5` | Informational states (blue) |
| `--console` | `#EFF1F5` | Terminal/console surfaces |
| `--console-elevated` | `#CCD0DA` | Console inner cards |
| `--on-console` | `#4C4F69` | Text on console |
| `--on-console-mute` | `#5C5F77` | Secondary text on console |

### Dark theme — Catppuccin Mocha

The dark theme is Catppuccin Mocha, the same palette as the author's
terminal pages (projs/me): crust canvas, mantle/base panes, mauve accent.
Status colors come straight from the Mocha palette.

| Token | Hex | Usage |
|---|---|---|
| `--canvas` | `#11111B` | Page background (crust) |
| `--surface` | `#181825` | Cards, header (mantle) |
| `--surface-muted` | `#1E1E2E` | Hover fills, secondary bands (base) |
| `--ink` | `#CDD6F4` | Headlines, primary text (text) |
| `--body` | `#A6ADC8` | Paragraph text (subtext0) |
| `--mute` | `#7F849C` | Metadata, captions (overlay1) |
| `--hairline` | `rgba(205,214,244,0.12)` | 1px borders/dividers |
| `--hairline-strong` | `rgba(205,214,244,0.30)` | Focused/active borders |
| `--accent` | `#CBA6F7` | Brand CTA, highlights (mauve) |
| `--accent-hover` | `#F5C2E7` | CTA hover (pink) |
| `--accent-ink` | `#11111B` | Text on accent fills (crust) |
| `--success` | `#A6E3A1` | Running / connected states (green) |
| `--warning` | `#FAB387` | Approval request states (peach) |
| `--danger` | `#F38BA8` | Reject / destructive states (red) |
| `--info` | `#89B4FA` | Informational states (blue) |
| `--console` | `#1E1E2E` | Terminal/console surfaces (base) |
| `--console-elevated` | `#313244` | Console inner cards (surface0) |
| `--on-console` | `#CDD6F4` | Text on console |
| `--on-console-mute` | `#9399B2` | Secondary text on console (overlay2) |

Rules:

- Mauve is the only marketing accent: `#8839EF` on Latte, `#CBA6F7` on
  Mocha. Use it for the primary CTA, diagram lanes, focus rings, and
  highlight glyphs. Never tint whole sections.
- Status colors (success/warning/danger/info) live inside console
  surfaces and approval UI, not on marketing chrome.
- Both themes must keep text contrast ≥ 4.5:1 (WCAG AA).

### Mockup palette

The device mockups are theme-aware: a Latte terminal on the light page, a
Mocha terminal on the dark one. The phone frame and macOS traffic lights
stay fixed realism colors.

Mocha (dark):

| Token | Hex | Usage |
|---|---|---|
| `--term-bg` | `#1E1E2E` | Terminal body (base) |
| `--term-bar` | `#181825` | Terminal title bar (mantle) |
| `--term-border` | `#45475A` | Terminal outer border (surface1) |
| `--term-fg` | `#CDD6F4` | Terminal text |
| `--term-mute` | `#9399B2` | Terminal secondary text |
| `--term-green` | `#A6E3A1` | `✓` success lines |
| `--term-blue` | `#89B4FA` | `▸` tool-call lines |
| `--term-yellow` | `#F9E2AF` | `!` waiting lines |
| `--term-orange` | `#FAB387` | Commands sent from phone |
| `--term-cyan` | `#94E2D5` | Prompt path |

Latte (light): `#EFF1F5` body, `#E6E9EF` bar, `rgba(76,79,105,0.18)` border,
`#4C4F69` text, `#5C5F77` mute, `#40A02B` green, `#1E66F5` blue,
`#DF8E1D` yellow, `#FE640B` orange, `#179299` cyan.

| Token | Hex | Usage |
|---|---|---|
| `--device-frame` | `#181825` | Phone bezel (same in both themes) |
| `--mac-red` / `--mac-yellow` / `--mac-green` | `#FF5F57` / `#FEBC2E` / `#28C840` | macOS traffic lights |

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

- Every card: 1px `hairline` border. Light theme adds one soft shadow
  (`0 1px 2px rgba(76,79,105,0.06), 0 8px 24px rgba(76,79,105,0.07)`);
  the dark TUI theme is **flat** — hairline only, `--shadow-card` is
  transparent.
- Page texture: a 40px hairline lattice behind everything, at ~30% of the
  hairline colour, never above content.
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
- Center: plain nav links: Architecture / How it works / Security / FAQ.
- Right: text-only language toggle `EN / 中`, theme toggle `Dark / Light`,
  mauve CTA.
- Mobile: nav collapses to a `menu` button; CTA stays visible.

### Hero

- Desktop: two columns. Left: `//` badge, display headline, description,
  primary (mauve) + secondary (outline) CTAs, and a one-line trust note.
  Right: console window.

### Device scene (hero)

- Left: macOS-style terminal window (theme-aware `term-*` palette —
  light terminal on the light page, dark on dark; square corners, 1px
  hairline): traffic-light dots only (no title), an in-progress
  user ↔ coding-CLI conversation in real-TUI style — full-width
  highlighted user bar (`❯`), mute "Thought for 4s…" lines, tool calls
  (`●` + `Edit(src/auth/middleware.ts)` with `└` sub-lines) — stuck at a
  full-width permission panel (blue top border, `Bash command`,
  `Do you want to proceed?`, highlighted `❯ 1. Yes`, key-hint footer),
  plus a TUI-style input box at the bottom (`›` prompt, blinking cursor).
- Right: Riffpad phone app (`surface`, square corners, hairline): status
  bar, `riffpad` header with `● synced`, session card for the same
  session, an approval card mirroring the terminal's pending approval
  with Approve / Reject chips, message input, bottom tabs.
- Between them: a two-lane sync connector — accent lane (events) and
  info-blue lane (commands) with tiny packets travelling both ways,
  `e2ee · 84ms` caption.
- The scene is interactive: approving/rejecting on the phone resolves
  the terminal's approval block; sending a preset message injects it
  into the terminal conversation (`⏎ phone`) and the agent answers on
  both sides.

### Architecture diagram

- `// architecture` label, title, description — same section header pattern
  as every other section.
- The diagram sits on a blueprint grid background (hairline grid lines,
  radial fade at the edges) — no window chrome.
- Animated SVG of the system, straight lanes only: `client` (your phone) ⇄
  `relay` (zero-knowledge) ⇄ `daemon` (your computer), the daemon fanning
  out to three icon-only CLI chips using the vendored brand icons
  (`components/brand-icons.ts`).
- Two straight flow lanes per link: accent dashed lane for events
  (daemon → phone), info-blue dashed lane for approvals (phone → daemon).
  Dashes scroll via `.flow-dash-line`; encrypted-envelope packets travel
  the lanes with SMIL `animateMotion` (hidden under
  `prefers-reduced-motion`).
- Node accent dots pulse; diagram scrolls horizontally below ~680px.

### How it works

- Three cards with large mono numbers `01 / 02 / 03`.
- Desktop: connected by a dashed 1px line through the row.
- Each card: number, title, description, and a step visual pinned to the
  bottom: a row of coding-CLI brand icons (vendored single-color SVG
  paths in `components/brand-icons.ts`, mute → ink on hover), a
  pseudo-QR pairing pattern (ink cells, accent finder blocks), and a
  push-notification card.

### Security console

- Left: `// security` label, title, description, and the security items
  as hairline-divided rows.
- Right: an animated end-to-end-encryption diagram (`.card`): plaintext
  lines leave the `daemon` card, pass an `encrypt` gate (padlock,
  accent dashed connector), turn into a dashed-border `relay` band of
  endlessly scrolling ciphertext (`.cipher-stream` marquee, duplicated
  content sliding -50%, alternating speeds/directions), then a
  `decrypt` gate (open padlock) restores the same plaintext at the
  `client` card. Ciphertext strings come from a seeded PRNG so SSR and
  client render match.

### FAQ

- Rows with `hairline` dividers, `md` radius on the whole block.
- Toggle marker: `>` when closed, `v` when open. Plain text, no chevron
  icons.

### CTA

- Centered block on `canvas`.
- Mauve CTA button `Get early access →`, then a small note.

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
- Use mauve as the brand accent on marketing surfaces; keep the rest of
  the Catppuccin ramp for status and consoles.
- Show the product as console + phone; the approval card is the hero.
- Use square cards with hairline borders: one soft shadow in Latte, flat
  panes in Mocha.
- Use minimal `>` / `//` / `●` glyphs only where they aid scanning.
- Keep CJK text sans-serif.

### Don't

- Don't recreate opencode's manpage: no cream-only canvas, no ASCII bracket
  bullets, no 4px-only geometry, no Berkeley-Mono-only identity.
- Don't use prompt-string chrome (`:~$`, `~/`, `$`) in the header, footer,
  or CTA.
- Don't wrap buttons or controls in brackets (`[button]`).
- Don't use Apple semantic colors on marketing chrome — status color comes
  from the Catppuccin palette.
- Don't add gradients, glassmorphism, or decorative imagery. The 40px
  lattice is the only background texture.
- Don't let the console surface appear more than once per viewport.
- Don't use emoji as icons.

## 9. Responsive

| Breakpoint | Behavior |
|---|---|
| ≥1024px | Two-column hero, 3-step row with dashed connector, bento 7/5 spans |
| 768–1023px | Hero stacks; bento keeps 7/5 spans |
| <768px | Single column; hero display 28px; bento stacks 1×1; nav collapses |

Touch targets ≥ 44px; focus rings visible in accent color; no horizontal
scroll. (The architecture diagram is the one exception: it keeps its 960px
canvas and scrolls horizontally *inside its own frame* below 768px — the page
itself never scrolls sideways.)

One container governs the whole page: top bar, hero, every section and the
footer all use `mx-auto w-full max-w-frame px-4 sm:px-6` (frame = 1200px), so
content edges line up from the logo down to the footer note. Narrower measures
(`max-w-[NNN]`, the CTA heading) cap line length on individual elements only —
never add a second page-width wrapper inside a section.
