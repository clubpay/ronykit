# License compliance

RonyKIT uses the [BSD 3-Clause License](../LICENSE).

[FOSSA](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit) tracks third-party dependency licenses. The FOSSA badge in the README shows the latest license scan for `main`.

## Full dependency report

Do not commit full license text exports to the repository root. FOSSA can identify embedded GPL and CC license text as project licenses. This causes false positive results at depth `0`.

Get or refresh the dependency inventory:

1. Open the [FOSSA project](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit).
2. Export the dependency or attribution report. Select CSV or PDF as required by your organization.
3. Store exports outside Git when possible. You can also use `docs/third-party-licenses-full.md`. [`.fossa.yml`](../.fossa.yml) excludes this file from scans.

Add only a short summary and a link to [COMPLIANCE.md](../COMPLIANCE.md). Do not add full license appendices.

## Local FOSSA CLI

The FOSSA CLI is optional.

```bash
# Install: https://github.com/fossas/fossa-cli
export FOSSA_API_KEY=...   # From FOSSA account settings

fossa analyze
fossa test
```

The analysis uses [`.fossa.yml`](../.fossa.yml) at the repository root.

## Known scanner and policy issues

These issues can occur with the **Standard Bundle Distribution** policy. Verify each issue before you resolve it in FOSSA.

| Package | FOSSA flag | Actual license or usage | Required action |
|---------|------------|-------------------------|-----------------|
| `ronykit` at depth 0 | GPL-2.0, CC-BY-NC-SA, and others | BSD-3-Clause in `LICENSE` | Set the project license to BSD-3-Clause and resolve the issue. Old license appendix text caused this result. Do not restore that text. |
| `go.opentelemetry.io/auto/sdk` | LGPL-2.1 | Apache-2.0, indirect through OpenTelemetry | Edit the package license to Apache-2.0 and resolve the issue. The LGPL text occurs only in upstream test fixtures. |
| `golang.org/x/text` | CC-BY-SA-* | BSD-3-Clause. Separate terms apply to CLDR data. | Edit the package license to BSD-3-Clause and resolve the issue. |
| `golang.org/x/crypto` | openssl-ssleay | BSD-3-Clause | Edit the package license to BSD-3-Clause and resolve the issue. |
| `github.com/smartystreets/goconvey` | OFL-1.1, CC-BY-3.0 | Test-only dependency in `flow` and `testenv` | Ignore or resolve as test-only. Production artifacts do not include this package. |
| `github.com/hashicorp/golang-lru/v2`, `github.com/libp2p/go-yamux/v5` | MPL-2.0 | Used by p2pcluster | Resolve as unmodified use of MPL file-level code. Keep the required notices. |
| `github.com/modelcontextprotocol/go-sdk` | CC-BY-4.0 | MCP gateway and `ronyup` | Resolve after you add the required CC-BY-4.0 attribution to `docs/NOTICE`. |

## Resolve issues in FOSSA

1. Open **Project**, then **Issues**. Filter by **License**.
2. For a false positive, open **Dependencies**. Select the package and **Edit Package**. Correct the license. Select **Reanalyze**, then **Resolve**.
3. For a test-only dependency, select **Ignore Package** and give a reason. You can also resolve it as non-distributed.
4. Start a new scan. Push to `main` or select **Rescan** in FOSSA.

### Suggested resolution notes

- **ronykit at depth 0:** The project uses BSD-3-Clause in `LICENSE`. Old third-party license appendix text caused the GPL and CC flags.
- **auto/sdk:** The upstream package uses Apache-2.0. The LGPL-2.1 flag comes only from test fixtures.
- **x/text and x/crypto:** The Go modules use BSD-3-Clause. Flagged subcomponent licenses do not apply to the compiled Go dependency.
- **goconvey:** This dependency is test-only. Production builds do not include it.
- **MPL for lru and yamux:** p2pcluster uses unmodified MPL-2.0 file-level code. The project keeps the required notices.
- **MCP go-sdk:** This package uses CC-BY-4.0. The project supplies the required attribution.

The [FOSSA badge](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit?ref=badge_shield&issueType=license) shows a passing status after all issues are resolved and scanned.
