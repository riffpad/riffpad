# Riffpad Landing

Marketing site for Riffpad, deployed to `riffpad.ai`.

## Tech Stack

- Next.js 14 (App Router)
- TypeScript
- Tailwind CSS
- Framer Motion

## Development

```bash
cd apps/landing
pnpm install
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000).

## Build

```bash
pnpm build
```

The site is exported to `dist/` for static hosting.

## Preview the static export

Because Next.js uses absolute asset paths (`/_next/...`), you must serve `dist/` over HTTP instead of opening `dist/index.html` directly with the browser.

```bash
pnpm serve
# or
python3 -m http.server 3000 --directory dist
```

Then open [http://localhost:3000](http://localhost:3000).

## Structure

```
app/          # Next.js App Router pages
components/   # React components
lib/          # Utilities and helpers
public/       # Static assets
```
