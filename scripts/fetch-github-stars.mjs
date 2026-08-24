#!/usr/bin/env node
/**
 * Fetches the GitHub stargazer count at build time and writes it to a
 * static JSON file consumed by the landing page header.
 *
 * This avoids client-side calls to api.github.com, which are rate-limited
 * per IP and frequently fail for visitors.
 */
import { writeFileSync, mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, "..");
const libDir = resolve(root, "apps/landing/lib");

const OUTFILE = resolve(libDir, "github-stars.json");

async function main() {
  let stars = null;
  try {
    const headers = { Accept: "application/vnd.github+json" };
    const token = process.env.GITHUB_TOKEN;
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
    const res = await fetch("https://api.github.com/repos/riffpad/riffpad", { headers });
    if (res.ok) {
      const data = await res.json();
      if (typeof data.stargazers_count === "number") {
        stars = data.stargazers_count;
      }
    } else {
      console.warn(`GitHub API returned ${res.status}; using cached/default stars`);
    }
  } catch (err) {
    console.warn("Failed to fetch GitHub stars:", err.message);
  }

  mkdirSync(libDir, { recursive: true });
  writeFileSync(OUTFILE, JSON.stringify({ stars, fetchedAt: new Date().toISOString() }, null, 2));
  console.log(`Wrote ${OUTFILE}: stars=${stars ?? "null"}`);
}

main();
