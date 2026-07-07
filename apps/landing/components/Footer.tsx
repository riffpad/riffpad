"use client";

import { DeerflowSignature } from "./DeerflowSignature";
import { useLanguage } from "./LanguageProvider";

export function Footer() {
  const { t } = useLanguage();

  const footerLinks = [
    {
      title: t.footer.product,
      links: [
        { label: t.footer.links.features, href: "#features" },
        { label: t.footer.links.pricing, href: "#pricing" },
        { label: t.footer.links.changelog, href: "/changelog" },
      ],
    },
    {
      title: t.footer.resources,
      links: [
        { label: t.footer.links.docs, href: "/docs" },
        { label: t.footer.links.github, href: "https://github.com/riffpad/riffpad" },
        { label: t.footer.links.status, href: "https://status.riffpad.ai" },
      ],
    },
    {
      title: t.footer.company,
      links: [
        { label: t.footer.links.about, href: "/about" },
        { label: t.footer.links.contact, href: "mailto:hello@riffpad.ai" },
        { label: t.footer.links.privacy, href: "/privacy" },
      ],
    },
  ];

  return (
    <footer className="border-t border-white/5 bg-background px-4 py-12 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="grid gap-10 md:grid-cols-2 lg:grid-cols-5">
          <div className="lg:col-span-2">
            <a href="/" className="flex items-center gap-2 font-display text-lg font-bold text-foreground">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent text-black">
                R
              </span>
              Riffpad
            </a>
            <p className="mt-4 max-w-xs text-sm text-muted">{t.footer.description}</p>
          </div>

          {footerLinks.map((group) => (
            <div key={group.title}>
              <h4 className="text-sm font-semibold text-foreground">{group.title}</h4>
              <ul className="mt-4 space-y-2">
                {group.links.map((link) => (
                  <li key={link.href}>
                    <a
                      href={link.href}
                      className="text-sm text-muted transition hover:text-foreground"
                    >
                      {link.label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-12 flex flex-col items-center justify-between gap-4 border-t border-white/5 pt-8 text-sm text-muted sm:flex-row"
        >
          <p>© {new Date().getFullYear()} Riffpad. All rights reserved.</p>
          <div className="flex items-center gap-6">
            <a
              href="https://github.com/riffpad/riffpad"
              className="transition hover:text-foreground"
            >
              GitHub
            </a>
            <a
              href="https://twitter.com/riffpad"
              className="transition hover:text-foreground"
            >
              X / Twitter
            </a>
            <DeerflowSignature />
          </div>
        </div>
      </div>
    </footer>
  );
}
