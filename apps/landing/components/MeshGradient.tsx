export function MeshGradient({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 1200 600"
      preserveAspectRatio="xMidYMid slice"
      className={className}
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        <filter id="blur" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur in="SourceGraphic" stdDeviation="80" />
        </filter>
        <linearGradient id="develop" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#007cf0" />
          <stop offset="100%" stopColor="#00dfd8" />
        </linearGradient>
        <linearGradient id="preview" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#7928ca" />
          <stop offset="100%" stopColor="#ff0080" />
        </linearGradient>
        <linearGradient id="ship" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#ff4d4d" />
          <stop offset="100%" stopColor="#f9cb28" />
        </linearGradient>
      </defs>
      <g filter="url(#blur)" opacity="0.45">
        <ellipse cx="200" cy="150" rx="350" ry="250" fill="url(#develop)" />
        <ellipse cx="900" cy="200" rx="300" ry="220" fill="url(#preview)" />
        <ellipse cx="600" cy="500" rx="380" ry="240" fill="url(#ship)" />
      </g>
    </svg>
  );
}
