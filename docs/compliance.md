# License compliance

RonyKit is distributed under the [BSD 3-Clause License](../LICENSE).

Third-party dependency licenses are tracked in [FOSSA](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit). The README FOSSA badge reflects the latest license scan for `main`.

## Full dependency report

Do not commit full license text exports into the repository root. FOSSA treats embedded GPL/CC license bodies as **project** licenses (depth `0` false positives).

To obtain or refresh the inventory:

1. Open the [FOSSA project](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit).
2. Export the dependency / attribution report (CSV or PDF, per your org workflow).
3. Optionally archive exports outside git, or under `docs/third-party-licenses-full.md` (excluded from FOSSA via [`.fossa.yml`](../.fossa.yml)).

Update [COMPLIANCE.md](../COMPLIANCE.md) with a short summary and link only—not full license appendices.

## Local FOSSA CLI (optional)

```bash
# Install: https://github.com/fossas/fossa-cli
export FOSSA_API_KEY=...   # from FOSSA account settings

fossa analyze
fossa test
```

Analysis respects [`.fossa.yml`](../.fossa.yml) at the repository root.

## Known flagged issues (scanner / policy)

These commonly appear under policy **Standard Bundle Distribution**. Resolve in FOSSA after verifying; use the notes below.

| Package | FOSSA flag | Actual / usage | Action |
|---------|------------|----------------|--------|
| `ronykit` (repo, depth 0) | GPL-2.0, CC-BY-NC-SA, … | **BSD-3-Clause** per `LICENSE` | Set project license to BSD-3-Clause; resolve. Caused by pasted third-party license text in old `COMPLIANCE.md`—do not reintroduce. |
| `go.opentelemetry.io/auto/sdk` | LGPL-2.1 | **Apache-2.0** (indirect via OpenTelemetry) | Edit package → Apache-2.0; resolve. LGPL text is from upstream test fixtures only. |
| `golang.org/x/text` | CC-BY-SA-* | **BSD-3-Clause** (CLDR data has separate terms) | Edit package → BSD-3-Clause; resolve. |
| `golang.org/x/crypto` | openssl-ssleay | **BSD-3-Clause** | Edit package → BSD-3-Clause; resolve. |
| `github.com/smartystreets/goconvey` | OFL-1.1, CC-BY-3.0 | Test-only (`flow`, `testenv`) | Ignore or resolve as test-only; not in production artifacts. |
| `github.com/hashicorp/golang-lru/v2`, `github.com/libp2p/go-yamux/v5` | MPL-2.0 | p2pcluster stack | Resolve: MPL file-level, unmodified use; maintain notices. |
| `github.com/modelcontextprotocol/go-sdk` | CC-BY-4.0 | MCP gateway, `ronyup` | Resolve: attribution in docs/NOTICE as required by CC-BY-4.0. |

## Resolving issues in FOSSA

1. **Project** → **Issues** (filter: License).
2. For false positives: **Dependencies** → package → **Edit Package** → correct license → **Reanalyze** → **Resolve**.
3. For test-only deps: **Ignore Package** with rationale, or resolve as non-distributed.
4. Trigger a **rescan** (push to `main` or **Rescan** in FOSSA).

### Suggested resolve notes

- **ronykit (depth 0):** Project is BSD-3-Clause per LICENSE. Prior GPL/CC flags were from third-party license appendix text, not project licensing.
- **auto/sdk:** Apache-2.0 per upstream; LGPL-2.1 is fixture-only false positive.
- **x/text, x/crypto:** Go module BSD-3-Clause; flagged subcomponent licenses do not apply to the compiled Go dependency.
- **goconvey:** Test-only; not shipped in production builds.
- **MPL (lru, yamux):** MPL-2.0 file-level; unmodified use via p2pcluster; notices maintained.
- **MCP go-sdk:** CC-BY-4.0; attribution provided per license.

After all issues are resolved, the [FOSSA badge](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit?ref=badge_shield&issueType=license) should show passing.
