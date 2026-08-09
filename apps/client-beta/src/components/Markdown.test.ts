import { describe, expect, it } from "vitest";
import { markdownToHtml } from "./Markdown";

describe("markdownToHtml", () => {
  it("renders plaintext code fences without crashing", () => {
    const html = markdownToHtml("```plaintext\nhello world\n```");
    expect(html).toContain("hello world");
  });

  it("renders unknown-language code fences as plain text", () => {
    const html = markdownToHtml("```yaml\nfoo: bar\n```");
    expect(html).toContain("foo: bar");
  });

  it("renders language-less code fences without crashing", () => {
    const html = markdownToHtml("```\njust text\n```");
    expect(html).toContain("just text");
  });

  it("renders basic markdown", () => {
    const html = markdownToHtml("**bold** and `code`");
    expect(html).toContain("<strong>bold</strong>");
  });
});
