import Image from "next/image";

import { cn } from "@/lib/utils/cn";

import { walletAssets } from "../config/wallet-config";

export function SceneFallback({ className }: { className?: string }) {
  return (
    <figure
      className={cn(
        "relative mx-auto aspect-[4/3] max-h-full w-full max-w-[48rem]",
        className,
      )}
    >
      <Image
        alt="An open Monierave digital wallet holding three branded payment cards"
        className="object-contain"
        fill
        priority
        sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 44rem"
        src={walletAssets.fallback}
      />
    </figure>
  );
}
