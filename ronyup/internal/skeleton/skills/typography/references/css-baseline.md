# Typography CSS Baseline

Drop-in CSS foundation for fluid/fixed type scales, measure, vertical rhythm,
and OpenType features.

```css
:root {
  --font-sans: "Plus Jakarta Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  --font-display: "Instrument Serif", Georgia, serif;
  --font-mono: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace;

  /* Fluid scale (Persuade / Read surfaces; use fixed rem on Operate dashboards) */
  --text-xs: 0.75rem;                                      /* 12px floor */
  --text-sm: 0.875rem;                                     /* 14px UI default */
  --text-base: 1rem;                                       /* 16px prose & mobile inputs */
  --text-lg: clamp(1.0625rem, 1rem + 0.25vw, 1.1875rem);   /* 17px → 19px */
  --text-xl: clamp(1.25rem, 1.15rem + 0.5vw, 1.5rem);      /* 20px → 24px */
  --text-2xl: clamp(1.5rem, 1.3rem + 1vw, 2rem);           /* 24px → 32px */
  --text-3xl: clamp(1.875rem, 1.5rem + 1.8vw, 2.75rem);    /* 30px → 44px */
  --text-display: clamp(2.5rem, 1.8rem + 3.2vw, 4.25rem);  /* 40px → 68px */
}

html {
  font-family: var(--font-sans);
  font-size: 100%;
  font-optical-sizing: auto;
  font-kerning: normal;
  font-variant-ligatures: common-ligatures contextual;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-size-adjust: 100%;
}

body {
  font-size: var(--text-base);
  line-height: 1.6;
}

h1, h2, h3, h4 {
  text-wrap: balance;
  line-height: 1.2;
  letter-spacing: -0.02em;
  font-weight: 600;
}

h1 {
  font-size: var(--text-3xl);
  line-height: 1.1;
  letter-spacing: -0.025em;
}

/* Asymmetric vertical rhythm in prose */
.prose :is(h2, h3, h4) {
  margin-block-start: 2em;
  margin-block-end: 0.5em;
}

.prose p,
.prose li {
  max-width: 65ch;
  text-wrap: pretty;
  overflow-wrap: break-word;
  margin-block-end: 1em;
}

/* Numeric & badge utilities */
.tabular-nums {
  font-variant-numeric: tabular-nums;
}

.code-id {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums slashed-zero;
}

.eyebrow {
  font-size: var(--text-xs);
  font-weight: 600;
  line-height: 1.25;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

@supports (text-box: trim-both cap alphabetic) {
  .optical-badge {
    text-box: trim-both cap alphabetic;
  }
}
```
