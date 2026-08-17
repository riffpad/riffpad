"use client";

import { useState, type ReactNode } from "react";
import { MenuIcon, CloseIcon } from "./icons";

type MobileMenuProps = {
  children: ReactNode;
};

export function MobileMenu({ children }: MobileMenuProps) {
  const [open, setOpen] = useState(false);

  return (
    <div className="relative md:hidden">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex h-9 w-9 cursor-pointer items-center justify-center text-mute transition-colors hover:text-ink"
        aria-label={open ? "Close menu" : "Open menu"}
        aria-expanded={open}
      >
        {open ? <CloseIcon className="h-4 w-4" /> : <MenuIcon className="h-4 w-4" />}
      </button>

      {open && (
        <div
          className="absolute right-0 top-full mt-2 border border-hairline bg-surface p-2 shadow-card"
          onClick={() => setOpen(false)}
        >
          <div className="flex flex-col gap-1">
            {children}
          </div>
        </div>
      )}
    </div>
  );
}
