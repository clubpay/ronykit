# CSS Typography Baseline

Read this when generating CSS for a web project, building a design system, or
auditing stylesheets. Every property maps to a rule in `SKILL.md`. Pair the
type scale with the `design-tokens` skill.

## Baseline template

Copy-paste starting point — swap in the project's chosen faces.

```css
*, *::before, *::after { box-sizing: border-box; }

html {
  font-size: clamp(16px, 2.5vw, 20px);   /* 15–25px, fluid */
  -webkit-text-size-adjust: 100%;        /* prevent iOS resize */
}

body {
  font-family: /* chosen-body, */ Georgia, "Times New Roman", serif;
  line-height: 1.38;                     /* 120–145% sweet spot */
  text-rendering: optimizeLegibility;    /* enables kern + liga */
  font-feature-settings: "kern" 1, "liga" 1;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* Line-length control — cap every text container */
main, article, .prose {
  max-width: min(65ch, 90vw);            /* 45–90 chars */
  margin-inline: auto;
  padding-inline: clamp(1rem, 4vw, 2rem);
}

/* Paragraphs — space between OR first-line indent, never both */
p { margin: 0 0 0.75em; }                /* default for web */
/* p + p { text-indent: 1.5em; margin: 0; }  // indent alternative */

/* Headings — bold, more space above than below, no hyphenation */
h1, h2, h3, h4 {
  line-height: 1.15;
  hyphens: none;
  font-weight: 700;
}
h1 { font-size: clamp(1.5rem, 4vw, 2.5rem);  margin: 2.5em 0 0.5em; }
h2 { font-size: clamp(1.25rem, 3vw, 1.75rem); margin: 2em 0 0.4em; }
h3 { font-size: 1.1em;                        margin: 1.5em 0 0.3em; }

/* Emphasis — bold OR italic, never combined; never underline for emphasis */
em { font-style: italic; }
strong { font-weight: 700; }

/* ALL CAPS — always letterspaced */
.caps { text-transform: uppercase; letter-spacing: 0.06em; font-feature-settings: "kern" 1; }

/* Small caps — real only (font must have smcp) */
.small-caps { font-variant-caps: small-caps; letter-spacing: 0.05em; font-feature-settings: "smcp" 1, "kern" 1; }

/* Tables — remove cell borders, keep one rule under the header, pad generously */
table { border-collapse: collapse; width: 100%; }
th, td { padding: 0.5em 1em; text-align: left; vertical-align: top; border: none; }
thead th { border-bottom: 1.5px solid currentColor; font-weight: 600; }
.data-table td { font-variant-numeric: tabular-nums lining-nums; }  /* numeric columns */

/* Links — subtle underline, not heavy */
a { color: inherit; text-decoration-line: underline; text-decoration-thickness: 1px; text-underline-offset: 2px; }

/* Horizontal rule — thin, no patterns */
hr { border: none; border-top: 1px solid currentColor; opacity: 0.3; margin: 2em 0; }

@media print {
  body { font-size: 11pt; line-height: 1.3; }
  main { max-width: none; }
  h1, h2, h3 { page-break-after: avoid; }
  p { orphans: 2; widows: 2; }
}
```

## Fluid type with clamp

```css
body { font-size: clamp(16px, 2.5vw, 20px); }
main { max-width: min(65ch, 90vw); margin-inline: auto; padding-inline: clamp(1rem, 4vw, 2rem); }
h1   { font-size: clamp(1.5rem, 4vw, 2.5rem); }
```

Principles: scale `font-size` and container `width` together; always
`max-width` on text; don't rely on `ch` for *exact* measure (it sizes to the
`0` glyph); keep ≥ `1rem` inline padding on mobile.

## OpenType features

```css
.body       { font-feature-settings: "kern" 1, "liga" 1, "calt" 1; }
.prose      { font-feature-settings: "kern" 1, "liga" 1, "onum" 1; }  /* oldstyle figures */
.data-table { font-feature-settings: "kern" 1, "tnum" 1, "lnum" 1; }  /* tabular lining figures */
.small-caps { font-feature-settings: "kern" 1, "smcp" 1; }
```

## Dark mode

```css
@media (prefers-color-scheme: dark) {
  body {
    color: #e0e0e0;
    background: #1a1a1a;
    font-weight: 350;            /* dark bg makes type look heavier */
    -webkit-font-smoothing: auto;
  }
}
```

## React/JSX inline pattern

Apply entities (or real characters) directly in JSX text — see the JSX trap in
`SKILL.md`.

```jsx
// WRONG
<p>"Hello," she said. "It's a beautiful day..."</p>
<p>Price: $12 x 4 = $48</p>
<p>Pages 1-10</p>

// RIGHT
<p>&ldquo;Hello,&rdquo; she said. &ldquo;It&rsquo;s a beautiful day&hellip;&rdquo;</p>
<p>Price: $12 &times; 4 = $48</p>
<p>Pages 1&ndash;10</p>
```
