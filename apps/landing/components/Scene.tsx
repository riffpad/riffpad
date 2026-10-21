"use client";

import { DeviceMockup } from "./DeviceMockup";
import { InstallCommand } from "./InstallCommand";

export function Scene() {
  return (
    <section className="mx-auto w-full max-w-frame px-4 py-16 sm:px-6 sm:py-24">
      <DeviceMockup />
      <InstallCommand className="mt-10 sm:mt-12" />
    </section>
  );
}
