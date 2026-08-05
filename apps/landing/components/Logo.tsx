export function Logo({ className = "h-6 w-6" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 300 300"
      className={className}
      aria-hidden="true"
      fill="currentColor"
    >
      <g transform="rotate(-90, 150, 150)">
        <rect x="70" y="70" width="40" height="40" fill="#F7A501" />
        <rect x="130" y="70" width="40" height="40" fill="#F7A501" />
        <rect x="190" y="70" width="40" height="40" fill="#F7A501" />
        <rect x="70" y="130" width="100" height="40" fill="#F7A501" />
        <rect x="190" y="130" width="40" height="40" fill="#F7A501" />
        <rect x="70" y="190" width="160" height="40" fill="#F7A501" />
      </g>
    </svg>
  );
}
