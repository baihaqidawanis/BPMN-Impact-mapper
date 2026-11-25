"use client";

import { useEffect, useRef, type ReactNode } from "react";
import { gsap } from "gsap";

/** Local, deliberately small interpretation of React Bits' FadeContent pattern.
 * Content is visible without JS; no ScrollTrigger or decorative effects.
 */
export function FadeContent({ children, className, revision, scrollRequest = 0 }: {
  children: ReactNode;
  className?: string;
  revision: string;
  scrollRequest?: number;
}) {
  const container = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const element = container.current;
    if (!element || !element.getClientRects().length) return;
    const media = gsap.matchMedia();
    media.add("(prefers-reduced-motion: no-preference)", () => {
      gsap.fromTo(element, { opacity: 0.72, y: 6 }, {
        opacity: 1, y: 0, duration: 0.28, ease: "power3.out",
        clearProps: "opacity,transform",
      });
      const rows = element.querySelectorAll('[data-testid="impact-details"]');
      if (rows.length > 0 && rows.length <= 24) {
        gsap.fromTo(rows, { opacity: 0.4, y: 5 }, {
          opacity: 1, y: 0, duration: 0.26, stagger: 0.022, ease: "power2.out",
          clearProps: "opacity,transform",
        });
      }
    }, element);
    return () => media.revert();
  }, [revision]);

  useEffect(() => {
    if (!scrollRequest) return;
    const element = container.current;
    if (!element || !element.getClientRects().length) return;
    // Leave the viewport alone when the new result is already readable.
    const top = element.getBoundingClientRect().top;
    if (top >= 140 && top < window.innerHeight - 160) return;
    const preference = window.matchMedia("(prefers-reduced-motion: reduce)");
    if (preference.matches) {
      window.scrollTo({ top: Math.max(0, window.scrollY + top - 148), behavior: "instant" });
      return;
    }

    let disposed = false;
    let frame = 0;
    let disposeScroll: (() => void) | undefined;
    // Load Lenis only for intentional navigation, never for ordinary scrolling.
    void import("lenis").then(({ default: Lenis }) => {
      if (disposed) return;
      const scroll = new Lenis({ smoothWheel: false, syncTouch: false, autoRaf: false });
      let stopped = false;
      const interrupt = () => {
        if (stopped) return;
        stopped = true;
        cancelAnimationFrame(frame);
        scroll.destroy();
        window.removeEventListener("wheel", interrupt);
        window.removeEventListener("touchstart", interrupt);
        window.removeEventListener("keydown", interrupt);
        preference.removeEventListener("change", interrupt);
      };
      disposeScroll = interrupt;
      window.addEventListener("wheel", interrupt, { passive: true });
      window.addEventListener("touchstart", interrupt, { passive: true });
      window.addEventListener("keydown", interrupt);
      preference.addEventListener("change", interrupt);
      const started = performance.now();
      const tick = (time: number) => {
        if (disposed || stopped) return;
        scroll.raf(time);
        if (time - started < 700) frame = requestAnimationFrame(tick);
        else interrupt();
      };
      scroll.scrollTo(element, { offset: -148, duration: 0.38 });
      frame = requestAnimationFrame(tick);
    }).catch(() => {
      if (!disposed) window.scrollTo({ top: Math.max(0, window.scrollY + top - 148), behavior: "instant" });
    });
    return () => {
      disposed = true;
      cancelAnimationFrame(frame);
      disposeScroll?.();
    };
  }, [scrollRequest]);

  return <div ref={container} className={className}>{children}</div>;
}
