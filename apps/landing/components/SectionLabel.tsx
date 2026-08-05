export function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <span className="text-sm font-bold uppercase tracking-wider text-mute">
      {children}
    </span>
  );
}
