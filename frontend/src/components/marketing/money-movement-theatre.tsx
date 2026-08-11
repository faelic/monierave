"use client";

import {
  ArrowDownLeft,
  ArrowRight,
  ArrowUpRight,
  Check,
  CircleDollarSign,
  Clock3,
  ListTree,
  WalletCards,
  X,
} from "lucide-react";
import { useEffect, useState } from "react";

import styles from "./money-movement-theatre.module.css";

const views = [
  {
    id: "balances",
    eyebrow: "Account view",
    title: "Posted balances",
    description: "Know what the ledger has completed.",
    icon: WalletCards,
  },
  {
    id: "movement",
    eyebrow: "Transfer view",
    title: "Money movement",
    description: "Review value before and after it moves.",
    icon: CircleDollarSign,
  },
  {
    id: "records",
    eyebrow: "Activity view",
    title: "Transfer records",
    description: "Follow every outcome without ambiguity.",
    icon: ListTree,
  },
] as const;

type ViewId = (typeof views)[number]["id"];
type RecordOutcome = "posted" | "failed";

const navigation = ["Overview", "Accounts", "Transfer", "Activity"] as const;

function BalanceView() {
  return (
    <div className={styles.balanceView} data-theatre-panel="balances">
      <div className={styles.panelHeading}>
        <div>
          <p>Accounts</p>
          <h3>Current posted balances</h3>
        </div>
        <span>Updated just now</span>
      </div>

      <div className={styles.accountGrid}>
        <article className={styles.accountCard}>
          <div className={styles.accountIdentity}>
            <span aria-hidden="true">M</span>
            <div>
              <p>Everyday USD</p>
              <small>Account ••••••1756</small>
            </div>
          </div>
          <p className={styles.balanceLabel}>Current posted balance</p>
          <strong data-financial-number>$12,480.00</strong>
          <span className={styles.accountStatus}>Active</span>
        </article>

        <article className={styles.accountCard}>
          <div className={styles.accountIdentity}>
            <span aria-hidden="true">M</span>
            <div>
              <p>Reserve USD</p>
              <small>Account ••••••6048</small>
            </div>
          </div>
          <p className={styles.balanceLabel}>Current posted balance</p>
          <strong data-financial-number>$4,260.00</strong>
          <span className={styles.accountStatus}>Active</span>
        </article>
      </div>

      <div className={styles.ledgerNote}>
        <Check aria-hidden="true" />
        <div>
          <strong>Backed by completed ledger postings</strong>
          <p>
            Pending transfers do not change these figures until their postings
            complete.
          </p>
        </div>
      </div>
    </div>
  );
}

function MovementView() {
  return (
    <div className={styles.movementView} data-theatre-panel="movement">
      <div className={styles.panelHeading}>
        <div>
          <p>Internal transfer</p>
          <h3>See the value move</h3>
        </div>
        <span className={styles.zeroFee}>Fee $0.00</span>
      </div>

      <div className={styles.transferStage}>
        <article className={styles.transferAccount}>
          <span className={styles.transferDirection}>
            <ArrowUpRight aria-hidden="true" /> From
          </span>
          <p>Everyday USD</p>
          <strong data-financial-number>$12,480.00</strong>
          <small>Account ••••1756</small>
        </article>

        <div aria-hidden="true" className={styles.valueRoute}>
          <span className={styles.routeLine} />
          <span className={styles.valuePacket}>$250</span>
          <ArrowRight className={styles.routeArrow} />
        </div>

        <article className={styles.transferAccount}>
          <span className={styles.transferDirection}>
            <ArrowDownLeft aria-hidden="true" /> To
          </span>
          <p>Jordan A.</p>
          <strong data-financial-number>•••• 9042</strong>
          <small>Verified Monierave recipient</small>
        </article>
      </div>

      <dl className={styles.transferSummary}>
        <div>
          <dt>Amount</dt>
          <dd data-financial-number>$250.00</dd>
        </div>
        <div>
          <dt>Currency</dt>
          <dd>USD</dd>
        </div>
        <div>
          <dt>Fee</dt>
          <dd data-financial-number>$0.00</dd>
        </div>
        <div>
          <dt>Total</dt>
          <dd data-financial-number>$250.00</dd>
        </div>
      </dl>
    </div>
  );
}

function RecordsView() {
  const [outcome, setOutcome] = useState<RecordOutcome>("posted");
  const isPosted = outcome === "posted";

  return (
    <div className={styles.recordsView} data-theatre-panel="records">
      <div className={styles.panelHeading}>
        <div>
          <p>Transfer TRF-84A2</p>
          <h3>One record, every state</h3>
        </div>
        <div
          aria-label="Transfer record example"
          className={styles.outcomeSwitch}
        >
          <button
            aria-pressed={isPosted}
            onClick={() => setOutcome("posted")}
            type="button"
          >
            Posted
          </button>
          <button
            aria-pressed={!isPosted}
            onClick={() => setOutcome("failed")}
            type="button"
          >
            Failed
          </button>
        </div>
      </div>

      <div className={styles.recordLayout} data-record-outcome={outcome}>
        <ol className={styles.timeline}>
          <li data-state="complete">
            <span className={styles.timelineNode}>
              <Check aria-hidden="true" />
            </span>
            <div>
              <strong>Intent reviewed</strong>
              <p>Recipient, amount, currency, fee, and total confirmed.</p>
            </div>
            <time>10:42:08</time>
          </li>
          <li data-state="processing">
            <span className={styles.timelineNode}>
              <Clock3 aria-hidden="true" />
            </span>
            <div>
              <strong>Pending</strong>
              <p>Monierave is processing the transfer atomically.</p>
            </div>
            <time>10:42:09</time>
          </li>
          <li data-state={outcome}>
            <span className={styles.timelineNode}>
              {isPosted ? (
                <Check aria-hidden="true" />
              ) : (
                <X aria-hidden="true" />
              )}
            </span>
            <div>
              <strong>{isPosted ? "Posted" : "Failed"}</strong>
              <p>
                {isPosted
                  ? "Both ledger postings completed and balances now reflect the transfer."
                  : "No money moved and neither account balance changed."}
              </p>
            </div>
            <time>10:42:10</time>
          </li>
        </ol>

        <aside className={styles.recordSummary}>
          <p>Record summary</p>
          <strong data-financial-number>$250.00 USD</strong>
          <dl>
            <div>
              <dt>Recipient</dt>
              <dd>Jordan A.</dd>
            </div>
            <div>
              <dt>Fee</dt>
              <dd>$0.00</dd>
            </div>
            <div>
              <dt>Outcome</dt>
              <dd data-outcome={outcome}>{isPosted ? "Posted" : "Failed"}</dd>
            </div>
          </dl>
        </aside>
      </div>
    </div>
  );
}

function ProductPanel({ activeView }: { activeView: ViewId }) {
  return (
    <div className={styles.productFrame}>
      <div className={styles.frameBar}>
        <div aria-hidden="true" className={styles.windowControls}>
          <span />
          <span />
          <span />
        </div>
        <p>Monierave workspace</p>
        <span className={styles.secureState}>
          <span aria-hidden="true" /> Secure session
        </span>
      </div>

      <div className={styles.workspace}>
        <aside
          aria-label="Product preview navigation"
          className={styles.sidebar}
        >
          <span className={styles.sidebarMark}>M</span>
          <nav>
            {navigation.map((item) => (
              <span
                data-active={
                  (activeView === "balances" && item === "Accounts") ||
                  (activeView === "movement" && item === "Transfer") ||
                  (activeView === "records" && item === "Activity")
                    ? "true"
                    : "false"
                }
                key={item}
              >
                {item}
              </span>
            ))}
          </nav>
          <span className={styles.sidebarUser}>FA</span>
        </aside>

        <div className={styles.panel} key={activeView}>
          {activeView === "balances" ? <BalanceView /> : null}
          {activeView === "movement" ? <MovementView /> : null}
          {activeView === "records" ? <RecordsView /> : null}
        </div>
      </div>
    </div>
  );
}

export function MoneyMovementTheatre() {
  const [activeView, setActiveView] = useState<ViewId>("balances");
  const [userSelected, setUserSelected] = useState(false);

  useEffect(() => {
    if (userSelected) return;

    const prefersReducedMotion =
      typeof window.matchMedia === "function" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (prefersReducedMotion) return;

    const interval = window.setInterval(() => {
      setActiveView((current) => {
        const index = views.findIndex((view) => view.id === current);
        return views[(index + 1) % views.length]!.id;
      });
    }, 7600);

    return () => window.clearInterval(interval);
  }, [userSelected]);

  const selectView = (view: ViewId) => {
    setUserSelected(true);
    setActiveView(view);
  };

  return (
    <section
      aria-labelledby="money-movement-title"
      className={styles.section}
      id="money-movement"
    >
      <div className={styles.inner}>
        <div className={styles.headingBlock}>
          <p className={styles.eyebrow}>Money, made visible</p>
          <h2 className={styles.heading} id="money-movement-title">
            See what changes when money moves.
          </h2>
          <p className={styles.introduction}>
            Follow posted balances, recipient details, fees, and transfer
            outcomes through one clear record.
          </p>
        </div>

        <div
          aria-label="Money movement demonstrations"
          className={styles.tabs}
          role="tablist"
        >
          {views.map((view, index) => {
            const Icon = view.icon;
            const active = activeView === view.id;

            return (
              <button
                aria-controls="money-movement-product-panel"
                aria-selected={active}
                className={styles.tab}
                data-active={active ? "true" : "false"}
                id={`money-movement-tab-${view.id}`}
                key={view.id}
                onClick={() => selectView(view.id)}
                role="tab"
                type="button"
              >
                <span className={styles.tabNumber}>0{index + 1}</span>
                <Icon aria-hidden="true" />
                <span>
                  <small>{view.eyebrow}</small>
                  <strong>{view.title}</strong>
                  <span>{view.description}</span>
                </span>
              </button>
            );
          })}
        </div>

        <div
          aria-labelledby={`money-movement-tab-${activeView}`}
          id="money-movement-product-panel"
          role="tabpanel"
        >
          <ProductPanel activeView={activeView} />
        </div>
      </div>
    </section>
  );
}
