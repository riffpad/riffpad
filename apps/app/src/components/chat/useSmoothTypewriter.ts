"use client";

import { useEffect, useRef, useState } from "react";

const MS_PER_CHAR = 14;
const MAX_DURATION = 600;
const MIN_DURATION = 50;

export function useSmoothTypewriter(
  target: string,
  isActive: boolean,
  freeze = false
) {
  const [displayed, setDisplayed] = useState(target);
  const targetRef = useRef(target);
  const displayedRef = useRef(target);
  const frozenRef = useRef<string | null>(null);

  useEffect(() => {
    targetRef.current = target;

    if (freeze && frozenRef.current === null) {
      frozenRef.current = displayedRef.current;
    }

    if (freeze) {
      return;
    }

    if (!isActive) {
      if (displayedRef.current !== target) {
        displayedRef.current = target;
        setDisplayed(target);
      }
      return;
    }

    if (displayedRef.current.length >= target.length) {
      return;
    }

    const startLength = displayedRef.current.length;
    const charsToReveal = target.length - startLength;
    const duration = Math.min(
      MAX_DURATION,
      Math.max(MIN_DURATION, charsToReveal * MS_PER_CHAR)
    );

    let startTime: number | null = null;
    let raf = 0;

    const animate = (now: number) => {
      if (startTime === null) startTime = now;
      const elapsed = now - startTime;
      const progress = Math.min(1, elapsed / duration);
      const eased = 1 - Math.pow(1 - progress, 3);
      const nextLength = Math.floor(startLength + charsToReveal * eased);
      const next = target.slice(0, nextLength);

      if (displayedRef.current !== next) {
        displayedRef.current = next;
        setDisplayed(next);
      }

      if (progress < 1 && targetRef.current === target) {
        raf = requestAnimationFrame(animate);
      }
    };

    raf = requestAnimationFrame(animate);

    return () => {
      cancelAnimationFrame(raf);
    };
  }, [target, isActive]);

  return freeze ? frozenRef.current ?? displayed : displayed;
}
