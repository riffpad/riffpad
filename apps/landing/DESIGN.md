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
- The hero opens with one claim and one CTA; below it sits a two-pane
  scene: an nvim TUI running a coding CLI and the Riffpad phone app, on the
  same session. The TUI waits for approval; the phone shows the approval
  card — the product, not a metaphor.
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

The hero scene is a screenshot of a real session, so it keeps one palette:
**Catppuccin Mocha in both page themes**. The TUI pane and the phone next to
it share the same fixed ramp; only `tui-bg` follows the theme, because the
dark page needs a three-step stack — page `crust` (`#11111B`) < panes
`mantle` (`#181825`) < muted band `base` (`#1E1E2E`) — so the panes lift off
the band instead of dissolving into it. Light keeps `base` and is unchanged.

| Token | Hex | Usage |
|---|---|---|
| `tui-bg` | `#1E1E2E` light / `#181825` dark | TUI body and phone screen (base on Latte pages, mantle on Mocha pages) |
| `tui-hairline` | `rgba(205,214,244,0.14)` | Panel outline (command bar, TUI, phone) — same in both themes |
| `tui-divider` | `rgba(205,214,244,0.10)` | Inner separators: tabline edge, statusline top, sheet/card rules |
| `tui-border` | `#45475A` | Interaction-only emphasis (hover/focus). Never a static frame — on the dark surfaces it reads as a selection ring |
| `tui-surface` | `#313244` | Statusline, selected option, hairline rules (surface0) |
| `tui-fg` | `#CDD6F4` | TUI text (text) |
| `tui-mute` | `#6C7086` | Gutter numbers, hints (overlay0) |
| `tui-mauve` | `#CBA6F7` | `NORMAL` block, user prompt, running dots (mauve) |
| `tui-green` | `#A6E3A1` | `●` tool rows, `✓` approved (green) |
| `tui-blue` | `#89B4FA` | Approval panel border, `❯` selected option (blue) |
| `tui-red` | `#F38BA8` | `✗` rejected (red) |

## 3. Typography

One family everywhere: **Geist Mono** (variable, 100–900), self-hosted.

| Token | Size | Weight | Line Height | Tracking | Use |
|---|---|---|---|---|---|
| `display-xl` | `clamp(36px, 5.4vw, 84px)` (78px at 1440) | 700 | 1.04 | -0.03em | Hero headline |
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

### Section rhythm

Backgrounds alternate screen by screen. Odd screens sit on `canvas` (left
transparent, so the 40px lattice shows through); even screens are flat
`surface-muted` full-bleed bands whose content still fits the 1200 container:

| # | Block | Background |
|---|---|---|
| 1 | Hero copy | `canvas` |
| 2 | Scene (TUI + phone) | `surface-muted` band |
| 3 | Architecture | `canvas` |
| 4 | How it works | `surface-muted` band |
| 5 | Security | `canvas` |
| 6 | FAQ | `surface-muted` band |
| 7 | CTA | `canvas` |
| 8 | Footer | `surface` + 1px `hairline` top rule |

Bands carry their own vertical breathing room (`py-16`, `py-24` from `sm`
up) so panes never touch the band edge, and their top edge always starts
below the fold — the hero copy keeps the first screen to itself.

### Top bar

- Height 56px, `surface` background, 1px `hairline` bottom rule.
- Left: Riffpad logo mark + `riffpad` wordmark.
- Center: plain nav links: Architecture / How it works / Security / FAQ.
- Right: text-only language toggle `EN / 中`, theme toggle `Dark / Light`,
  mauve CTA.
- Mobile: nav collapses to a `menu` button; CTA stays visible.

### Hero

- One column, left-aligned: `// remote control for cli agents` label, then a
  near-full-width display headline (`clamp(36px, 5.4vw, 84px)`, `1.04`
  leading — ~10 of the 12 columns at 1440), **one** line of supporting copy,
  the CTA row, and the install command line.
- The primary CTA is the loudest thing on the page: accent fill, `64px`
  tall, `18px` label, full-width on mobile; `Read the docs` sits next to it
  as a small plain underlined text link — a full step below it in weight.
- The install prompt reads as a prompt, not a shell command: no `❯` glyph.
  It sits on the same Mocha surface as the TUI panes (`tui-bg` /
  `tui-hairline`, square, no shadow) with three cells — a `PASTE INTO YOUR
  AGENT` label cell in mauve (uppercase), the whole prompt in mono
  (`help me set up riffpad: curl -fsSL https://riffpad.ai/SKILL.md`) and
  `COPY` as a bordered cell on the right (72px on phones, 88px from `sm`).
  The copy cell is wide enough for `COPIED ✓`, so the label swap never moves
  the divider. One row from `sm` up; below it the prompt drops to its own
  full-width row under the label and wraps to two lines — never clipped,
  never scrolled sideways.
- `// v0.2.6 · self-hosted · e2e` closes that row at the far right (static
  text, `xl` and up) — never a graphic.
- The copy block owns the whole first screen: it is `100svh - 56px` tall
  (the header offset) and vertically centred, so nothing else may peek above
  the fold. The scene band (TUI + phone) starts on the next screen, as its
  own full-bleed block.

### Device scene (hero) — nvim TUI + phone

- Two panes, top-aligned and bottom-aligned: the wide pane is a Neovim
  session (TUI, `2fr`), the narrow one is the Riffpad phone (`1fr`, capped at
  320px). No connector, divider or caption between them — the gap carries the
  relationship.
- The TUI pane is a screenshot, not a window: no traffic lights, no window
  title bar, no shadow. It is flat and square with a 1px `tui-hairline`
  outline, and it always renders Catppuccin **Mocha**, in both page themes.
  Panes sit above the band on fill alone (page `crust` < panes `mantle` <
  band `base`) — the frame stays a hairline, never a bright ring.
- Chrome, top to bottom: a `tui-surface` tabline (file name tab + modified
  `+`), the transcript, the statusline. Nothing else.
- Gutter: plain absolute line numbers (`1…N`) in `tui-mute`; the cursor line
  is `tui-fg`. No relative numbers.
- Content: the user prompt (`❯`, mauve), a mute "thought 4s · read 3 files"
  line, `●` tool calls with `└` sub-lines, then the agent's pending
  `Bash(rm src/old.ts)` call.
- While the approval is pending, a four-line `tui-blue` bordered panel sits
  inside the pane: `Bash command`, the command itself, then `❯ 1. Yes`
  (selected, `tui-surface` fill) and `2. No`. No prose, no key-hint footer.
- The panel lives in a **fixed-height slot** (96px, 110px from `sm`), so
  resolving the approval leaves the pane, the band and everything below it at
  exactly the same size — the result line takes the last transcript row and
  the slot simply empties.
- Statusline (single row, `tui-surface`): `NORMAL` mode block (mauve fill,
  `tui-bg` text), the file path, then right-aligned `claude · utf-8 · 37%`.
- The phone is the same Mocha ramp and the same flat geometry: 12px radius
  at most, 1px `tui-hairline`, **no shadow in either theme** (no bezel either —
  the phone is a pane, not a device impression). It stretches to
  the pane's height so both bottom edges line up.
- Phone content: session bar with `● synced`, the metadata chips, six
  finished tool rows (enough to fill the pane's height — never a blank band
  under the header), the pending `$ rm src/old.ts` row, then an approval
  sheet pinned to the bottom edge — status dot, summary, `Approve` /
  `Reject` (44px tall).
- The scene is interactive: approving/rejecting on the phone drops the TUI
  approval panel and swaps the pane's `Waiting for approval…` line **in
  place** for `✓ approved from phone · rm src/old.ts` (green) or `✗ rejected
  from phone · agent paused` (red) — no stale waiting row above the result.
  The phone sheet collapses to an `Approved` / `Rejected` chip.

### Architecture diagram

- `// architecture` label, title, description — same section header pattern
  as every other section.
- The diagram sits on a blueprint grid background (24px hairline grid,
  radial fade at the edges) — no window chrome. The panel is opaque
  (`bg-surface`) so it masks the page's 40px body lattice underneath: only
  the 24px blueprint grid shows inside the frame, in both themes — never a
  grid inside a grid.
- One SVG of the system, straight lanes only: `client` (your phone) ⇄
  `relay` (zero-knowledge) ⇄ `daemon` (your computer), the daemon fanning
  out to three icon-only CLI chips using the vendored brand icons
  (`components/brand-icons.ts`).
- Two straight flow lanes per link: accent dashed lane for events
  (daemon → phone), info-blue dashed lane for approvals (phone → daemon).
  The dashes flow (`.flow-dash-line` over `strokeDasharray="6 6"`, 1s linear
  loop; `.flow-fast` runs at 0.45s) and the same class drives the
  daemon ↔ CLI chip links. A packet rides each lane on a 4s `animateMotion`
  loop, fading in and out at both ends, and the node accent dots pulse.
- The diagram scrolls horizontally below ~680px.

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

- Scroll reveals carry the sections. `ScrollReveal` (IntersectionObserver,
  `threshold: 0.1`, `rootMargin: "0px 0px -50px 0px"`) lifts each block 24px
  and fades it in over `500ms ease-out`, then unobserves. Delays stagger
  0 / 75 / 150 / 225ms down a section, plus 110ms per card in the
  how-it-works row. It wraps the same blocks in Architecture, How it works,
  Security, FAQ, CTA and Footer.
- The architecture lanes flow (`.flow-dash-line`, 1s linear, `.flow-fast` at
  0.45s — keep the dasharray at 6 6 so the loop is seamless), packets travel
  the event and approval lanes on 4s loops with an opacity fade at both ends,
  and the node accent dots pulse.
- The security `relay` band is a ciphertext marquee: `.cipher-stream` slides
  duplicated rows `translateX(-50%)` at 18s + 5s per line, alternating
  direction.
- Buttons and links change colour over `transition-colors` (`200ms`) instead
  of instantly: the primary CTA darkens one step, ghost and mute links go
  `ink`.
- Reduced motion is a hard contract. A global `prefers-reduced-motion:
  reduce` override pins every animation to `0.01ms` / one iteration and every
  transition to `0.01ms` (and `scroll-behavior: auto`); `.arch-packet` is
  `display: none` because SMIL ignores that override; `ScrollReveal` skips
  the observer and renders its final state (`opacity-100`, no translate).
  Under reduced motion everything is static and fully visible — no hidden
  or half-faded block.

## 8. Do / Don't

### Do

- Keep monospace as the single voice; let weight and size do hierarchy.
- Use mauve as the brand accent on marketing surfaces; keep the rest of
  the Catppuccin ramp for status and consoles.
- Show the product as an nvim TUI + phone; the approval card is the hero.
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
| ≥1024px | Hero copy fills the fold, scene splits 2fr/1fr (TUI + phone, equal heights); 3-step row with dashed connector; bento 7/5 spans |
| 768–1023px | Hero scene stacks (TUI, hairline, centred phone); bento keeps 7/5 spans |
| <768px | Single column; hero display 28px; CTA full-width; bento stacks 1×1; nav collapses |

Touch targets ≥ 44px; focus rings visible in accent color; no horizontal
scroll. (The architecture diagram is the one exception: it keeps its 960px
canvas and scrolls horizontally *inside its own frame* below 768px — the page
itself never scrolls sideways.)

One container governs the whole page: top bar, hero, every section and the
footer all use `mx-auto w-full max-w-frame px-4 sm:px-6` (frame = 1200px), so
content edges line up from the logo down to the footer note. Narrower measures
(`max-w-[NNN]`, the CTA heading) cap line length on individual elements only —
never add a second page-width wrapper inside a section.
