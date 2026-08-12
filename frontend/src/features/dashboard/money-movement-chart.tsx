"use client";

import { useState } from "react";

import { formatMinorAmount } from "@/features/dashboard/financial-format";
import type { MoneyMovement } from "@/lib/api/contracts";

const chartWidth = 760;
const chartHeight = 248;
const plotTop = 20;
const baseline = 202;

export function MoneyMovementChart({ data }: { data: MoneyMovement }) {
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const activeBuckets = data.buckets.filter(
    (bucket) => bucket.incoming > 0 || bucket.outgoing > 0,
  );

  if (activeBuckets.length === 0) {
    return (
      <div className="border-line-200 grid min-h-64 place-items-center rounded-md border border-dashed text-center">
        <div>
          <p className="font-semibold">No posted movement in this period.</p>
          <p className="text-ink-600 mt-1 text-sm">
            Incoming and outgoing activity will appear here after it is posted.
          </p>
        </div>
      </div>
    );
  }

  const maximum = Math.max(
    1,
    ...data.buckets.flatMap((bucket) => [bucket.incoming, bucket.outgoing]),
  );
  const groupWidth = chartWidth / data.buckets.length;
  const barWidth = Math.min(15, Math.max(4, groupWidth * 0.3));
  const verticalSpace = baseline - plotTop - 8;
  const heightFor = (amount: number) => (amount / maximum) * verticalSpace;
  const labelIndexes = new Set([
    0,
    Math.floor((data.buckets.length - 1) / 2),
    data.buckets.length - 1,
  ]);
  const firstActiveIndex = data.buckets.findIndex(
    (bucket) => bucket.incoming > 0 || bucket.outgoing > 0,
  );
  const lastActiveIndex = data.buckets.findLastIndex(
    (bucket) => bucket.incoming > 0 || bucket.outgoing > 0,
  );
  const selectedBucket =
    activeIndex === null ? null : data.buckets[activeIndex];

  return (
    <div>
      <p className="text-ink-600 mb-3 text-sm">
        {movementInsight(activeBuckets)}
      </p>
      <div className="relative">
        <svg
          aria-label={`Posted money in and money out for the selected ${data.currency} account`}
          className="h-auto w-full overflow-visible"
          role="img"
          viewBox={`0 0 ${chartWidth} ${chartHeight}`}
        >
          <defs>
            <linearGradient id="movement-in" x1="0" x2="0" y1="0" y2="1">
              <stop offset="0" stopColor="#65e7a8" />
              <stop offset="1" stopColor="var(--success-700)" />
            </linearGradient>
            <linearGradient id="movement-out" x1="0" x2="0" y1="0" y2="1">
              <stop offset="0" stopColor="var(--product-accent)" />
              <stop offset="1" stopColor="#8f8153" />
            </linearGradient>
            <filter
              id="movement-glow"
              height="180%"
              width="180%"
              x="-40%"
              y="-40%"
            >
              <feGaussianBlur stdDeviation="5" />
            </filter>
          </defs>

          {firstActiveIndex >= 0 ? (
            <rect
              fill="var(--product-accent-soft)"
              height={baseline - plotTop}
              opacity="0.4"
              rx="12"
              width={(lastActiveIndex - firstActiveIndex + 1) * groupWidth}
              x={firstActiveIndex * groupWidth}
              y={plotTop}
            />
          ) : null}

          {[plotTop, plotTop + (baseline - plotTop) / 2].map((y) => (
            <line
              key={y}
              stroke="var(--product-border)"
              strokeDasharray="3 8"
              x1="0"
              x2={chartWidth}
              y1={y}
              y2={y}
            />
          ))}
          <line
            stroke="var(--product-border-strong)"
            strokeWidth="1.5"
            x1="0"
            x2={chartWidth}
            y1={baseline}
            y2={baseline}
          />

          {data.buckets.map((bucket, index) => {
            const center = groupWidth * index + groupWidth / 2;
            const incomingHeight = heightFor(bucket.incoming);
            const outgoingHeight = heightFor(bucket.outgoing);
            const hasMovement = bucket.incoming > 0 || bucket.outgoing > 0;
            return (
              <g key={bucket.start}>
                {activeIndex === index && hasMovement ? (
                  <line
                    stroke="var(--product-border-strong)"
                    strokeDasharray="3 4"
                    x1={center}
                    x2={center}
                    y1={plotTop}
                    y2={baseline}
                  />
                ) : null}
                {incomingHeight > 0 ? (
                  <>
                    <rect
                      fill="#45d58a"
                      filter="url(#movement-glow)"
                      height={incomingHeight}
                      opacity="0.2"
                      rx={barWidth / 2}
                      width={barWidth + 8}
                      x={center - barWidth - 6}
                      y={baseline - incomingHeight}
                    />
                    <rect
                      fill="url(#movement-in)"
                      height={incomingHeight}
                      rx={barWidth / 2}
                      width={barWidth}
                      x={center - barWidth - 2}
                      y={baseline - incomingHeight}
                    />
                  </>
                ) : null}
                {outgoingHeight > 0 ? (
                  <>
                    <rect
                      fill="var(--product-accent)"
                      filter="url(#movement-glow)"
                      height={outgoingHeight}
                      opacity="0.16"
                      rx={barWidth / 2}
                      width={barWidth + 8}
                      x={center - 2}
                      y={baseline - outgoingHeight}
                    />
                    <rect
                      fill="url(#movement-out)"
                      height={outgoingHeight}
                      rx={barWidth / 2}
                      width={barWidth}
                      x={center + 2}
                      y={baseline - outgoingHeight}
                    />
                  </>
                ) : null}
                <rect
                  aria-label={bucketLabel(bucket, data.currency)}
                  fill="transparent"
                  height={baseline - plotTop}
                  onBlur={() => setActiveIndex(null)}
                  onFocus={() => setActiveIndex(index)}
                  onMouseEnter={() => setActiveIndex(index)}
                  onMouseLeave={() => setActiveIndex(null)}
                  role="button"
                  tabIndex={hasMovement ? 0 : -1}
                  width={groupWidth}
                  x={index * groupWidth}
                  y={plotTop}
                />
                {labelIndexes.has(index) ? (
                  <text
                    fill="var(--product-text-muted)"
                    fontSize="11"
                    textAnchor="middle"
                    x={center}
                    y="238"
                  >
                    {formatChartDate(bucket.start)}
                  </text>
                ) : null}
              </g>
            );
          })}
        </svg>

        {selectedBucket ? (
          <div
            className="border-line-300 pointer-events-none absolute top-3 z-10 w-44 -translate-x-1/2 rounded-md border bg-[var(--product-surface-raised)] p-3 shadow-xl"
            style={{
              left: `${Math.min(
                88,
                Math.max(
                  12,
                  ((activeIndex! + 0.5) / data.buckets.length) * 100,
                ),
              )}%`,
            }}
          >
            <p className="text-ink-600 text-xs font-semibold">
              {formatChartDate(selectedBucket.start)}
            </p>
            <dl className="mt-2 space-y-1.5 text-sm">
              <TooltipAmount
                amount={selectedBucket.incoming}
                currency={data.currency}
                label="Money in"
              />
              <TooltipAmount
                amount={selectedBucket.outgoing}
                currency={data.currency}
                label="Money out"
              />
              <TooltipAmount
                amount={selectedBucket.incoming - selectedBucket.outgoing}
                currency={data.currency}
                label="Net"
                signed
              />
            </dl>
          </div>
        ) : null}
      </div>

      <table className="sr-only">
        <caption>Posted money movement by period</caption>
        <thead>
          <tr>
            <th>Period</th>
            <th>Money in</th>
            <th>Money out</th>
          </tr>
        </thead>
        <tbody>
          {data.buckets.map((bucket) => (
            <tr key={bucket.start}>
              <td>{formatChartDate(bucket.start)}</td>
              <td>{formatMinorAmount(bucket.incoming, data.currency)}</td>
              <td>{formatMinorAmount(bucket.outgoing, data.currency)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TooltipAmount({
  amount,
  currency,
  label,
  signed = false,
}: {
  amount: number;
  currency: MoneyMovement["currency"];
  label: string;
  signed?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="text-ink-600">{label}</dt>
      <dd className="font-semibold" data-financial-number>
        {signed && amount >= 0 ? "+" : ""}
        {formatMinorAmount(amount, currency)}
      </dd>
    </div>
  );
}

function movementInsight(activeBuckets: MoneyMovement["buckets"]): string {
  if (activeBuckets.length === 1) {
    return `All movement in this period occurred on ${formatChartDate(activeBuckets[0]!.start)}.`;
  }
  const busiest = activeBuckets.reduce((current, bucket) =>
    bucket.incoming + bucket.outgoing > current.incoming + current.outgoing
      ? bucket
      : current,
  );
  return `The busiest movement period was ${formatChartDate(busiest.start)}.`;
}

function bucketLabel(
  bucket: MoneyMovement["buckets"][number],
  currency: MoneyMovement["currency"],
) {
  return `${formatChartDate(bucket.start)}: ${formatMinorAmount(bucket.incoming, currency)} in, ${formatMinorAmount(bucket.outgoing, currency)} out`;
}

function formatChartDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    day: "numeric",
    month: "short",
  }).format(new Date(value));
}
