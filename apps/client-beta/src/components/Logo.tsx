// Riffpad mark from riffpad-legacy (shapes-only variant). The fill uses
// currentColor so CSS can theme it (ink in light mode, white in dark).
export default function Logo({ size = 28 }: { size?: number }) {
  return (
    <svg
      className="logo"
      width={size}
      height={size}
      viewBox="0 0 300 300"
      fill="currentColor"
      aria-hidden="true"
    >
      <g transform="rotate(-90, 150, 150)">
        <rect x="70" y="70" width="40" height="40" />
        <rect x="130" y="70" width="40" height="40" />
        <rect x="190" y="70" width="40" height="40" />
        <rect x="70" y="130" width="100" height="40" />
        <rect x="190" y="130" width="40" height="40" />
        <rect x="70" y="190" width="160" height="40" />
      </g>
    </svg>
  );
}
