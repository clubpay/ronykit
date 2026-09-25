# Procedures and Code

This file owns **P1–P11**: step-by-step instructions, UI verbs, code in prose, code samples, placeholders, and command-line syntax. W1 (condition before instruction) belongs to [voice-and-words.md](voice-and-words.md); S12 (the bold/code-font/italics summary) belongs to [structure-and-formatting.md](structure-and-formatting.md); A1–A7 (API reference text, docstrings) belong to [api-reference.md](api-reference.md) — mentioned here only where a procedure or sample touches them.

## 1. Anatomy of a Procedure (P1–P4)

A procedure exists to get one reader from a starting state to a finished one. Every step names a single imperative action, states or implies where to act, and lets the reader confirm what happened before moving on.

**Rules:**
- **P1 — Numbered steps, one action each.** Split "click X and then configure Y" into two steps. If a step has two actions, the reader can't tell which one failed.
- **Condition before instruction (W1, owned by voice-and-words.md).** "If you want X, do Y" — not "Do Y if you want X." The reader decides whether the step applies before reading how to do it.
- **P4 — Where to act, what to expect.** Name the screen, menu, or file the action happens in, and state the observable result: a new status, a redirect, a file on disk.
- **P3 — "Optional:" prefix.** Not "(Optional)", not "You can also…" — the word "Optional:" at the start of the step, so a scanning reader can skip it without reading the rest.
- **P2 — Single step = bullet.** A procedure with exactly one action is a bullet (`*` or `-`), never "1." — a numbered list of one implies steps 2 and 3 are coming.
- **P2 — Sub-steps a/b/c, then i/ii/iii.** Use letters for a sub-sequence inside a numbered step; drop to roman numerals only when a lettered sub-step itself has to branch — for example, by operating system. Don't add a third level to document a style preference.
- **Document the shortest path.** When the UI and the CLI both accomplish the task, pick one and mention the other in a single sentence, not as a parallel procedure.

**Before:**

```markdown
To turn on automatic deployments, you should first go to the project
settings and look for the Deployments tab, then enable it, and after
that you can pick which branch you want it to watch. You should also
set a build command if you have one, and once you're done you can
save it, and it will deploy automatically from then on when you push
to that branch.
```

**After:**

```markdown
1. In the **shipit** console, open your project and click **Settings**.
2. Click **Deployments**.
3. Turn on **Automatic deployments**.
4. Select the branch to watch:
   a. Click the **Branch** menu.
   b. Choose the branch, for example `main`.
5. Optional: Enter a build command, for example `pnpm build`.
6. Click **Save**. The **Status** column shows **Watching** next to
   the selected branch.
```

A step that must branch by platform drops to roman numerals inside the lettered sub-step:

```markdown
1. Open a terminal.
   a. On macOS or Linux, run `chmod +x shipit`.
      i. If `shipit` isn't on your `PATH`, move it to `/usr/local/bin`.
      ii. If it is, skip to step 2.
   b. On Windows, skip this step.
```

A single-action procedure stays a bullet:

```markdown
To restart the deployment watcher:

* Click **Restart** in the **Deployments** panel.
```

Source: procedures

## 2. UI Elements and Device Verbs (P5)

**Rules:**
- **Bold the element's name**, not its type: "Click **Save**", not "Click the **Save** button." Name the type only when the label alone is ambiguous ("click the **Region** dropdown" when a section is also called Region).
- **Casing matches the UI**, with one exception: an ALL-CAPS label in the interface is written in sentence case in prose. A button rendered `SUBMIT ORDER` is still "click **Submit order**."
- **Menu paths** bold each segment and join with `>`: "**File > Save as**" (inferred — the guide doesn't give this exact notation, but it follows directly from bolding UI element names in sequence).
- **Device verbs** match how the reader is expected to interact with the target, not the writer's own device.

| Verb | Situation | Example |
|------|-----------|---------|
| Click | Mouse or trackpad — buttons, links, checkboxes | "Click **Deploy**." |
| Tap | Touchscreen — the same targets on mobile or tablet | "Tap **Deploy**." |
| Select | Device-agnostic — menu items, dropdown options, radio buttons | "Select **Production** from the **Environment** menu." |
| Enter | Typing a value into a field | "Enter your project ID." |
| Type | Free-text input where "enter" would also read correctly — the two are interchangeable (inferred) | "Type a description for the release." |
| Choose | Picking one option among several presented together — close in meaning to "select" (inferred) | "Choose a region for the new environment." |

**Before → after:**
- "Click on the SAVE CHANGES button to save your changes." → "Click **Save changes**."
- "Go to File, then Export, then click on PDF." → "Click **File > Export > PDF**."

Source: ui-elements

## 3. Code in Text (P6)

**Gets code font:** filenames, paths, commands, flags, parameters, values, method and function names, class names, HTTP status codes, environment variables.
**Doesn't:** product names, UI labels (those are bold, per P5), and general concepts ("the deploy step," not `the deploy step`).

Code isn't a part of speech: don't inflect it as a verb or pluralize it by adding a suffix outside the backticks. Add a plain-English noun instead — "call the `close` method," not "`close`ing the file"; "`Widget` objects," not "`Widget`s."

**Before → after:**
- "Open the config.yaml file and set the timeout value to 30 seconds." → "Open `config.yaml` and set `timeout` to `30`."
- "Run shipit deploy with the dash-dash-force flag to skip confirmation." → "Run `shipit deploy --force` to skip confirmation."
- "The API returns a 404 Not Found if SHIPIT_API_KEY isn't set." → "The API returns `404 Not Found` if `SHIPIT_API_KEY` isn't set."
- "After closing() the file, check for leftover Widgets — the Shipit CLI logs them." → "After you call the `close` method, check for leftover `Widget` objects — the shipit CLI logs them." (the method and class stay in code font; the product name doesn't)

Source: code-in-text

## 4. Code Samples (P7, P10)

**Rules:**
- **P7 — Introduce every sample with a sentence ending in a colon.** A bare code block gives the reader no reason to read it and no way to know what it's for.
- **One concept per sample.** A sample that shows authentication and retry logic in one block forces a reader debugging retries to also parse auth.
- **Runnable and minimal.** Nothing in the sample should be unrelated to the concept it demonstrates; nothing needed to run it should be missing.
- **Realistic names.** Production-shaped data, not `foo`/`bar` — a reader pattern-matches against their own variables and data.
- **Wrap at 80 characters.** Long lines force horizontal scrolling in both the doc and the terminal the reader pastes into.
- **Show expected output** in a second fenced block, immediately after the sample.
- **Comments explain why**, not what the syntax already says.
- **Tag the language** on every fence (` ```go `, ` ```bash `) — untagged blocks can't be highlighted or reliably copied (convention).

**Before → after (function):**

```go
// before — no intro, mixed concepts, foo/bar, no output shown
func foo(bar map[string]any) []string {
	x := bar["data"].([]map[string]any)
	var result []string
	for _, i := range x {
		if i["active"] == true && i["score"].(int) > 50 && (i["region"] == "us" || i["region"] == "eu" || i["region"] == "apac") {
			result = append(result, i["name"].(string))
		}
	}
	return result
}
```

The following function returns the names of active users in a supported region who scored above 50:

```go
var supportedRegions = []string{"us", "eu", "apac"}

func HighScorers(users []User) []string {
	var names []string
	for _, u := range users {
		// Scoring only applies in supported regions, so filter first.
		if !slices.Contains(supportedRegions, u.Region) {
			continue
		}
		if u.Active && u.Score > 50 {
			names = append(names, u.Name)
		}
	}
	return names
}
```

```
fmt.Println(HighScorers(users))
[Priya Patel Sam Nguyen]
```

**Before → after (CLI):**

```markdown
To see your deployments just run this: shipit list-deployments --all
--verbose --since=2026-01-01 --until=2026-08-29 --format=json --pretty
| jq '.[] | select(.status=="failed")'
```

The following command lists deployments that failed since the start of the year:

```bash
shipit list-deployments \
  --since=2026-01-01 \
  --status=failed
```

```
DEPLOYMENT   BRANCH   STATUS   FINISHED
d-8f2a1c     main     failed   2026-03-14T10:02:00Z
```

Source: code-samples, tech-writing/two "Sample code"

## 5. Placeholders (P8)

A placeholder stands in for a value the reader supplies. Write it `ALL_CAPS_WITH_UNDERSCORES` — never `<your-api-key>` (angle brackets read as literal characters to a reader who doesn't know the convention) and never `MY_PROJECT_ID` or `YOUR_API_KEY` (the prefix reads as part of the name). After the sample, list what each placeholder means in a "Replace the following:" block.

**Before:**

```bash
shipit deploy --key=<your-api-key> --project=YOUR_PROJECT_ID
```

**After:**

Deploy your project by running the following command:

```bash
shipit deploy --key=API_KEY --project=PROJECT_ID
```

Replace the following:

- `API_KEY`: the key from the project's **Settings > API keys** page.
- `PROJECT_ID`: the project ID shown at the top of the **Overview** page.

Source: placeholders

## 6. Command-Line Syntax (P9)

| Notation | Meaning | Example |
|----------|---------|---------|
| `[optional]` | The argument may be omitted | `shipit deploy [--dry-run]` |
| `{a\|b}` | Choose exactly one | `shipit logs {--tail\|--since=DATE}` |
| `...` | The argument repeats | `shipit tag ITEM...` |
| `ALL_CAPS` | A placeholder, not a literal | `shipit init PROJECT_NAME` |

**One command per line** (convention). Don't chain unrelated commands with `&&` in a sample unless the sample's stated purpose is to show chaining.

**Long commands wrap with `\` continuation**, one flag per line, as in the CLI sample in section 4.

**Prompt characters.** Whether to show a leading `$` isn't specified in the guide (convention): this skill's default is to omit it — the fenced ` ```bash ` tag already marks the block as a shell command, and a bare `$` gets pasted verbatim by readers who don't know to drop it. Follow a project's existing samples if they already include `$` — local convention wins (see SKILL.md's precedence rule).

**Filenames** written inside docs — sample config files, script names — are lowercase, hyphenated, ASCII:

- "See Getting_Started.MD for setup instructions." → "See `getting-started.md` for setup instructions."

Source: code-syntax, filenames

## 7. Sample-Code Quality Checklist (P10)

| Check | Why | Example fix |
|-------|-----|--------------|
| Language tag on every fence | Without it, editors and readers can't syntax-highlight or copy cleanly | ` ``` ` → ` ```go ` |
| Lines wrap at 80 characters | Long lines force horizontal scrolling in docs and terminals | Break a long flag list after the first flag with `\` |
| One concept per sample | A reader debugging one idea shouldn't have to parse three | Split an auth-and-retry sample into two samples |
| Realistic names, no foo/bar | `foo`/`bar` carries no information about real data | `func foo(bar)` → `func HighScorers(users)` |
| Comments explain why, not what | "# add 1" repeats the syntax; "why" earns the reader's attention | `// add 1` → `// Retry once before failing the request` |
| Placeholders in `ALL_CAPS`, explained after | `<your-key>` reads as a literal or gets pasted verbatim | `--key=<your-key>` → `--key=API_KEY` + a "Replace the following" entry |
| Expected output shown in a second block | Without it, the reader can't tell if the sample worked | Add a fenced block with the actual return value or CLI output |
| Sample is runnable as shown | An undefined variable or missing import fails silently for the reader | Add the missing `import` or define the variable inline |
| Introduced by a sentence ending in a colon | A bare code block gives no reason to read it | "Here's code:" → "The following command lists failed deployments:" |
| Verified against code or `--help`, not invented | An unverified flag teaches a command that fails | Confirm `--status` exists via `shipit deploy --help` (see P11) |

Source: code-samples, tech-writing/two "Sample code"

## 8. Verify Facts Before Style (P11)

P11 is the one blocking rule in this file, and it comes before every other rule here: a beautifully formatted step for a flag that doesn't exist teaches the reader something false with total confidence. Style makes a doc readable; it does nothing to make a doc true.

Trace every command, flag, parameter, default, and described behavior to one of four sources before it goes in a doc:

1. **The parser or argument definitions.** Grep the CLI's flag-parsing code (`argparse`, `cobra`, `clap`, `yargs`, or the project's own dispatcher) for the exact flag name, type, and default value.
2. **`--help` output.** Run the actual command and read its usage text; don't reconstruct it from memory of a similar tool or an older version.
3. **Tests.** A flag's real behavior — including edge cases — often matches its test fixtures more precisely than its comments or its `--help` string.
4. **The user.** When no code is reachable, ask; don't infer a plausible-sounding default.

When none of the four resolves a fact, write `TODO(verify): confirm whether --status accepts a comma-separated list` inline and move on. Never fill the gap with a guess that reads as confident prose — a `TODO(verify)` marker is visible and fixable; a wrong sentence that reads well is neither.

The skill's rule (SKILL.md, section 8): "Never include a command, flag, or parameter you didn't see in code or receive from the user." A wrong polished doc is worse than an ugly right one — polish signals authority, so a reader trusts a wrong flag name precisely because the sentence around it reads well. An ugly doc with a `TODO(verify)` marker at least tells the reader where the doc stops vouching for itself.

Source: skill rule
