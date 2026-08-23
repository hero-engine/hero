# Public truth reconciliation

This page records how the hosted documentation applies the v0.34 public claim
registry verified at source revision `75ea3cb1`. It is a documentation
disposition, not a mutation of the registry's evidence authority.

| Claim ID | Hosted documentation disposition | Evidence surface |
|---|---|---|
| `public-config-example` | Corrected | Decoder-backed full example plus synchronization test |
| `harness-native-workflows` | Corrected | Installation, project setup, What Is Hero, commands, MCP setup |
| `supported-install-targets` | Corrected | Installation/project setup list and `hero install --help` authority |
| `cold-audit-and-verify` | Corrected | Verified-delivery path and delivery workflow |
| `attention-mail-focus` | Bounded | Attention overview plus Mail and Focus references |
| `hero-serve-intelligence` | Bounded | Local Server and MCP reference; team/external prerequisites stated |
| `code-host-operations` | Bounded optional | Provider, credential, repository, consent, and broker boundaries stated |
| `tracker-evidence-and-mutations` | Bounded optional | Supported providers, credentials, evidence reads, and explicit mutation consent stated |
| `cross-repo-peering` | Corrected optional | One graph per project; asynchronous Mail and explicit receiver promotion |
| `headless-runtime` | Bounded preview | Provider/execution prerequisites and approval gates stated |
| `engineering-default-pack` | Corrected | Core plus Engineering and lightweight PM/QA assistance stated |
| `optional-domain-packs` | Bounded optional | Focused PM, QA, and Sales composition and maturity boundary stated |
| `hosted-docs-deployment-parity` | Still unproven | Build generates revision markers; later visibility/launch gate must deploy and crawl production |

The hosted occurrences of the P0 satellite, monorepo, delivery-close, and
configuration claims were also corrected: satellite management uses
`hero install satellites` flags, monorepos retain one root corpus,
`hero spec verify <slug>` is the sole normal delivery close, and the full JSON
example matches the production decoder fixture.

Mutable command, agent, skill, and MCP totals were removed from narrative copy.
Exact inventories come from `hero doctor` or runtime `tools/list` with target,
filter, and revision context.
