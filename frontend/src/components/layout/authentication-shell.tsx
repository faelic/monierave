import { ArrowLeft } from "lucide-react";
import Link from "next/link";

import { AuthenticationAtmosphere } from "@/components/auth/authentication-atmosphere";
import { SkipLink } from "@/components/ui/skip-link";

import styles from "./authentication-shell.module.css";

export function AuthenticationShell({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className={styles.shell}>
      <SkipLink />
      <div className={styles.atmosphere} data-auth-atmosphere>
        <AuthenticationAtmosphere />
      </div>
      <Link className={styles.homeLink} href="/">
        <ArrowLeft aria-hidden="true" className="size-4" />
        Home
      </Link>
      <main className={styles.main} id="main-content">
        <div className={styles.content}>
          <div className={styles.brand}>
            <Link
              aria-label="Monierave home"
              className={styles.brandLink}
              href="/"
            >
              <svg aria-hidden="true" fill="none" viewBox="0 0 32 32">
                <rect
                  height="26"
                  rx="8"
                  stroke="currentColor"
                  strokeWidth="2"
                  width="26"
                  x="3"
                  y="3"
                />
                <path
                  d="M9 21V11l7 6 7-6v10"
                  stroke="currentColor"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2.25"
                />
                <path
                  d="M11 25h10"
                  stroke="currentColor"
                  strokeLinecap="round"
                  strokeWidth="2"
                />
              </svg>
            </Link>
          </div>
          {children}
        </div>
      </main>
    </div>
  );
}
