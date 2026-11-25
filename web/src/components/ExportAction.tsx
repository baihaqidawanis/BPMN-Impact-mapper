import type { ReactNode } from "react";
import styles from "./ExportAction.module.css";

type Props = { children:ReactNode; variant?:"primary"|"secondary" } & ({href:string;onClick?:never}|{href?:never;onClick:()=>void});

export function ExportAction({ children, variant="secondary", href, onClick }:Props) {
 const className=`${styles.action} ${styles[variant]}`;
 const content=<><svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M10 3v9m-3-3 3 3 3-3M4 12v4h12v-4"/></svg><span>{children}</span></>;
 return href?<a href={href} className={className}>{content}</a>:<button type="button" onClick={onClick} className={className}>{content}</button>;
}
