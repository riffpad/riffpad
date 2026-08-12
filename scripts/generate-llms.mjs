#!/usr/bin/env node
/**
 * Generates /llms.txt and /llms-full.txt for riffpad.ai from the docs sources.
 *
 * - llms.txt: curated index (H1 + blockquote + H2 groups) for LLM crawlers.
 * - llms-full.txt: every linked docs page concatenated as plain Markdown,
 *   so an agent can fetch the whole site context in one request.
 *
 * Runs automatically before the landing dev/build commands.
 */
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, "..");
const docsDir = resolve(root, "apps/docs");
const publicDir = resolve(root, "apps/landing/public");

const SITE = "https://riffpad.ai";
const TAGLINE =
  "Watch, approve, and steer AI coding agents from your phone.";

const GROUPS = [
  {
    heading: "Docs",
    items: [
      {
        title: "Quickstart",
        desc: "Install the CLI, sign in, pair, and start your first hosted session",
        file: "en/guide/quickstart.md",
        url: "/docs/en/guide/quickstart",
      },
      {
        title: "Remote control",
        desc: "Watch, approve, and steer coding agent sessions from your phone",
        file: "en/guide/remote.md",
        url: "/docs/en/guide/remote",
      },
      {
        title: "CLI",
        desc: "Full riffpad command reference",
        file: "en/reference/cli.md",
        url: "/docs/en/reference/cli",
      },
      {
        title: "Architecture",
        desc: "How the daemon, relay, and clients interact",
        file: "en/reference/architecture.md",
        url: "/docs/en/reference/architecture",
      },
      {
        title: "Security model",
        desc: "End-to-end encryption and zero-knowledge relay",
        file: "en/reference/security.md",
        url: "/docs/en/reference/security",
      },
      {
        title: "FAQ",
        desc: "Frequently asked questions",
        file: "en/faq.md",
        url: "/docs/en/faq",
      },
    ],
  },
];

function loadBody(file) {
  const raw = readFileSync(resolve(docsDir, file), "utf8");
  // Strip BOM and any VitePress frontmatter.
  const body = raw
    .replace(/^\uFEFF/, "")
    .replace(/^---\n[\s\S]*?\n---\n/, "")
    .trim();
  // Absolute-ize internal docs links so the full file works standalone.
  return body.replace(/(\]\()\/docs\//g, `$1${SITE}/docs/`) + "\n";
}

function renderIndex() {
  const lines = [
    `# Riffpad`,
    ``,
    `> ${TAGLINE}`,
    ``,
    `## Landing`,
    ``,
    `- [Riffpad](${SITE}/): ${TAGLINE}`,
    ``,
  ];

  for (const group of GROUPS) {
    lines.push(`## ${group.heading}`, ``);
    for (const item of group.items) {
      lines.push(`- [${item.title}](${SITE}${item.url}): ${item.desc}`);
    }
    lines.push(``);
  }

  lines.push(
    `## Project`,
    ``,
    `- [GitHub](https://github.com/riffpad/riffpad): Source code, issues, releases, and install scripts`,
    ``,
  );
  return lines.join("\n");
}

function renderFull() {
  const sections = [
    `# Riffpad`,
    ``,
    `> ${TAGLINE}`,
    ``,
    `## Landing`,
    ``,
    `${TAGLINE}`,
    ``,
    `Project site: ${SITE}/`,
    ``,
  ];

  for (const group of GROUPS) {
    for (const item of group.items) {
      sections.push(
        `## ${item.title}`,
        ``,
        loadBody(item.file),
        ``,
        `---`,
        ``,
      );
    }
  }

  return sections.join("\n").replace(/\n{3,}/g, "\n\n").trim() + "\n";
}

writeFileSync(resolve(publicDir, "llms.txt"), renderIndex());
writeFileSync(resolve(publicDir, "llms-full.txt"), renderFull());

console.log("Generated apps/landing/public/llms.txt and llms-full.txt");
