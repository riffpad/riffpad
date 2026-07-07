"use client";

import { motion } from "framer-motion";
import { Check } from "./Icons";
import { useLanguage } from "./LanguageProvider";

export function Pricing() {
  const { t, lang } = useLanguage();

  const currency = lang === "zh" ? "¥" : "$";
  const plans = [
    {
      name: t.pricing.free.name,
      price: `${currency}0`,
      period: t.pricing.free.period,
      description: t.pricing.free.description,
      features: t.pricing.freeFeatures,
      cta: t.pricing.free.cta,
      href: "https://app.riffpad.ai",
      highlighted: false,
    },
    {
      name: t.pricing.pro.name,
      price: `${currency}${lang === "zh" ? "79" : "12"}`,
      period: t.pricing.pro.period,
      description: t.pricing.pro.description,
      features: t.pricing.proFeatures,
      cta: t.pricing.pro.cta,
      href: "https://app.riffpad.ai/billing",
      highlighted: true,
    },
    {
      name: t.pricing.team.name,
      price: `${currency}${lang === "zh" ? "249" : "39"}`,
      period: t.pricing.team.period,
      description: t.pricing.team.description,
      features: t.pricing.teamFeatures,
      cta: t.pricing.team.cta,
      href: "mailto:hello@riffpad.ai",
      highlighted: false,
    },
  ];

  return (
    <section id="pricing" className="relative bg-surface px-4 py-24 sm:px-6 lg:px-8">
      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute right-0 top-0 h-[500px] w-[500px] rounded-full bg-blue-600/5 blur-[120px]" />
      </div>

      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-semibold uppercase tracking-widest text-accent">
            {t.pricing.eyebrow}
          </p>
          <h2 className="mt-3 font-display text-3xl font-bold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
            {t.pricing.title}
          </h2>
          <p className="mt-4 text-muted">{t.pricing.subtitle}</p>
        </div>

        <div className="mt-16 grid gap-6 lg:grid-cols-3">
          {plans.map((plan) => (
            <motion.div
              key={plan.name}
              whileHover={{ y: -6 }}
              transition={{ duration: 0.2 }}
              className={`relative rounded-2xl p-6 sm:p-8 ${
                plan.highlighted
                  ? "border-2 border-accent bg-background shadow-[0_0_60px_-15px_rgba(200,255,0,0.15)]"
                  : "border border-white/10 bg-background"
              }`}
            >
              {plan.highlighted && (
                <span className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-accent px-3 py-1 text-xs font-bold text-black">
                  {t.pricing.pro.popular}
                </span>
              )}
              <h3 className="text-lg font-semibold text-foreground">{plan.name}</h3>
              <p className="mt-1 text-sm text-muted">{plan.description}</p>
              <div className="mt-4 flex items-baseline">
                <span className="font-display text-4xl font-bold tracking-tight text-foreground">
                  {plan.price}
                </span>
                <span className="ml-1 text-sm text-muted">{plan.period}</span>
              </div>

              <a
                href={plan.href}
                className={`mt-6 block rounded-lg px-4 py-3 text-center text-sm font-bold transition ${
                  plan.highlighted
                    ? "bg-accent text-black hover:bg-accent-hover"
                    : "border border-white/10 bg-surface text-foreground hover:border-white/20"
                }`}
              >
                {plan.cta}
              </a>

              <ul className="mt-6 space-y-3">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-3 text-sm text-muted">
                    <Check className="mt-0.5 h-4 w-4 shrink-0 text-accent" />
                    {feature}
                  </li>
                ))}
              </ul>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
