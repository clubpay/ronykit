# HTML Entity & Character Reference

Read this to look up the correct entity for a character, or when auditing HTML
for incorrect characters. (JSX text needs real characters or a `{'\u2019'}`
expression — entities don't render in JSX text; see `SKILL.md`.)

## Quick substitution table

| If you see | Use | Entity | Rule |
| ---------- | --- | ------ | ---- |
| `"…"` straight | "…" | `&ldquo;` `&rdquo;` | Curly double quotes |
| `'…'` straight | '…' | `&lsquo;` `&rsquo;` | Curly single quotes |
| `it's` | it's | `&rsquo;` | Apostrophe = closing single |
| `--` | – | `&ndash;` | En dash, ranges |
| `---` | — | `&mdash;` | Em dash, sentence breaks |
| `...` | … | `&hellip;` | One ellipsis character |
| `(c)` | © | `&copy;` | Real copyright |
| `(TM)` | ™ | `&trade;` | Real trademark |
| `(R)` | ® | `&reg;` | Real registered |
| `12 x 34` | 12 × 34 | `&times;` | Multiplication sign |
| `56 - 12` (math) | 56 − 12 | `&minus;` | Minus sign |
| `6' 10"` (measure) | 6' 10" | `&#39;` `&quot;` | Foot/inch stay straight |

## Quotes & apostrophes

```
&ldquo;  “  U+201C  opening double quote
&rdquo;  ”  U+201D  closing double quote
&lsquo;  ‘  U+2018  opening single quote
&rsquo;  ’  U+2019  closing single quote / apostrophe
&quot;   "  U+0022  straight double (inch mark only)
&#39;    '  U+0027  straight single (foot mark only)
```

## Dashes

```
-           U+002D  hyphen (compound words, line breaks)
&ndash;  –  U+2013  en dash (ranges 1–10, connections Sarbanes–Oxley)
&mdash;  —  U+2014  em dash (sentence breaks—like this)
&shy;       U+00AD  soft hyphen (invisible break hint)
```

## Symbols

```
&hellip;  …  U+2026  ellipsis
&times;   ×  U+00D7  multiplication
&minus;   −  U+2212  minus
&divide;  ÷  U+00F7  division
&plusmn;  ±  U+00B1  plus-minus
&copy;    ©  U+00A9  copyright
&trade;   ™  U+2122  trademark
&reg;     ®  U+00AE  registered
&sect;    §  U+00A7  section mark
&para;    ¶  U+00B6  pilcrow
&deg;     °  U+00B0  degree
&amp;     &  U+0026  ampersand (proper names only; write "and" in body)
&bull;    •  U+2022  bullet
```

## Spaces

```
&nbsp;     U+00A0  nonbreaking space (prevents a line break)
&thinsp;   U+2009  thin space (half a word space)
&ensp;     U+2002  en space
&emsp;     U+2003  em space
&hairsp;   U+200A  hair space (thinnest)
```

## Primes (foot / inch / minute / second)

```
&#39;     '  U+0027  foot / minute (straight single)
&quot;    "  U+0022  inch / second (straight double)
&prime;   ′  U+2032  true prime (if the font has it)
&Prime;   ″  U+2033  true double prime
```

## Common accented characters

Accents in proper names are mandatory (François, Plácido, Köln).

```
&eacute; é   &egrave; è   &aacute; á   &agrave; à
&iacute; í   &oacute; ó   &uacute; ú   &ntilde; ñ
&uuml;   ü   &ouml;   ö   &ccedil; ç   &szlig;  ß
```

## Usage examples

```html
<p>&ldquo;She said &lsquo;hello&rsquo; to me,&rdquo; he reported.</p>
<p>In the &rsquo;70s, rock &rsquo;n&rsquo; roll dominated.</p>
<p>Pages 4&ndash;8 &middot; the Sarbanes&ndash;Oxley Act</p>
<p>The em dash adds a pause&mdash;and is underused.</p>
<p>Under &sect;&nbsp;1782, a refund may apply.</p>
<footer>&copy;&nbsp;2025 MegaCorp&trade;</footer>
<p>The room is 12&#39;&nbsp;6&quot; &times; 8&#39;&nbsp;10&quot;.</p>
```
