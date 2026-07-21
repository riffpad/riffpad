"use client";

import { useEffect, useState } from "react";
import {
  Group,
  Panel,
  Separator,
  type Layout,
  useDefaultLayout,
} from "react-resizable-panels";

const STORAGE_KEY = "riffpad:dock-layout:v1";

interface DockLayoutProps {
  left: React.ReactNode;
  center: React.ReactNode;
  right: React.ReactNode;
}

const PANEL_IDS = ["file-tree", "center", "chat"] as const;

function storage(): Storage {
  if (typeof window === "undefined") {
    return {
      getItem: () => null,
      setItem: () => {},
      removeItem: () => {},
      length: 0,
      key: () => null,
      clear: () => {},
    };
  }
  return window.localStorage;
}

export function DockLayout({ left, center, right }: DockLayoutProps) {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const { defaultLayout, onLayoutChanged } = useDefaultLayout({
    id: "riffpad-dock",
    panelIds: PANEL_IDS as unknown as string[],
    storage: {
      getItem: (key) => storage().getItem(`${STORAGE_KEY}:${key}`),
      setItem: (key, value) => storage().setItem(`${STORAGE_KEY}:${key}`, value),
    },
  });

  if (!mounted) {
    // Avoid hydration mismatch by rendering a static layout first.
    return (
      <div className="flex-1 grid grid-cols-[minmax(0,260px)_minmax(0,1fr)_minmax(0,320px)] overflow-hidden">
        <div className="bg-card/70 backdrop-blur flex flex-col min-h-0">
          {left}
        </div>
        <div className="flex flex-col min-w-0 bg-card/30 min-h-0">
          {center}
        </div>
        <div className="flex flex-col bg-card/70 backdrop-blur min-w-0 min-h-0">
          {right}
        </div>
      </div>
    );
  }

  return (
    <Group
      id="riffpad-dock"
      orientation="horizontal"
      style={{ height: "100%" }}
      defaultLayout={defaultLayout}
      onLayoutChanged={onLayoutChanged}
    >
      <Panel
        id={PANEL_IDS[0]}
        defaultSize={260}
        minSize={180}
        maxSize={400}
        className="flex flex-col bg-card/70 backdrop-blur min-h-0"
      >
        {left}
      </Panel>
      <Separator className="group relative w-[5px] bg-transparent focus-visible:outline-none">
        <div className="absolute inset-y-0 left-1/2 -translate-x-1/2 w-px rounded-full bg-hairline transition-all group-hover:w-[3px] group-hover:bg-primary/40 group-data-[separator-active]:w-[3px] group-data-[separator-active]:bg-primary/50" />
      </Separator>
      <Panel
        id={PANEL_IDS[1]}
        minSize={320}
        className="flex flex-col min-w-0 bg-card/30 min-h-0"
      >
        {center}
      </Panel>
      <Separator className="group relative w-[5px] bg-transparent focus-visible:outline-none">
        <div className="absolute inset-y-0 left-1/2 -translate-x-1/2 w-px rounded-full bg-hairline transition-all group-hover:w-[3px] group-hover:bg-primary/40 group-data-[separator-active]:w-[3px] group-data-[separator-active]:bg-primary/50" />
      </Separator>
      <Panel
        id={PANEL_IDS[2]}
        defaultSize={320}
        minSize={260}
        maxSize={500}
        className="flex flex-col bg-card/70 backdrop-blur min-w-0 min-h-0"
      >
        {right}
      </Panel>
    </Group>
  );
}

export type { Layout };
