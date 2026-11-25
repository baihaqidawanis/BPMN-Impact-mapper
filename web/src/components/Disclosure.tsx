"use client";

import * as Collapsible from "@radix-ui/react-collapsible";
import type { ReactNode } from "react";
import styles from "./Disclosure.module.css";

type DisclosureProps = {
  title: string;
  openLabel?: string;
  closeLabel?: string;
  className?: string;
  children: ReactNode;
};

export function Disclosure({ title, openLabel = "Show details", closeLabel = "Hide details", className = "", children }: DisclosureProps) {
  return (
    <Collapsible.Root className={className}>
      <Collapsible.Trigger type="button" className={styles.trigger}>
        <span>{title}</span>
        <span className={styles.action}>
          <span className={styles.closed}>{openLabel}</span>
          <span className={styles.open}>{closeLabel}</span>
          <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true"><path d="m5 8 5 5 5-5" /></svg>
        </span>
      </Collapsible.Trigger>
      <Collapsible.Content className={styles.content}>{children}</Collapsible.Content>
    </Collapsible.Root>
  );
}
