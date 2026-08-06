"use client";

import { AdaptiveVaultVisual } from "./adaptive-vault-visual";

export function VaultLaboratory() {
  return (
    <section aria-labelledby="vault-preview-title">
      <div className="max-w-3xl">
        <p className="text-jade-500 text-sm font-bold tracking-[0.16em] uppercase">
          Tier A experiment
        </p>
        <h2
          className="mt-3 font-serif text-3xl font-medium tracking-[-0.025em] text-white sm:text-4xl"
          id="vault-preview-title"
        >
          Card within a controlled security boundary
        </h2>
        <p className="mt-4 max-w-2xl leading-7 text-white/68">
          The composition uses a transparent vault, brushed card surface, and
          restrained transaction paths. It contains no customer or financial
          data.
        </p>
      </div>

      <AdaptiveVaultVisual
        className="vault-lab-stage mt-9 h-[22rem] rounded-lg border border-white/12 sm:aspect-[16/10] sm:h-auto"
        showStatus
        theme="dark"
      />
    </section>
  );
}
