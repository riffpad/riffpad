"use client";

import { motion } from "framer-motion";
import { Check } from "./Icons";
import { useLanguage } from "./LanguageProvider";
import { Hedgehog } from "./Hedgehog";

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
    <section id="pricing" className="bg-background px-4 py-20 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-xs font-bold uppercase tracking-wider text-body">
            {t.pricing.eyebrow}
          </p>
          <h2 className="mt-2 text-3xl font-bold text-foreground sm:text-4xl">
            {t.pricing.title}
          </h2>
          <p className="mt-3 text-body">{t.pricing.subtitle}</p>
        </div>

        <div className="mt-12 grid gap-4 lg:grid-cols-3">
          {plans.map((plan) => (
            <motion.div
              key={plan.name}
              whileHover={{ y: -4 }}
              transition={{ duration: 0.2 }}
              className="relative rounded-md border border-hairline bg-surface p-6 sm:p-8"
            >
              {plan.highlighted && (
                <>
                  <span className="absolute -top-3 left-6 rounded-full bg-accent px-2 py-0.5 text-xs font-bold text-on-accent">
                    {t.pricing.pro.popular}
                  </span>
                  <div className="absolute -right-3 -top-4 hidden lg:block">
                    <Hedgehog className="h-10 w-10" />
                  </div>
                </>
              )}
              <h3 className="text-lg font-bold text-foreground">{plan.name}</h3>
              <p className="mt-1 text-sm text-body">{plan.description}</p>
              <div className="mt-3 flex items-baseline">
                <span className="text-3xl font-extrabold text-foreground">
                  {plan.price}
                </span>
                <span className="ml-1 text-sm text-muted">{plan.period}</span>
              </div>

              <a
                href={plan.href}
                className={`mt-5 block rounded-md px-4 py-2.5 text-center text-sm font-bold transition ${
                  plan.highlighted
                    ? "bg-accent text-on-accent hover:bg-accent-pressed"
                    : "bg-surface-soft text-foreground hover:bg-hairline-soft"
                }`}
              >
                {plan.cta}
              </a>

              <ul className="mt-5 space-y-2">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-2 text-sm text-body">
                    <Check className="mt-0.5 h-4 w-4 shrink-0 text-accent-green" />
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
