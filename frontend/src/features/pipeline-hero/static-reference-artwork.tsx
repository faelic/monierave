import { cn } from "@/lib/utils/cn";

export function StaticReferenceArtwork({
  className,
}: {
  className?: string;
}) {
  return (
    <div
      aria-label="A metallic network of payment rails"
      className={cn("landing-reference-art", className)}
      role="img"
    />
  );
}
