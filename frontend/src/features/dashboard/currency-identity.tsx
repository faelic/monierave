import type { Account } from "@/lib/api/contracts";

const currencyNames: Record<Account["currency"], string> = {
  EUR: "Euro",
  USD: "US Dollar",
};

export function CurrencyIdentity({
  currency,
  compact = false,
}: {
  compact?: boolean;
  currency: Account["currency"];
}) {
  return (
    <span className="inline-flex items-center gap-2">
      <CurrencyFlag currency={currency} />
      <span className="font-semibold">{currency}</span>
      {compact ? null : (
        <span className="text-ink-600">· {currencyNames[currency]}</span>
      )}
    </span>
  );
}

function CurrencyFlag({ currency }: { currency: Account["currency"] }) {
  return (
    <span
      aria-hidden="true"
      className="border-line-200 inline-grid size-5 shrink-0 overflow-hidden rounded-full border bg-white shadow-sm"
    >
      {currency === "USD" ? <UnitedStatesFlag /> : <EuropeanUnionFlag />}
    </span>
  );
}

function UnitedStatesFlag() {
  return (
    <svg viewBox="0 0 24 24">
      <rect fill="#fff" height="24" width="24" />
      {[0, 4, 8, 12, 16, 20].map((y) => (
        <rect fill="#b22234" height="2" key={y} width="24" y={y} />
      ))}
      <rect fill="#3c3b6e" height="13" width="12" />
      {[2, 6, 10].flatMap((y) =>
        [2, 6, 10].map((x) => (
          <circle cx={x} cy={y} fill="#fff" key={`${x}-${y}`} r="0.8" />
        )),
      )}
    </svg>
  );
}

function EuropeanUnionFlag() {
  return (
    <svg viewBox="0 0 24 24">
      <rect fill="#003399" height="24" width="24" />
      {Array.from({ length: 12 }, (_, index) => {
        const angle = (index / 12) * Math.PI * 2;
        return (
          <circle
            cx={12 + Math.sin(angle) * 6}
            cy={12 - Math.cos(angle) * 6}
            fill="#ffcc00"
            key={index}
            r="0.75"
          />
        );
      })}
    </svg>
  );
}
