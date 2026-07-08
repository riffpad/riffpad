import Image from "next/image";

export function Logo({ className = "h-8 w-8" }: { className?: string }) {
  return (
    <Image
      src="/rounded_rect_rotated_shapes_only_favicon.png"
      alt="Riffpad"
      width={32}
      height={32}
      unoptimized
      className={className}
    />
  );
}
