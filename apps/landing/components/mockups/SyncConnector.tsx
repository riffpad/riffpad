"use client";

export function SyncConnector({
  syncing,
  label,
}: {
  syncing: boolean;
  label: string;
}) {
  const dash = syncing ? "flow-dash-line flow-fast" : "flow-dash-line";
  return (
    <div className="flex flex-col items-center" aria-hidden="true">
      {/* mobile: vertical connector spanning the gap between the two devices */}
      <svg
        className="h-32 w-8 lg:hidden"
        viewBox="0 0 24 96"
      >
        <path
          id="sync-ev-v"
          d="M7 96 L7 0"
          fill="none"
          stroke="var(--accent)"
          strokeOpacity={0.6}
          strokeWidth={2}
          strokeDasharray="6 6"
          className={dash}
        />
        <path
          id="sync-cmd-v"
          d="M17 96 L17 0"
          fill="none"
          stroke="rgb(var(--info))"
          strokeOpacity={0.6}
          strokeWidth={2}
          strokeDasharray="6 6"
          className={dash}
        />
        <rect
          width={8}
          height={5}
          x={-4}
          y={-2.5}
          fill="var(--accent)"
          className="arch-packet"
        >
          <animateMotion dur="1.6s" repeatCount="indefinite">
            <mpath href="#sync-ev-v" />
          </animateMotion>
          <animate
            attributeName="opacity"
            values="0;1;1;0"
            keyTimes="0;0.2;0.8;1"
            dur="1.6s"
            repeatCount="indefinite"
          />
        </rect>
        <rect
          width={8}
          height={5}
          x={-4}
          y={-2.5}
          fill="rgb(var(--info))"
          className="arch-packet"
        >
          <animateMotion dur="1.6s" begin="-0.8s" repeatCount="indefinite">
            <mpath href="#sync-cmd-v" />
          </animateMotion>
          <animate
            attributeName="opacity"
            values="0;1;1;0"
            keyTimes="0;0.2;0.8;1"
            dur="1.6s"
            begin="-0.8s"
            repeatCount="indefinite"
          />
        </rect>
      </svg>

      {/* desktop: horizontal connector between the two devices */}
      <svg
        className="hidden h-6 w-16 lg:block"
        viewBox="0 0 64 24"
      >
        <path
          id="sync-ev"
          d="M64 7 L0 7"
          fill="none"
          stroke="var(--accent)"
          strokeOpacity={0.6}
          strokeWidth={2}
          strokeDasharray="6 6"
          className={dash}
        />
        <path
          id="sync-cmd"
          d="M0 17 L64 17"
          fill="none"
          stroke="rgb(var(--info))"
          strokeOpacity={0.6}
          strokeWidth={2}
          strokeDasharray="6 6"
          className={dash}
        />
        <rect
          width={8}
          height={5}
          x={-4}
          y={-2.5}
          fill="var(--accent)"
          className="arch-packet"
        >
          <animateMotion dur="1.6s" repeatCount="indefinite">
            <mpath href="#sync-ev" />
          </animateMotion>
          <animate
            attributeName="opacity"
            values="0;1;1;0"
            keyTimes="0;0.2;0.8;1"
            dur="1.6s"
            repeatCount="indefinite"
          />
        </rect>
        <rect
          width={8}
          height={5}
          x={-4}
          y={-2.5}
          fill="rgb(var(--info))"
          className="arch-packet"
        >
          <animateMotion dur="1.6s" begin="-0.8s" repeatCount="indefinite">
            <mpath href="#sync-cmd" />
          </animateMotion>
          <animate
            attributeName="opacity"
            values="0;1;1;0"
            keyTimes="0;0.2;0.8;1"
            dur="1.6s"
            begin="-0.8s"
            repeatCount="indefinite"
          />
        </rect>
      </svg>
      <span className="hidden text-[10px] text-mute lg:block">{label}</span>
    </div>
  );
}

