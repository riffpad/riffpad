export function Hedgehog({ className = "w-16 h-16" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 64 64"
      className={className}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <style>{`
        .spike { fill: var(--hedgehog-spike); }
        .body { fill: var(--accent); }
        .face { fill: var(--hedgehog-spike); }
      `}</style>
      <path
        d="M32 8L26 18L18 14L20 24L10 26L18 32L10 40L20 40L18 50L28 44L32 54L36 44L46 50L44 40L54 40L46 32L54 26L44 24L46 14L38 18L32 8Z"
        className="spike"
      />
      <circle cx="32" cy="36" r="16" className="body" />
      <circle cx="26" cy="34" r="2" className="face" />
      <circle cx="38" cy="34" r="2" className="face" />
      <ellipse cx="32" cy="40" rx="3" ry="2" className="face" />
      <circle cx="26" cy="34" r="4" stroke="var(--hedgehog-spike)" strokeWidth="1.5" fill="none" />
      <circle cx="38" cy="34" r="4" stroke="var(--hedgehog-spike)" strokeWidth="1.5" fill="none" />
      <line x1="30" y1="34" x2="34" y2="34" stroke="var(--hedgehog-spike)" strokeWidth="1.5" />
    </svg>
  );
}
