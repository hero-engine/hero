# Releases

Every release published to `hero-engine/hero-releases`, generated
automatically at docs build time. Pick a version in the left nav to jump
to its notes.

## v0.33.0 — 2026-08-18

### Major Features

- 7c0423aa060e2bad40b5375a8f8f6cb8d91b2c6a feat(install): add native Grok Build target
- 8088f225b61e2ff6e377fab7fb8d4b2bb7b082f5 feat(mail): add cross-project read contract

### Fixes

- ffc0e47c327e7ed8990fd75297f0fcfe19254f77 fix(index): preserve repo-scoped graph node identity
- cfc0c5d83f36a7323f4db379d5674185fad2139d fix(spec): ignore hidden Hero internals during discovery

## v0.32.0 — 2026-08-14

### Major Features

- 73fac3a3ba8bf118103c65d93e5e41889665da44 feat(cli): close interactive setup and connect
- 2767452418ff7885c4b60d3f6b7fb37961196d9b feat(cli): close prompt and TTY contract
- 268214cd55b059831984ad598f09fdb2657affdd feat(cli): complete corpus selectors
- f2626dc313d71f3bea7f56ceae42a5bf1d3fe32f feat(mcp): category + tier metadata on tools/list for harness-agnostic deferral

### Fixes

- 02c47d42eb62243da005f88def331f7ceb5fbe7c fix(cli): close prompt-and-tty-contract-closure audit gaps
- 70c62797f9a52cc2ce8a04257783a316afcacb62 fix(cli): preserve shared setup files on uninstall
- 81ede0f517e7be1514e967c815ce843fb660d84d fix(cli): reconcile successor test contracts
- 107272f8271c422f82254103c08bb712f554b496 fix(hero): reconcile interactive CLI audit evidence
- 76ed88ee6d8fed562b067074a456be30c99b60cc fix(release): supply version-matched Hero Code artifact

## v0.31.1 — 2026-07-30

### Changes

- Adds a GitHub-first, provider-neutral code-host boundary distinct from issue tracking, with versioned fixtures and credential-safe CLI and MCP surfaces.
- Adds bounded pull-request discovery, detail, commits, diffs, checks, reviews, comments, merge readiness, creation, collaboration, lifecycle, and merge operations.
- Adds authenticated actor identity, truthful diff availability, permission-aware mutations, and idempotent reconciliation.
- Hardens Jira attachment downloads with redirect=false and same-origin credential enforcement.
- Classifies GitHub SAML SSO requirements without exposing authorization URLs.
- Makes cancel-after-apply coverage deterministic across create, collaboration, lifecycle, and merge operations.

## v0.30.0 — 2026-07-27

### Major Features

- 1a0e66b07c7c39cc93a5b264b2f0012b0aecef25 feat(attention): define interaction consent contract
- 5a725dde8e8b42a4454fe4643cba71f3c1e3714c feat(attention): deliver conversational inbox operability
- b9e02e6250362db9ba7bf3ae00bd5ba63fbe975b feat(attention): deliver deferred work suggestions
- 6ab4e091066c25c357738e122dcf4bab0fc41586 feat(attention): deliver global attention read model
- f5a5b1bfd7253b58fb038fbc96ce84b38f6537c9 feat(attention): deliver project mail triage provenance
- b48f2a15bd243102bc74c09e780e4bb48f75e86a feat(cli): make hero status actionable <!-- drift-test:ignore -->
- ade0016883e8058510927f7b816e8049b112f084 feat(peering): route peer workflows through project mail
- 40cd4c2db0cce0ae0d5650c32f3f1128e2220d41 feat: add durable attention contracts
- 592cb9947d1b967389993daacfcdcac7323ea2cf feat: add incremental code index refresh
- 42055dcaa2fab793c8afac584dae5525c4aadff8 feat: add personal focus core
- 35fbc855bc54b5a75bcb0a89926cdb7b4f77caa7 feat: add project mail core
- 0816a56cef7b09fd6d80f818e4003a936e8c0634 feat: add tracker project snapshots and durable attention specs
- 08597718acbf7b11000af77a80d40e65502108da feat: keep code indexes fresh from git hooks

### Fixes

- 35ee79f297aab06b0e043a28f140d77ffbdef91f fix(attention): refresh suggestion surface counts
- 23dc1e19be86c9fadedf7a15244be4ebd289a516 fix(graph): repo-scope node identity and close the peer-spec staleness tail (#4)
- 9a6a8292e8dcba1b5cb5aeb87e2110f8ed4eaaf4 fix(release): make file locks portable
- 1e5da796d5acc7ded190f27b5bac74ee91a55be9 fix(spec): block initiative auto-completion on unbuilt declared children (#3)
- f4cc20d451b8a808ea54b682d1070df8c19ca59f fix: close intake primitive audit gaps
- 3712247b813805c624e8b964ac2e8fba547b6243 fix: keep code refresh lock in cache
- 189e28815e5f1200aaacfdce78668f1180fdf954 fix: refresh embeddings with code index

## v0.29.0 — 2026-07-22

### Major Features

- 3f675c67646afd481cd18bb2a42b34cfa61bdad5 feat(tracker): add lazy evidence sidecars

### Fixes

- 1d5ae9596a63c3af2497322fb228327cbc0633e9 fix(tracker): preserve Jira ADF description fidelity

## v0.28.2 — 2026-07-21

### Major Features

- 7981763753f941eda68fc45a5a44afcfab2d3b47 feat(tracker): add credential-safe brokered access

## v0.28.1 — 2026-07-21

### Fixes

- f4d3a18aa6dc9f05848c361cb82e1b45bee61a7a fix(tracker): make serve auto-refresh bulk-only

## v0.28.0 — 2026-07-20

### Major Features

- 85ada637fc91466b32b7b007193f059d912f4a22 feat(design): compact acceptance-criteria recaps
- c3823e086592b84f933f851af7ed51e287a1c2c7 feat(tracker): harden imports and add full-ticket evidence

## v0.27.3 — 2026-07-20

### Major Features

- f410356ab8524bd22b77bd508a54decc8a0ae994 feat(tracker): preserve native issue activity timestamps

## v0.27.2 — 2026-07-18

### Major Features

- 9d7c707a493da81bcec04f379066763b27894011 feat(chat): add machine-readable checkpoint sentinels to /research contract
- f5fb8d6b7fa9854417ab8fb8867494b03c4e99f0 feat(chat): extend domains/chat into the canonical generic Hero Chat pack
- 267f8eb2cf14f6d95704a9ab59770ceb4c623584 feat(chat): slim chat to basic + preserve research apparatus as dormant seed
- 04a0b5dcbd0b4d44dfb67270878bbcc710f822b1 feat(chat): trim research apparatus — keep chat simple, light research habits only

### Fixes

- cf6ebc28045410f7c5325e1bce246a336cb62d34 fix(graph): heal hero why on read + fix repoKey drift across graph writers (#2) <!-- drift-test:ignore -->

## v0.27.1 — 2026-07-18

## v0.27.0 — 2026-07-18

### Major Features

- aba3fa4d32986fe84e9f11dd2c836c7d4e3aba80 feat(doctor): show installed harness targets with per-target content counts
- a14a0abf626fd13540dfd99a29e203222df3ae01 feat(pm): PRD editor comms backing — pitch-author + stakeholder-communicator (pm-pack-completion #4)
- a7ccd0ef5a4c37ee7c696cb8a1b584a14a9a9fc1 feat(pm): adversarial critics bundle — drift, prioritization, doc, readout (pm-pack-completion #6)
- 03245c2ef86fdd308d2bf0ad15ab2feec6b83688 feat(pm): competitive + market grounding (pm-pack-completion #8)
- 15aeda7e11c63f2050fcf902dcaf4a265b3d7f48 feat(pm): discovery & framing coverage skills (pm-pack-completion #10)
- 7f652f4a8df111ae0ddf009ee6e532c6e308db87 feat(pm): doctrine spine + skill backfill + canonical AGENTS.md (pm-pack-completion #1)
- 917c5494621be0cd5585a717c4e30f75225b9773 feat(pm): exec narrative + evidence synthesis depth (pm-pack-completion #9)
- e2b82c41d2b505858016497a790e27c96cbbb402 feat(pm): experiment stage + metric RCA (pm-pack-completion #7)
- 2d3bfdf5cd254a693ade1675f626813488c3e80c feat(pm): remaining roles, scrubbers & launch (pm-pack-completion #11)
- de5da2fa011c191330e7c73a0f30f0caeebde5df feat(pm): story queue planning backing — capacity + cycle planner (pm-pack-completion #3)
- 7679134971240ea2a6c27222f908d15b6a98f0a1 feat(pm): story-detail deps + intake scrubber backing (pm-pack-completion #5)

## v0.26.4 — 2026-07-17

### Fixes

- e42e272dfb568f7a4ca056f86d8cf4f84ad302f2 fix(tracker): validate connect settings before persist; add sync link --force and dir/slug args

## v0.26.3 — 2026-07-17

### Major Features

- aa948a7aef7e2d0a46c84f0be69be5affb70d2f0 feat(design): show the acceptance-criteria table in the closing report
- 5e87f84afafbeb0c8fa2fef367ac111dc6801f65 feat(install): manifest-driven prune of product-removed agents/commands/skills

### Fixes

- 9d31597654e9e8e6c3d9423b646090c3d63c510b fix(codex): scope .codex/agents cleanup to legacy .md — stop wiping user files

## v0.26.1 — 2026-07-16

### Fixes

- 96db39359e807af8665e5cac5d98a97a0131a408 fix(projection): drop the drifting branch: line from NEXT.md; keep session:
- b86b821648907967a2a0ea818da54e5e091a7a0c fix(projection): regenerate NEXT.md branch-free with the fixed binary
- 353cc373cf7cb3f3f4743a57e834e0c7607f0572 fix(serve): join the parent-watchdog goroutine in tests — unbreak the -race CI gate

## v0.26.0 — 2026-07-16

### Major Features

- 8f8972463c42e1880f5e1347c2106f57f707fe97 feat(check): install-integrity — hero check detects damaged or stale installs (install-integrity-self-check) <!-- drift-test:ignore -->
- f122397c52c797fdd21f2264ff8f0da7f5928fb6 feat(config): add layered integration configuration
- 79f47e8c19866e363518d3b6dac55607d91640cd feat(install): spec install-integrity self-check; capture the verification lesson

### Fixes

- c10b4dbbd7dd19fc4604c1a8e091342fccff8338 fix(check): orphan-instruction-files no longer false-positives on a fresh clone
- 23c113539fa2da0d9a53bf4af644fd4fd4f2966b fix(install): MCP config points at the running binary; codex block moves to the User layer (codex-mcp-binary-path-resolution)
- 5a4d1515678bcded17fa520847290072d7d7791b fix(install): MCP wiring is portable and lives in the project layer, all four targets
- fa3b339449403c6c0bf927f6ebb3b3dfaf0056b5 fix(snapshot): stop the pointer writer erasing AGENTS.md; guard it for every target
- 0d69f13ff4a258ccf9d9ec472ba47dee8040de2f fix(spec): classify EARS under an AC-N label; document the labeled form
- 153ab290ed9d76a013df7e2597845da0a5f240dd fix(spec): stop fenced `## ` headings from forging phantom sections

## v0.25.1 — 2026-07-13

### Fixes

- 2b35aca09ab6820ff2ee13c3a337690ba360c112 fix(docs-check): count canonical install set in the engine repo; refresh doc counts
- 3c38bccb46b84fa2ebb71b594f94f0aa53b1fb7f fix(serve): stop hero mcp daemon dying mid-session; coexist instead of supersede <!-- drift-test:ignore -->

## v0.25.0 — 2026-07-13

### Major Features

- feat(doctor): add `hero doctor` — reports the running binary path, the <!-- drift-test:ignore -->
- feat(mcp): MCP `initialize` result now stamps binary + graph schema
- feat(docs): auto-generated release notes page, built from published

### Changes

- MCP daemon startup now supersedes redundant daemons per (workspace +
- Harness routing guidance across all six targets (opencode, cursor,

### Fixes

- fix(hero): schema-mismatch errors are now self-locating — they print
- fix(hero): corrected a lexical schema-version comparison that inverted
- fix(docs): corrected stale `hero verify` / `hero peer --peer` <!-- drift-test:ignore -->

## v0.24.1 — 2026-07-12

### Changes

- 07dbcfc58a11ed3c5adb6aa681c006ba8eb95086 fix(next): make NEXT.md projection deterministic from committed state

## v0.24.0 — 2026-07-12

### Changes

- df0490af9fccd15b525fa4a2f5b9676e105f2353 Merge branch 'main' of github.com-hero-engine:hero-engine/hero
- b640a38ca7d37fbef5b6dfe2514dee6c0c8903ee Merge core-commands-domain-neutral delivery
- b72100f20e85d2339e46dd15b8aa4c13849fdcff Merge delivery-gate-consistency delivery
- 6001260ceb9f142601e70feb5736e87943dbace1 Merge harness-agnosticism-sweep delivery
- 0b160fadfa69f6ebed98c80256e00f434bad1587 Merge remote-tracking branch 'origin/main'
- 5e6d11af48a1dd944ff22c07f4b692b27ee652c3 Merge routing-file-completeness delivery
- ae5fff657a7ae148d67c201ee649fa38d8e55f56 Merge token-efficiency-pass delivery
- df8431d3fb96bb9d36a01ecee0ea57eae0b3bb23 chore(hero): add events.log merge=union driver, refresh handoff projection
- bd2ba4e6d757be70909938617ee1f23b59fda0ce chore(hero): backfill created: on 42 specs from first git commit
- e6163a948ec258953c9f6271e383a80a6e9431a6 chore(hero): reconcile completed-but-stuck spec drift
- 758e39926a76f62559ee001047488eca358d5053 chore(hero): reconcile status drift (planning -> delivering)
- 60169acddff6fde23389a6675d7a474aa44441bf chore(hero): refresh projected NEXT.md and SNAPSHOT after gitignore fix
- 752e5169301c802a9a2e615bbce438239586853f chore(hero): refresh projected handoff state
- 4719c9883806ef6740dd396e325f09b7074be5aa chore(hero): regenerate handoff projections after content-remediation completion
- e633113e8901e5c4b3b9715f91a33038773e9495 chore(hero): regenerate handoff projections after harness-agnosticism-sweep merge
- defd65ace23c47560d92506afd512c8eac757c90 chore(hero): regenerate handoff projections after routing-file-completeness merge
- d4189870f9225c9a7508a9352c466c4d8a5bf5bd chore(hero): regenerate ready queue + peer manifest after merges
- 9ea47374f6d35e1c00ef1b9c2f0f576b4174e2a9 chore(next): commit clean-graph NEXT.md projection (bypass local hook)
- f9fef157bdccb7b42bc97eebe475aa33c11ef2da chore(next): sync projected NEXT.md with clean-graph checkpoint
- 4e99a0e9e9374a8327c4c6e6fc70237add4d27ad docs(codescan): close out created/slug clobber follow-up as wont-fix
- 04f6968b3268d9eb5a8fdc53dcb87bf3a2469442 docs(knowledge): sync project rules and context specs to current state
- 3f272ed174c9d49111567a81ce8ed961899df518 feat(codescan): make incremental scan produce a complete Result via scan cache
- c779580e787cdd9e407e42dff4eed143f84b4e12 feat(compose): emit priority + reciprocal conflicts-with on child stubs
- d2c9f6d9ca851e8c2745c673cc7d5115f0280da7 feat(drive): FindConflicts backstop — detect undeclared seams in the judge
- e3c677fb515da8ad65935509590596a199f15b82 feat(drive): priority- and conflict-aware next-spec selection in the judge
- 946bf482002b42d8f137dc91483cae1cdbd60bf9 feat(hooks): stop events.log churning the tree after every commit
- 606a746a060786cc7725f835df9bdd741e8c3899 feat(install): harness-native, target-aware install & upgrade
- c5c01bea2cdf17eafc78f77c831b93800f315176 feat(knowledge): flat tripwires trigger-highlight via knowledge_triggers seam
- 12aaf62c6b55ad484f1ac63503d07ce4a7998c77 feat(knowledge-surfacing): layout-agnostic ingest/retrieval, context injection, inline-flow relation fix
- dae20dcaa7d835e4d8861f401f036cddd2bb60ea feat(list): surface created in list JSON + make the created date reliable
- dd352ce47d64a4da94a916c792bad003abec6550 fix(cev2): reconcile context-engine-v2 corpus to verified reality (cev2-corpus-reconcile-to-v3)
- 414344c49e34336b09505d8a2f619c35ddc1001b fix(chat-pack): formalize domains/chat as client-embedded, not dead (chat-pack-disposition)
- 92919e6334821e61e1c1c18b5c59faa4c7894d20 fix(ci): drift gate ignores the updated: timestamp line
- 5e956524b9fffd26a40756125e66d6fa5e5b5329 fix(ci): make the NEXT.md drift gate winnable — drop churny count, fix scan flag
- b5e7652fde4c32f3250f97f4057e1d6e9f86a979 fix(codescan): stop incremental scan from deleting unchanged package knowledge
- 177e8a1bc7806f4b97280c594fa62f18feca37d0 fix(content): single-master core/engineering content + CI parity gate (content-dedup-resync)
- d0f6e0d05c25c27d8c7f9c0d559e11800f146b5d fix(core-commands): make core/commands/ domain-neutral (core-commands-domain-neutral)
- fdba2acc346eae9038bfcd3eeb861d3ba802210f fix(delivery): apply the ledger-citation and gate-bypass edits (delivery-gate-consistency)
- db899c599ab40451fc37c09c3494c40699a413d2 fix(delivery): unify Completion Ledger contract, remove verify-bypasses (delivery-gate-consistency)
- ce3e9131c711e4233496d63d1211140f605609a6 fix(harness): scope Claude-only assumptions, fix parity table + dogfood leakage (harness-agnosticism-sweep)
- f1b5388a804e9f27b102a30b63431b7cf602af86 fix(init): ignore machine-local .hero artifacts in managed .gitignore <!-- drift-test:ignore -->
- 4b2ac5823336e6b01a281e7b4f8e49e4d1bbbf0b fix(install): cursor skills flattening, README exclusion, --json exit-code/repair/migrate parity
- e00ae038f3f73daf606a3074701c6d931283935d fix(install): honor --json contract in --repair and --migrate short-circuits
- 03dfef9bb7914786ecf5a7c178a643cd76baed7f fix(opsrunner): kill data race — per-Runner clock instead of mutable globals
- 5d51798f9eed4ab34358185eba5c57fa6915848a fix(opsrunner): stop-and-wait Runner lifecycle to end pump goroutine leak
- 65e3333b7343ab51e20a9f64a35ecb99f8777275 fix(pm-pack): remediate phantom surfaces in PM pack (pm-pack-phantom-surfaces)
- 5c99b9424818573fa77769497d4dd247993a91ec fix(routing): complete AGENTS.md rosters, align pack skeleton (routing-file-completeness)
- 35addd4eb171d004a2d65dfc8a9bc0b2f58d1d79 fix(sales-pack): sync every sales-pack claim to the engine (sales-pack-reality-sync)
- 728b4aa478c51f3b7739c0abbcaac44319e69937 fix(scan): preserve authored created:/slug: on knowledge-entry regeneration
- 61ce4c86ffcefb1d87c9c5aca10805bbd018acf0 fix(smoke): give smoke scripts the built binary and a populated graph
- c79d47165dcfe063df77f780c5be278c466cd3f2 fix(spec): discover flat-named specs so verify resolves initiative children
- e7ecb907c5c3ab83aa47c657d913f5a18ae8e011 fix(token-efficiency): cut named audit verbosity, single-owner content homes (token-efficiency-pass)
- 9a51152f178773273fc32e3f4362b66f779d297c fix(verify): complete initiatives from already-archived children via reconcile
- 0349bba95223ff329ff98a841c8bd79c90c7a853 hero stuff <!-- drift-test:ignore -->
- 40a3b856759a8618315a033806293d90aba5dd41 plan(content-remediation): archive content audit, record multi-domain decision, compose remediation initiative
- bc86ad9ae02b4e33155e4b244ccb31a5691abbaf plan(team-oauth): complete + close hero-cloud peer loop

## v0.23.4 — 2026-07-02

### Changes

- 05cca7e08d7e6209c412cd714ead1a2f5c05eef4 fix(opsrunner): kill subprocess process group on ctx cancel (unblock release)
- 1a3227a391b5ac79775f28a0315b75ca2c576f00 plan(tracker-fixtures): complete initiative, close hero-code peer loop

## v0.23.2 — 2026-06-29

### Changes

- 2ac1f8bdae8fef5e5ee26d1ab71ee8ad21516a6b build(deps): use published sprout-Go v0.1.0 (drop local replace)
- edc8bedcc3c08e0c01be7d7b4bffd9d8ba618977 chore(hero): sync projected handoff state (NEXT/SNAPSHOT/events)
- 3877a1520f54458037032ff25316ef014d8f6582 chore(hero): sync projected handoff state after completion flip
- 6c92497acad62769e493fee7a6bbdf3962567746 docs(drive): document /drive in public docs + landing page
- 0d89c99b2064a54c60907c9ce2fd39ac4c59b505 feat(mocktracker): offline multi-tracker HTTP fake (SQLite + sprout-Go)
- 2f7d250c85ed981ffbf4281850f564e7190162c7 feat(tracker): add GitLab adapter (issues/epics/iterations round-trip)
- 7eb91e968518dcd02e9b42faafefab57ac50a26a fix(install): land delivery closing-gate contract across all targets; correct symlink→write docs
- b54365fb84b5e802de91fd22620ead529d32f9e5 plan(intake): add intake-capture-loop spec, capture-intent decision, harness-coverage tripwire
- a7d1bee35a7d1149386004c0c34610be1496738c plan(tracker-fixtures): mark mock-tracker-server + gitlab-tracker-support completed
- 8f051af0940de8b2a33233059283e115a6430c3b plan(tracker-fixtures): offline mock server (SQLite + sprout-Go) + GitLab adapter

## v0.23.1 — 2026-06-28

### Changes

- eff10ae4bac444616bcae9c33748f9226e23bc24 Add knowledge and mock export commands (#1)
- 9d063b6b764e4155535928cbd14b3277afa1bd3f Merge remote-tracking branch 'origin/main'

## v0.21.2 — 2026-06-24

### Changes

- ce338c5c79c817d47e1973e068239d2b7d24837a docs(peering): plan multi-CLI subagent backends; refresh projections
- 8196e8eec642ba8f2b0f921c9d50460f1a06cf65 fix(peering): surface subagent stdout and give actionable login errors

## v0.21.1 — 2026-06-24

### Changes

- 5fdea73ef993464df4824d0b7a4259ffcf7d8856 chore(hero): refresh NEXT and SNAPSHOT projections
- 7dbffb3f7b0cbb4b15ddb0b86370c367fab05588 fix(graph): stop double-nesting /api/v1 in graph sync URLs

## v0.21.0 — 2026-06-24

### Changes

- 97c0dbfb207c00442be463a020fb8bab18aece0e chore(triage): hand off 7 hero-code bugs; supersede install-target
- 6a86562dfc2cb095bd88b2b603f551da9ac870f0 design(planning): scope fks-cluster-detection (#3)
- 13f810f123075229021226353720425d1e7eae50 docs(knowledge): note Diátaxis "explanation" lineage in explainer-format skill
- 9f18fc3ad9810f0d86b9ca3d595f6cfbef69722d docs(knowledge): synthesize the cold-start-trust-hardening explainer
- cdcde69e98a303f288c0b1ffe8acf2a7c5bf00e9 docs(planning): capture team-mode cloud coordination feature
- 32ccaa3622c875eec7aea96e3f1e419e9bacae48 feat(knowledge): add the explainer knowledge type
- c46b7e9e7a8b985eb5d049b8218e92505291d812 feat(planning): scope feature-knowledge-synthesis initiative
- 1fde4fab3e5b06d4a532ace02bef197875483593 feat(synthesize): add hero synthesize and the hero_synthesize MCP tool <!-- drift-test:ignore -->
- 9570310c144741ed22b8b20e86548e5398630d73 feat(synthesize): deliver cluster detection (hero synthesize --detect) <!-- drift-test:ignore -->
- 75dd0cc2ef78cf337c51a326a72105d79d6aaa69 feat(synthesize): deliver living-doc amendment; complete the initiative
- 7eb151ca86bd8ea0207f430604bd82fbdda25e0e feat(synthesize): deliver the trust handshake (auto/review/off)
- 7c34dc7cf06670d113b4901fb58c656e1613fe2d fix(now): stop "you"/"me" claim sentinels colliding with real identities
- 72121934fa2498a703e6b785bd26b967389c7463 fix(queue): exclude knowledge specs from hero queue output <!-- drift-test:ignore -->
- 1d88b0a5731d311205d12b7188b342684e30abf1 fix(spectypes): skip spec-types.json export when only timestamp changed

## v0.20.0 — 2026-06-24

### Changes

- 5fe933a53d03651fee035525f2e631e70b7b298e chore(planning): complete cold-start-trust-hardening deliveries
- a6a46dd1077ea8d58b883df40050b0d49c9f3184 docs(agents): document relation syntax; map relates-to to an edge
- 074f30cc4d3dd05fc9a41b5fd66166df5197d69d feat(check): warn on [[wikilinks]] that create no graph edges
- 0cd04b038b3e77c88c57275b8aac284ffaef5041 fix(blocked): derive blockers from frontmatter for cold-start parity
- c446101f3cbdd2eff8a4c38f3d5314a319f7ba9e fix(check): severity-aware human summary, collapse kickoff noise
- ad630736c3db6e652c202f56957da1e944d775ad fix(scan): label Tier-2 as optional enrichment, not graph-critical
- 6ffff4251d75b458a7bb8dcc521a77f648589da6 fix(spec): recognize relation shorthands and block-style lists
- b5b91d0cbb1dd8f4f6b159490fb131c67b7017ed fix(verify): archive sibling artifacts with the spec
- 9395c1e9d1a117b75f5e75f072f371ab1f628616 fix(verify): require declared children complete before auto-completing initiative

## v0.19.1 — 2026-06-23

### Changes

- 4a01d65b8cf71048804305a872a5f6fd43b9489d docs(planning): add cold-start-trust-hardening initiative
- d10a5be600fd86c93d7d6f170399240d3fbe1c9c fix(verify): gate delivery checks on spec lifecycle status

## v0.18.0 — 2026-06-12

### Changes

- 86ede335a799c9a91cf8f4de83cd07b705bfa58b docs(web): landing + docs content refresh — peering, semantic search, verify gate
- 7374ea40fefe6521cb8e604455681b836012f40a feat(verify): close spec completion loop — ledger→graph writeback, initiative auto-complete, exercise demotion
- db611bbee4d1080e93664c3001ee5cc9104ff8db fix(docs): correct hero verify → hero spec verify in web docs <!-- drift-test:ignore -->

## v0.17.1 — 2026-06-10

### Changes

- d4573c05f719376cde5b73c9f75b1998df8ec9d9 feat(planning): retrieval-quality initiative + agent-safety-conventions spec
- fe7ae28a81c936619daf16955845f2261849fe32 fix(next): eliminate projected-file timestamp churn and commit-list chicken-and-egg

## v0.17.0 — 2026-06-09

### Changes

- 12574a9f0ac01dbfda7447c3671a57405fdf54ab feat(context): tripwire system — forbidden-option guardrails [tripwire-system]
- 6020658af8fb188590d02404821a42afea6f3dd4 feat(e2e): e2e-discovery — 6-AC discovery verb smoke suite (search/ask/recap/next/resume)
- 2c78aeb3829dda7a6429ceb355367dfc8a2235ed feat(e2e): e2e-traversal — 8-AC traversal verb smoke suite (why/blocked/impact/relevant/suggest/conflicts)
- 763b12ace641decc623567dc1b8a7152c4cb7191 feat(e2e): e2e-validation — repeatable smoke harness and run-1 findings documented
- e9874a2edd32c1fd8cf83d34d6446c8f898cda22 feat(embeddings): embedded inference — zero-dep Model2Vec semantic retrieval, hybrid RRF search
- 85e1386a95bb6b1fbc364a6a8deee73fe0264402 feat(graph): graph conflict detection — detect concurrent node divergence (SC-1..4)
- f1763f82c200fd8b3f6b76038ff81f6be5c806e4 feat(install): monorepo satellite installs — one workspace, many subfolder entry points [monorepo-satellite-installs]
- fc6ce2ba242b743be5ac85d9d68d7a620fc40a42 feat(integrity): spec status integrity — graph-verified delivery claims [spec-status-integrity]
- 89b1169edee46fd326a645290fb20c47770276e4 feat(peering): cross-repo peering — peer identity, manifest, handoff lifecycle, sync peer call, contract imports
- 9f81f99fb0784b804bd1e844740a6c9ed679021e feat(retrieval): unified retrieval layer — FTS5 BM25 with type boosts, single Retrieve() interface
- 3e792fbaf2ee2ff5c79793259c6c1e0ba3099053 feat(sales): hero-sales domain pack — MEDDPICC qualification, deal strategy, forecast, pipeline (markdown-first)
- c8ac9750393917657f61a70bd9e3c2a41b6ffcdc feat(search): unified-search — repo label for cross-repo results [unified-search]
- cd3a3f7e4e517ef6a6399f080a9c7a8e8a77a2b2 feat(smoke): per-feature smoke coverage — continuous real-world verification [per-feature-smoke-coverage]
- 326da19b338196685a0814376f69978c3c732268 feat(traversal): traversal queries — hero why and hero blocked [traversal-queries] <!-- drift-test:ignore -->
- afe95534b942811b739459100a91afbedb7a95c8 fix(codex): commands as skills + target-aware AGENTS.md bridge Codex workflow execution gap
- 48052ccdba6518b6c3a83c496eeaa079ec1da39f fix: fold methodology-derived step into vocabulary.Resolve (vocabulary-resolve-misses-methodology-derivation)
- 94caba04f72a5ad9ea3ebc431d6f4965aa00c80d fix: forward dialect fields in MergeLocal (hero-local-merge-missing-dialect-fields)
- 19684689a4e5985f048ce5e60faab3b410c2b3f1 fix: populate spec-types.json frontmatter from source markdown (spec-types-cache-frontmatter-empty)
- f4beda7b42a36ebd375eb15d48b8fa297b610f9d verify and archive delivery-gate-enforcement + compact-handoff-test-coverage
- 67b4390a1735f43e18ed795d4d0cc9855c8dd6c7 verify and archive master-ingest-restore

## v0.16.5 — 2026-06-06

### Changes

- 1427a091b69888e858b8b555c926bdd6cd812311 fix(graph): tolerate newer schema — warn instead of failing when db is ahead

## v0.16.4 — 2026-06-06

### Changes

- 228097cc663c53594ea7ba82dda933a8a4cdefcf fix(upgrade): detect all installed targets — codex, copilot, generic were missed

## v0.16.3 — 2026-06-06

### Changes

- 71e4b1b8fed422d1a139a5689f9e686fb3894a4b feat(verify): delivery gate enforcement — hero verify becomes the load-bearing checkpoint <!-- drift-test:ignore -->

## v0.16.2 — 2026-06-05

### Changes

- 15e6104285659343dfdde232426c905052f4d94d fix(cli): wire top-level `hero connect` alias for `hero sync connect` <!-- drift-test:ignore -->
- 9537eb1aa2b92ebfdc716e4f4887a0e423f19bf2 fix(scan): detect .gsp, compound asset-pipeline extensions, raise walk cap

## v0.16.1 — 2026-06-04

### Changes

- 76dc9fc6ee4c4829585ad7ada7072fff890e44f8 Merge: NEXT/handoff subsystem overhaul
- 57fcb5c74f033d2109837f10628c4137bc87467f chore(next): close merge specs (2 delivered via union, 1 superseded)
- f576fec1d6b8166f7bca5f8f2346021003e85538 docs(next): add NEXT-subsystem bug specs from deep-research pass
- 827c8064dd77fd8c3f33a7e446440dd274e39dfb docs(next): capture the converged goal-capture design (was chat-only)
- ab8c299fcdca0fd9872affa33f51e4400c3aab1a docs(next): diagnose desktop sidebar notRunning + archive intent spec
- 09f3f104818b158d8615760a57cd22b0d05dc930 docs(next): record resume-brief gap finding in handoff simplification spec
- e59f705f154754d85513925fe99524dede81d1ca docs(next): spec resume-brief-surfaces-handoff — close the load half
- 5fec0a4a49981957d5daf61f1261a06236fc800a docs(next): spec the handoff simplification (deep-dive synthesis)
- 8ddf5ea52998118a261dfee69f5d52ab17bc5cd1 docs(next): tier the intent-capture design (first / embed / distill)
- e873b1f6a815970195a7122eb332f266b62b1825 docs(next): triage the content-quality eval findings into specs
- 468974ac7383aef10d801cd04dd0414118fc0a22 feat(next): auto-emit UserAsk at end-of-turn checkpoint — kill handoff drift
- c8bc9acf1746da5ccf422e4154f1823ef7039f60 feat(next): capture session goal (kickoff intent), not just last message
- 99ad332d58584ce30d3f2cb0d883c220348ae53f feat(next): surface handoff in resume brief — close the load half of the magic
- 3efe9ec59fb84ad8c92505e2f078cf23e3028733 fix(next): auto-migrate the projection gate + born-projected installs
- 940ab59c0ca64387a52e5005728aba8f3cd5d0b6 fix(next): cross-machine handoff loads under divergent git identity
- 41b4ae16a073370c59500d9ff69d83b1f1ba1ea8 fix(next): make handoff-file staging unconditional — files always travel
- 5542e1fc298635f00a7eb954453cf00250c42e9e fix(next): project per-user handoff in team mode + mode-aware migration path
- 5c77b8373e28088ab131a9ae30e62451e93a3b67 fix(next): rebuild project context for the resume brief
- 5d9dbc6a5360355a8bdcde065025dfbb579d8d4b refactor(next): merge=union for projected files, delete custom merge driver
- 5c09c3750e56a972713d7ce7648898e1ad59d48b style: fix gofmt struct-tag alignment drift in Config
- 4edc4b831c4f57df6e7ed23acb572838bd3aaed5 test(next): handoff continuity guardrail — protect the magic

## v0.16.0 — 2026-06-04

### Changes

- 76dc9fc6ee4c4829585ad7ada7072fff890e44f8 Merge: NEXT/handoff subsystem overhaul
- 57fcb5c74f033d2109837f10628c4137bc87467f chore(next): close merge specs (2 delivered via union, 1 superseded)
- f576fec1d6b8166f7bca5f8f2346021003e85538 docs(next): add NEXT-subsystem bug specs from deep-research pass
- 827c8064dd77fd8c3f33a7e446440dd274e39dfb docs(next): capture the converged goal-capture design (was chat-only)
- ab8c299fcdca0fd9872affa33f51e4400c3aab1a docs(next): diagnose desktop sidebar notRunning + archive intent spec
- 09f3f104818b158d8615760a57cd22b0d05dc930 docs(next): record resume-brief gap finding in handoff simplification spec
- e59f705f154754d85513925fe99524dede81d1ca docs(next): spec resume-brief-surfaces-handoff — close the load half
- 5fec0a4a49981957d5daf61f1261a06236fc800a docs(next): spec the handoff simplification (deep-dive synthesis)
- 8ddf5ea52998118a261dfee69f5d52ab17bc5cd1 docs(next): tier the intent-capture design (first / embed / distill)
- e873b1f6a815970195a7122eb332f266b62b1825 docs(next): triage the content-quality eval findings into specs
- 468974ac7383aef10d801cd04dd0414118fc0a22 feat(next): auto-emit UserAsk at end-of-turn checkpoint — kill handoff drift
- c8bc9acf1746da5ccf422e4154f1823ef7039f60 feat(next): capture session goal (kickoff intent), not just last message
- 99ad332d58584ce30d3f2cb0d883c220348ae53f feat(next): surface handoff in resume brief — close the load half of the magic
- 3efe9ec59fb84ad8c92505e2f078cf23e3028733 fix(next): auto-migrate the projection gate + born-projected installs
- 940ab59c0ca64387a52e5005728aba8f3cd5d0b6 fix(next): cross-machine handoff loads under divergent git identity
- 41b4ae16a073370c59500d9ff69d83b1f1ba1ea8 fix(next): make handoff-file staging unconditional — files always travel
- 5542e1fc298635f00a7eb954453cf00250c42e9e fix(next): project per-user handoff in team mode + mode-aware migration path
- 5c77b8373e28088ab131a9ae30e62451e93a3b67 fix(next): rebuild project context for the resume brief
- 5d9dbc6a5360355a8bdcde065025dfbb579d8d4b refactor(next): merge=union for projected files, delete custom merge driver
- 5c09c3750e56a972713d7ce7648898e1ad59d48b style: fix gofmt struct-tag alignment drift in Config
- 4edc4b831c4f57df6e7ed23acb572838bd3aaed5 test(next): handoff continuity guardrail — protect the magic

## v0.15.3 — 2026-06-03

### Changes

- c1ff1aa16f3ad0c4d9c93c51c949d816e295cef8 Merge claude/admiring-chaplygin-a8d157: docs-check + NEXT-travel install guidance
- ed901abd374e294b5418bc02ba885821be784f4e docs(routing): add /mock natural-language routing guidance
- da246f476927be002cf11d1910e3adcb1cf5d8c2 feat(install): instruct agents to include .hero/NEXT.md and .hero/next/*.md in commits
- 99d1fc74ef7a7aa683afb3a3dd900aa2758a027d fix(docs-check): count skills as directories, not flat .md files
- b26c2f7bed64a2d1ed723adf9e8581a453910143 retro(pre-commit-auto-stage-next): record missed AC and capture learnings

## v0.15.2 — 2026-06-03

### Changes

- bcb9424813c7778ef9510c51fa49a20b73e4e121 fix(serve): reap orphaned hero mcp processes with a parent-liveness watchdog <!-- drift-test:ignore -->

## v0.15.1 — 2026-06-02

### Changes

- 8baf4524c3437aeb5f97b518e40f72326479e87f fix(skill): embed SwiftUI mock screenshots as base64 data URIs

## v0.14.7 — 2026-06-01

### Changes

- 9ea7bf208f0f33ab48bf519682c2399ef1253b97 feat(spec): stamp completed_at on status→completed transition + backfill (peer-handed off)

## v0.14.6 — 2026-06-01

### Changes

- 969a18e2e31f16d49e5e3ad7c7e2f7358859797a fix(mock): move renderer selection from prompts to deterministic CLI signal

## v0.14.5 — 2026-05-31

### Changes

- e7304a21678221d3afb520018b62529a19fdf5f4 fix(upgrade): stop swallowing install errors and honor --force after a partial run

## v0.14.4 — 2026-05-31

### Changes

- d95acb9246a2a299116aaed66537b7f8130edcd8 chore(handoff): SNAPSHOT timestamp refresh
- 49c25a6816cbc969025ce560465705cce93f4381 chore(handoff): refresh NEXT/SNAPSHOT/peer-manifest after v0.14.3
- 6e81ad29ee58f2fd71c6519b95abdb82dae35838 fix(ui-designer): hoist MOCKUP_FILES return contract so subagent links surface

## v0.14.3 — 2026-05-31

### Changes

- 6468ec8bc33f0fd7d9e60289f0e33f92f8e01d74 fix(install): detect strand drift in PersistentPreRun + hero check <!-- drift-test:ignore -->

## v0.14.2 — 2026-05-31

### Changes

- ecc7c395dfd2acd60ba18aaba17f36ea63864835 fix(install): close codex cleanup hole + dedup managed MCP entries

## v0.14.1 — 2026-05-31

### Changes

- 15f225c72bae1984fa3a33b0b7fc9e2f18032ece fix(workspace): bound Locate walk-up via WithStopAt to isolate tests

## v0.14.0 — 2026-05-31

### Changes

- 4da926d639fefa9968bf04ea2438448814e72fcc chore(handoff): refresh SNAPSHOT + peer-manifest after supersede delivery
- 3fc989b33da16eb435851f8130f3ed9374c5957c feat(deliver): add cold-audit pass, headline delivery receipt, next-up suggestion
- f0bec7f53ff13a992a9ee72b87ed595fe766aece feat(retrieval): respect supersede signal in hybrid path with config kill-switch
- 33d3adbf6a508592c3d22afc8fd613fd6e5fae30 feat(rules): add honesty/pushback rules to agent instructions
- d7fd9e9152a582240d0fda93c99d26f2f4bd44d6 feat(superseded-specs-soft-archive): frontmatter-driven supersede genealogy
- 90b85787f6cbdaaaeb812627f51899394fe0ad98 fix(deliver): keep the file-inventory receipt on every delivery, not just noteworthy
- d843d036aa02fec721315033eb532cbee6ffdd8e fix(docs): swiftui-mockup-renderer references hero spec mock, not hero mock <!-- drift-test:ignore -->
- 1859b99a84b1f28b9d2fc2678af688c74461be31 fix(snapshot): resolve path-format parent references to slugs
- 0e0196597181e3803649bd97d9319d353366c40e fix(workflows): make mockup links reach the user and stop accepting PARTIAL as done

## v0.13.0 — 2026-05-30

### Changes

- bdedde68a94a64af3c5b10fd1f52273266bd364e feat(embeddings): embedded inference — zero-dependency semantic retrieval
- b9ff0653f6dd8a35e5105462515ddb50c1f865c9 fix(graph): tolerate partial migrations so schema upgrades self-heal

## v0.12.2 — 2026-05-25

### Changes

- 344b81aa2307ac78d0a6e65aba5165b03848e78e docs(agents): add internal-lookup tool routing guidance
- 18c6d39fd20ea48424f3b483ed3297cb1e6097b2 docs(agents): sync Go fallback with AGENTS.md tool routing addition
- f2da497adb10c654bf90f9cee329653337d088ed feat(graph): add hero graph edge add write CLI [pm-ui-brand-button-prereq] <!-- drift-test:ignore -->
- b0d39d5b5d7470458501269913ccd88cc2dd326e feat(graph): add hero graph node add write CLI [pm-ui-brand-button-prereq-2] <!-- drift-test:ignore -->
- 7a60709736d431cf2ddf9dd5a869085a21775fd8 fix(cli): hero search --json honored on all paths <!-- drift-test:ignore -->
- 09636c7df40eb6c2c37ca874868642ed10b471c8 spec(search): tiered response with max_results + pagination

## v0.12.1 — 2026-05-22

### Changes

- 1b340d1f763fa10d74fa2aed8eb684c70c7b5631 feat(engineering): install delivery-completion-discipline — Excellence Bar + Completion Ledger + symmetric anti-under-delivery rules
- 00ee6458a73beb5ccf33572f7fa029f929b05d70 fix(opsrunner): bump subprocess-kill timeout from 5s to 15s for CI reliability

## v0.12.0 — 2026-05-22

### Changes

- 84fb8e4cdd4c5193dce030ffdb996c2bdba456a5 docs(spec): add compact-handoff-test-coverage follow-up
- 8d1637cafe429bb94de537f0898ef052175f0d8d docs(spec): archive next-compact-handoff as completed
- 29b47161086ea76a2b8209ac08f5ebfd8ad5aa94 feat(mock): link mockups back to specs and surface them in /deliver
- b24151b0551fb4aaf6f991a16bcca0ec5c16644f feat(next): compact handoff — session-scoped resume envelope at compaction time
- 956f8c248cf3e20c5a7b49bda7d987702db08f60 feat(next): wire Codex SessionStart{compact} installer — spec complete
- 145710722196f68eb72d5533e9c8807aa76e4c8b feat(skills): read-don't-guess — replace passive assumption rule with procedural grounding check
- 8a9dd105af06e83e490a455abec05b88e1197d7f fix(next): transcript-fallback kickoff + uninstall reaches all hero git hooks <!-- drift-test:ignore -->
- a5621a44be46b91eb02fffb26aa22f656e258641 test(next): close compact handoff coverage gaps

## v0.11.0 — 2026-05-20

### Changes

- 0778b06014c334b95e346bd6d87c04d723d65b4c chore(content): remove root agents/, commands/, skills/, AGENTS.md
- 92c94aacb6703c23fccc770cbbad10f3f274d834 chore(domains): sync root → domains/engineering/ for legacy-fallback parity
- e77042c4183843beb715fae16c22133d0c27cc7b docs(knowledge): dashboard sprint plan, conventions, decisions, side bug
- 008df1a74d05e4f5d16ffe0060377da1f3ec3b24 docs(spec): archive contentfs-legacy-fallback-removal as completed
- d987b99641f9b5f91cf802c359eaa7cf336e9311 docs(spec): archive hero-code-handover-pack as completed (D9)
- 1a2cfe4bbcb8daeffbc00966269c4ce94302cd38 docs(spec): archive hero-serve-dashboard-redesign as completed
- ca0ed03ce4a6a5b9d49ab1e8c62d70d535f5d362 docs(spec): archive hero-serve-multi-project as completed
- a547d6fb467a255ff9fb9ced70d9da377da5c74a docs(spec): archive hero-serve-project-section initiative as completed
- 72c931316a2b08b9f4390419bc46ab22846c60d7 docs(spec): archive hero-serve-project-section-aggregate as completed
- 2bb883002e700adf6c6b1ae749917eab1c758190 docs(spec): archive hero-serve-project-section-destructive as completed
- eff693f45391b0662d459ad9b890d93984a8d925 docs(spec): archive hero-serve-project-section-mvp as completed
- 0e5ec1bc171815838829aa7ae1d2ac389d7aee68 docs(spec): archive hero-serve-project-section-opsrunner as completed
- 75172fe1de0505441d097d98def1caa328e9f69f docs(spec): archive pm-platform-delivery sprint as completed (D10)
- f0b85d4ed455150527a3590b9217b8c9d0fe0838 docs(spec): record follow-up advisory to hero-code on archived sprint
- 3cb7720e742107f5a9d4f7930948b751150111b6 feat(config): SprintConfig opt-in for planned-sprint UI
- 17757282978b23b06563f8abab3e7cd7b7c003ef feat(content): cut engineering through domain pack, drop legacyContent
- e9c2a7812b0100ee85142df7badf04e1664b169a feat(graph): DSKG Phase 1 — schema v3 with domain column (D5)
- e298fb5ec05b6d186fd4a3f136f8253202294a63 feat(graph): DSKG Phase 2 — write-path stamping + invariants (D6)
- 4d313434ec041390bf4e44643501f336b14506e7 feat(graph): DSKG Phase 3 — showcase read-path filtering + dashboard data layer (D7)
- 772a967d5806188da6a4d046ea8093360e5d2abf feat(install): layer universal core onto every install via OverlayFS
- a5aeb4cc46b7b4a2319c4e31e8d2965fc8b760db feat(install): pack AGENTS.md loader + domain-filtered agent materialization (D3)
- b711dbd10725972163183ce76ee6a4f7cd5bc2ba feat(scan): pluggable scan dispatch + codescan domain wire-through (D4)
- f293610d76f784b1c8de09c17d0ae094269c378a feat(serve): Now metric strip — rolling-window activity counts
- ec52e48c4d3ab3db2b9976fd218ab248d5961003 feat(serve): Now page redesign — activity-led layout with rolling windows
- 7a256a554aa0468e28c8d24c29a25347e5564cc3 feat(serve): Project section MVP — read-only per-project operator page
- 9654d6e070aa0679ad6217789f599e8b4206a27b feat(serve): Project section aggregate — /p/all/project cross-project view
- 6af73241564c36444680d83015f9cd0e49365a37 feat(serve): Project section destructive ops — remove grace window, Danger Zone, Stop daemon
- 5e5306da4388c4bf90924586b60fda056bf1b12a feat(serve): Project section healthcache — TTL-bounded check cache + peer probes
- 28b4ca3bc42bda35d90891bead9bc9fe30bd959b feat(serve): Project section opsrunner — lifecycle ops with SSE progress
- 7192081a0037d83f92c66813fcbbbeca3c20aadd feat(serve): Work page — default tab "This week", Sprint gate, themes row
- ffea739e5ec55efbf31616e586eb4e2cd1255913 feat(serve): cross-project aggregate routing for /p/all/now and /p/all/work
- b288086baf0c0c5aa4b91f88fb9388f65069c6aa feat(serve): daemon lifecycle (stop, status, --force, PID file)
- f4941ac7db75dd9f3ea551d5c277c3c4f48fdf30 feat(shell): make MetricTile clickable via optional Href
- 6f8b8ba84e028a5d2ed6ffedb7518b09fa5f4eff feat(spec): DSKG Phase 4 — spec frontmatter domain field; archive DSKG (D8)
- 589146f38c8b4205ab51603d71be167e89ec61d9 fix(peering): persist peer-call findings and drop 400-char stdout cap
- 9109a721183c5a3baecdb945a0642dd7424cf31c fix(peering): tolerant ApproxInt for peer-call budget fields
- 62a76c5a89c8e78b0536e5b0bfd3a72fe00b2709 fix(serve): broaden Now inbox sources and wire proposal snapshot
- 9fbeb7b279f6e048c6a028e0789a742b6d5ff338 fix(serve): emit delivery_complete event from spec completion path
- 8d2872d1fed2903664abfe204e0d0e25250a53d7 fix(serve): reconcile dashboard "you" identity with writer-side git config
- 5893de728427c7db71c505b20a3121cc43038652 fix(serve): source top-nav adapter chip from live registry probe
- d407deed3e91fce46500043345696a2301612edb fix(serve): stop pairing "no agent running" with "since X ago" in Now headline
- f99312748a02807a1914d6564c046092745e4c08 plan(pm): close D2 — archive spec-type-registry + inline-propose-output-mode
- 9cf21d0c069a558c01e11902aa3bd67e79cda59c plan(pm): design 3 primitives + author delivery sprint + close D1 (DPA)
- f50bb50fd700a12865be5b04e18a4f30d253db9b spec(plan): dashboard redesign + project section feature specs
- c91f8be4e69d0af8aaf5f08b7465822e84fe3174 spec(plan): split hero-serve-project-section into initiative + 5 phased children

## v0.10.3 — 2026-05-19

### Changes

- 5c658513e9463a7093d52f057a833293126d363e refactor(install): drop byte-equality gate from legacy-dir cleanup

## v0.10.2 — 2026-05-19

### Changes

- e9b47eaa9b70b0d799aedf62596bb7ba2cfc20e4 fix(install): force-cleanup legacy .hero/{agents,commands,skills}/ mirror dirs

## v0.10.1 — 2026-05-19

### Changes

- 79d361f625bbd7b10a27f0fa922a3fada8510669 chore(spec): archive accepted decision specs → .hero/specs/decisions/
- 8f2e393fdf272ebd077573d9f4db9db47160cb58 docs(drift): fix 15 real CLI-invocation drift hits surfaced by new test
- 2cc138aa3984609606c3419cd5ab41ed2effd89c docs(import): add CLI reference page for hero import <!-- drift-test:ignore -->
- 9901b4bbad5c8e0a8b857e557e2f21b4c67716e5 docs(import): add hero import CLI reference page content <!-- drift-test:ignore -->
- e53d8b37a948c14665aab9d4cffd76f890174930 docs(serve): add minimal route inventory for web UI homes
- 8528c2e73624edac797e9c395a7dc3eb3592129b feat(docs-check): markdown CLI-invocation drift detector
- c5d4505dcd8bce7a966a7fd28f1a960b590f99c0 feat(lifecycle): enforce spec contracts at transition sites
- 933a6a38b5b12eeb29bea7d7a693e6365cc77fe9 refactor(managed): consolidate AGENTS.md/CLAUDE.md/NEXT.md writers into one orchestrator
- 121b6681b1b083cb8e48bf66a7c05773c63211d3 spec(scaffold): six v0.10.1 follow-ups — drift test, consolidation, decision back-fills, docs gaps

## v0.10.0 — 2026-05-18

### Changes

- 4ea18c4bc64fd1ed458f6068b519d14d3c262535 Merge B3: internal/methodology/ Go package
- a935782ae83a36a42d3fc968db24eeeb66aab48e Merge B4: internal/tasks/ + hero task CLI <!-- drift-test:ignore -->
- bf655985eb80834bcde97068cf0a95bd3e806d8d Merge B5: inline-propose Go side
- 091ffca0bc11c037dcfccebff6e3dd28c82432d5 Merge C1: inline-propose v1 test fixtures
- 94911df2918427459bf89ae5cc2e0cdfe152c9d5 Merge C2: docs/contracts/README.md discovery index
- 7b7363eaaf0c764fbceca0b213e1549f9498afef Merge C3: active-dialect read-path doc
- 4b38c32af9ef35a39f015ce75bbf3c941d11104a Merge C4: JSON Schema for spec-types-v1.1
- fdd54ee45f0ff2c3dee144458c040aa476f584b3 Merge C5: examples/scrum-workspace sample
- 6e4e451afdfe152139b6621f4bc4a4cd535f9a7b Merge bug-fix: align initiative prose with YAML required-sections
- 7ba6614af4c81b5182f65875474c77e2d6b7bd48 Merge bug-fix: hero.local.json forwards dialect fields through MergeLocal
- 39516dcf76a9413e47f4c6cab3d0945214f0daba Merge bug-fix: populate frontmatter schema on 10 spec types
- 04ef53a4afd431666417698dc65dcc07251881d3 Merge bug-fix: vocabulary.Resolve folds methodology-derivation
- 419a4c7e4774059eb7c5ac86aab9023fd78abf06 Merge feature: document vocabulary auto_select schema
- 0b79d2f649076bff033b722c8ae30fb53985805d build(docs): wrangler.toml for Cloudflare Workers static assets
- 313d6fe16810ead83408d8d7b88649e17f205785 chore(peering): record hero-code advisory handoff trail
- 2952923cd6de80a15767467d549106dbcfe624a6 chore(peering): record hero-code handover-pack advisory trail
- 812320500a5fd48d53a6dd062f2fb08322793469 chore(peering): register hero-code + hero-cloud as peers; fix doc drift
- 317673667333263552109b1bdb60648cf7b5f62c chore(spec): archive hero-trust-global-scope to specs/
- cdbdc6ebaa75e38fe325811731ab5c3885f45f4f chore(spec): archive next-checkpoint-cross-repo-pollution → specs/
- ae6da6fbe8757ded8028727e8bd97cd79721740a chore(spec): move next-as-projection to .hero/specs/ via hero spec complete <!-- drift-test:ignore -->
- e7d95949a7bb49cb5d095ed224f4ceb3991c5315 ci: NEXT.md projection drift gate (AC-12)
- 292ba85a968ab309c4d3191c1b1e87074c8688e3 docs(contracts): active-dialect read contract for consumers
- 7c661bae8e6f0ab4e4b37770cee1cd2b13520ed2 docs(contracts): discovery index for the four cross-language contracts
- 728671476c0b8d8345bcf3989359279c3a5e9ff2 docs(contracts): publish spec-types v1.1 JSON Schema
- 2986d05c8aebc635aa575ef3c95dafb76b1ecc35 docs(contracts): vocabulary auto_select authoring contract
- 95ce020379ec45d6dc7482433c3d5cd09a44093d docs(convention): contracts package import discipline
- b9b6a66f917853a9ce9d097b7579f9b53a6f44bc docs(landing): lead with natural-language routing
- 8bf5666a8a51bec78bf23b9e91fff760b81d198c docs(landing): polish loop-section copy on heroengine.ai
- 71bc34351424bc4690a4fbc3cf53e98215aa08eb docs(peering): cross-repo peering setup guide + README integration
- c9571adcd8e69d8b96403ebd6b448f5eb7046565 docs(spec): hero-cloud-split phase 1 landed; phase 2 next
- 3f95efd6636ce919e14704979a2bf4aeda666d56 docs(spec): hero-cloud-split — workspace init landed in hero-cloud
- aaa3ae2c1c63e3bd83452272f8beb16e43d0292d docs(v0.10): refresh for snapshot, peering, NEXT projection + fix stale hero pull <!-- drift-test:ignore -->
- 07a0403712e001f56b3fef8b078c38ecd18d4ba2 feat(cli): pre-flight migration gate for hero next checkpoint (AC-14) <!-- drift-test:ignore -->
- c269b604035aba3a906b92b2bbf9462d7f23c8dd feat(cli): wire spec-types.json cache export into PersistentPreRun
- 4427cec5960fd8f5ff1f89baefcf7e4e4f70d6d0 feat(contracts): add contracts package skeleton + import discipline (hero-cloud-split phase 0)
- 1763417f3c0314c6a10174125f6bfaa27f4580e7 feat(contracts): enforce cloud→contracts boundary + migrate cloud imports (hero-cloud-split phase 0)
- a0873cc43ef635fa3464f0ce2a7deed5a15087ff feat(contracts): inline-propose v1.0 test fixtures for hero-code
- e81a7ad025ff1af18addadf17f49f64b03062694 feat(domains): chat — content bundle for hero-chat Sprint 4
- fbcbe7739bebe2e64fa8b939a915918fd2aebdbe feat(domains): finish B1 cutover — verify PM pack + lock ContentFS decision
- 04f8af2fa159f1cc2b5a4b10ad156fd8040768eb feat(examples): scrum-workspace sample for hero-code consumers (C5)
- 522e4a9502452509b20547b89495829945601f3c feat(install): SessionStart hook fires hero next ingest for cross-machine continuity <!-- drift-test:ignore -->
- 8e07c9a3b697222b4f7cf0539bdb679129cbbdf8 feat(install): route cross-repo peering intents from natural language
- 61a80f2feb12a31249bf64d3b31e2d80c0c9b65b feat(landing): add heroengine.ai landing page under web/landing/
- b565efe1c238e62499566902a27942db43e7f37c feat(methodology): internal/methodology Go package (B3)
- 19b691833c278183e8bc46b9bc4701b37993d8ed feat(peering): /peer slash command + cross-repo-peering skill
- 4fa06a72b3b4a7e46bc857180d710b8124fec655 feat(peering): contract-import passive surfacing (cross-repo-peering phase 3)
- a62461abc9e930c5ba1d4d1290e5348ae601f991 feat(peering): handoff lifecycle + trail + async drop (cross-repo-peering phase 1)
- b3aa7763a4da8fd30210b779a1e310ea1bdc3293 feat(peering): hero peer CLI tree + relevant --peer fallback (cross-repo-peering phase 2) <!-- drift-test:ignore -->
- e0ce3c071b687b927ab5e67bd224680df70ee7df feat(peering): mint peer UUIDs + contracts/peering/ + dual-key resolver (cross-repo-peering phase 0)
- 2b53b26d61b51ff6a9ce22a520f27d6e1639238b feat(peering): pilot harness + peering-protocol doc (cross-repo-peering phase 3)
- fbadf805fef1452182adbf40644fe10406090f02 feat(peering): sync peer call dispatcher + auto-fire reconciler (cross-repo-peering phase 2)
- bc280f7eacd5738b2a50395ba7b2d2f568987672 feat(pm): finish domain-plugin-architecture cutover (B1)
- 164041477e3034e22dc11323f5c39d2b65ced7c0 feat(pm-foundation): land PM pack, core spec-types + methodologies, vocabulary package
- 47f08d89274bec6793ea581b9490e6abdbf308da feat(propose): inline-propose Go side — shim, daemon store, REST/SSE
- d71a47db6cbd70d92a38afbebba2ea06200302ac feat(rendering): vocabulary + methodology-aware rendering spread (B6)
- ccb091a3c13790abf8ba994c1950b47d7d549faf feat(serve): close Now home follow-ups — chat-input fragment, hero SSE, no-adapter notice, live session ledger
- 986806e05aea35e3c2b759dd51cdc62d06c07b64 feat(serve): land Hero Now home — personal cold-start surface at /now
- 70265f7545deae63154e91f9be6f133920382997 feat(serve): land Hero Surface Architecture initiative + ship surface shell
- 95e57c7a92869f7c2392fb36756e8749edf80351 feat(serve): land Hero Work / Knowledge / Agents / People homes — initiative complete
- d3eb4a1671a0533625af80aa88066ae444adde18 feat(serve): land Hero chat dispatcher + ⌘K overlay + runner-free slashes
- 424bb36de663aff4e77064b5ca172ebfcc57cabb feat(serve): polish v1 — fix 22 sub-route 404s, default views, Now data, Work firehose
- 9c4d61f7da6627be87ffbd02050185bdb5cae00d feat(serve): polish v2 — per-item detail routes, filter reconcile, CSS + verb cleanup, chat-input on all homes
- 95f20d030f5240ff7bb550c17a0b0fab547a1413 feat(serve): polish v3 — disabled chat, detail breadcrumbs, feed dedup, mdrender tables, sub-route titles
- 83c767c94db3914790656884c53cb77ba8607411 feat(serve): polish v4 — view-tab active, mdrender wrapped li + escaped pipes, dedup groups, detail titles, table CSS
- 9f988e536b764a68fa94dcd53fd70d7d075b6a11 feat(serve): polish v5 — knowledge dir-style entries, agents root title, SSE toolbar route hint, mdrender escape state
- 5110b1adec3c53ad3271df6b8a0155c60cead6ba feat(snapshot): land project-snapshot — projector, CLI, MCP, /project home, archive containment
- 1936d0cb513556538f60efbdb44b45ad9f44f213 feat(spectypes): registry loader, schema 1.1 export, lint parity (B2)
- 17f79c6ff2a4e4c074a474cc88456d98a7699ec3 feat(split): hero-cloud-split phase 3 — remove cloud trees from hero
- 52c7f99d9630034fe623fd91f5ceac53dd95ddc2 feat(tasks): additive internal/tasks/ package + hero task CLI (B4) <!-- drift-test:ignore -->
- 9ee9b5271872828563258bd1aeb196a664a0fe7a feat(trust): add optional <project|global> scope to `hero trust claude` <!-- drift-test:ignore -->
- 48ceaa3fc1021027608d7807d8253f5b9b0032c1 fix(config): forward vocabulary + methodology fields through MergeLocal
- 0b973b2e403e0b79fe38561e28878e1464b5fbd7 fix(docs): contrast on active sidebar nav links
- 0cfe4038228dfa604c32805804d052db9efd51a4 fix(next): stop .local.md narrative accumulation + repo-scope handoff reads
- a8445588362206e67bd29a0e516fe27128fde8d0 fix(recap): treat empty repo as empty recap, not error
- c49bb2d47a1555fc16587544ccf78db3d18c98d8 fix(spec-types): align initiative prose with YAML required-sections
- 7c5da1dd6d599f0f5073562ecad0f6383f7a8934 fix(spec-types): author initiative frontmatter block (closes Bug 2 gap)
- 00a2962faa9ddd5394cc5825c0f99dfb8f3ae780 fix(spectypes): populate frontmatter schema on 10 spec types
- 688ec693a525b7384cd33059f8179824bf38c964 fix(vocabulary): fold methodology-derived precedence into Resolve
- 8933fb9624f9861c8b86629bbc279d23aeb00054 fix: correct .envrc gitignore pattern (trailing comment broke matching)
- 69e926e0c54103136a4745642867d03bf68bff9f refactor(web): make docs and landing peer surfaces under web/
- 97c85bdb4216d9c510d725257ea3f320ca514791 skills(next-merge-recovery): self-heal protocol for projected NEXT files (AC-16)
- 5f8331ea49818b818b88c76ad7a3da0b1afb13b9 spec(bug): hero install --target claude emits both CLAUDE.md and AGENTS.md <!-- drift-test:ignore -->
- 545cef8c7263b2d5d2820c9ff78034e1aa392834 spec(contract-gaps): file 5 follow-up specs from hero-code-handover-pack
- 2289d890c8d8f94241f05ce6c90b374707d4acd9 spec(hero-code-handover-pack): sprint manifest + completion checklist
- 31609b10b275e6be51b53a6f70e445a974a2965e spec(inline-propose-v1.1): file gated post-hero-code anchor.value spec
- 7807117afca6e3d99abc6e2fc89987b92f3c39d1 spec(next-as-projection): correct audit — keep `checkpoint` name, drop `set` subcommands, flip to delivering
- 61efffbe17b8cb5f01616e0bd9b5c459b49a53b6 spec(next-as-projection): mark completed, refresh Kickoff with delivery summary
- 5318d9c3bf9e70fa48a59be26d7d9d9058de8b42 spec(peering): amend with Phase 0-3 reality + Phase 4 polish items
- d80c4a21deeef96c2337ed20d14ede75968b35a9 spec(plan): hero-pm design pass — agent pack, mockups, handoff doc
- 9d3a1923df1f9a9d1aa1327a129fc3e545789f8d spec(plan): hero-qa design pass — locks, research, agent pack, mockup brief
- 651b35b15f8d53c7daa8d66b00229ad98c0402e6 spec(plan): hero-qa handoff-to-hero-code for implementation kickoff
- 441d2c9a72a988e6f9c98b12002659965092e64d spec(plan): hero-qa mockups — AI-surface retrofit + new Screen 09
- 662c203d9dd0a56566806c21e467e4245493375a spec(plan): hero-qa mockups — eight surfaces + landing index
- 89ea70f3d84aa0560cb93cfc5c04523109e61b50 spec(plan): hero-trust-global-scope; trust self + queue refresh
- 7b202c10e877911e27eae0962e9322891932be1f spec(plan): reshape hero-domains to PM-first; scaffold six child stubs
- d7da7c864259cfc2fd267fc3a893b96d355ea432 spec(planning): hero-serve-scope-decision (high priority, ASAP-but-not-now)
- 93a80247a935cd19d4cdda3a30abf01e4dea894b spec(planning): inline-propose output mode + tracker-fronting decision
- 258b8123dfae61d7d24f5588d17ecd66e56eb10f spec(planning): land governance + community-edition designs + recap-register scaffold
- 9fdb85edf3ecf3ca113b86068ac36b4e64c607cd spec(planning): remove cloud-side specs migrated to hero-cloud
- da1ccb68fdd72fb510fc8823e52bc3590bbc7d2f test(cli): E2E cross-machine round-trip via two ephemeral graph DBs (AC-6, AC-11)

## v0.9.2 — 2026-05-15

### Changes

- ab44dfa7c2f2c135d0791eb6cf9733535da95f8b decision(domain): adopt heroengine.ai as canonical domain
- f187b28084c4c62b6a6fe066e2c5d13f10334a97 feat(dist): scoop bucket + install scripts for Linux/Windows

## v0.9.1 — 2026-05-14

### Changes

- c60dce8fb0ac2075e02aff29c708785935a20c63 fix(install): always regenerate managed regions; drop --force-managed

## v0.9.0 — 2026-05-13

### Changes

- 16e030fe5a56fdf57e95c872d05999e42dc55c9e fix(agents): add name+description frontmatter to all agent files
- 5fabd7a7c6e7eac7292c110795d28f40de587df8 fix(claude): write Bash(hero:*) permission allowlist on install
- 4fb54d55cc88aa805d4c90e3012a662e14231afe fix(install): make canonical content materialization idempotent
- ef9b994c2785561974cf0b646105a39013bff159 fix(scan,install): unbreak fresh-install onboarding for non-Claude-Code harnesses
- c9b76e7b8ee1a0a78b9bffc8aba6d9a007741114 refactor(install): render-direct architecture; auto-sync siblings; Layer 1 verification
- c9934c30d494f6c9a927dc254009fb94d16216f5 release: place brew formula under Formula/ subdirectory
- 170e23f0648f51edbc6d4cf6e97ba60f1b469ba3 spec(complete): claude-trust-permission-allowlist
- 1542ec1ad28dda1725991dce4796ddd17387c80b spec(complete): multi-harness-install-collision
- ad869b82008f6d1fb672a99ce261012a2f8f0217 spec(complete): scan-output-cleanup
- d3dc74cf2fc1c2d037aaba3f3481b697d58dcb75 ux(scan): clean up three first-five-minutes papercuts

## v0.8.1 — 2026-05-12

### Changes

- 686166dc1ecad710f938dd6c9a15912039206ac4 release: wire up homebrew tap via goreleaser
