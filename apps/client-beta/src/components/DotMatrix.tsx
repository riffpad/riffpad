// 3×3 dot-matrix loading indicator: dots light up in sequence, sweeping
// across the matrix like a marquee.
export default function DotMatrix() {
  return (
    <span className="dot-matrix" aria-hidden="true">
      {Array.from({ length: 9 }).map((_, i) => (
        <span key={i} style={{ animationDelay: `${i * 0.12}s` }} />
      ))}
    </span>
  );
}
