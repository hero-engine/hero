# scrum-workspace — sample Hero workspace for hero-code development

This is a working scrum workspace for hero-code development. It declares
`methodology: scrum` plus `vocabulary: agile-scrum`, and ships a handful
of realistic e-commerce specs across the lifecycle so hero-code can
render real content without having to author synthetic test data.

## How to use

```sh
cp -r examples/scrum-workspace /tmp/acme
cd /tmp/acme
hero status
```

`hero status` should show the active layer with `methodology: scrum` and
`vocabulary: agile-scrum`, and the four specs as Stories / Tech Debt
Stories / Bugs / Initiatives per the agile-scrum display map.

To point hero-code's Rust dashboard at it, set the workspace path to
`/tmp/acme` (or wherever you copied it).

## What's inside

```
examples/scrum-workspace/
├── README.md                            # this file
└── .hero/
    ├── hero.json                        # workspace config (scrum + agile-scrum)
    └── planning/
        ├── features/
        │   ├── checkout-redesign/spec.md           # Story, planning
        │   ├── cart-abandonment-emails/spec.md     # Story, delivering, child of q3-conversion-lift
        │   └── express-checkout/spec.md            # Story, completed, child of q3-conversion-lift
        ├── bugs/
        │   └── promo-code-rounding/spec.md         # Bug, planning, severity minor
        └── initiatives/
            └── q3-conversion-lift/spec.md          # Initiative, planning, two children
```

Spec inventory:

- **checkout-redesign** — single-page checkout feature in `planning`.
- **cart-abandonment-emails** — recovery-email feature in `delivering`
  with three acceptance criteria and one open task (`T-1`).
- **express-checkout** — Apple/Google Pay feature in `completed`.
- **promo-code-rounding** — rounding bug in `planning` with full repro
  steps and a diagnosis.
- **q3-conversion-lift** — quarterly initiative in `planning`, with
  `cart-abandonment-emails` and `express-checkout` declared as children
  via the `relations:` frontmatter.

## Note on workflow

This workspace is a **static fixture**. Running `hero install` or other
state-mutating commands inside it from the source tree is not the
intended workflow — they will rewrite files in this checked-in
directory. Copy the workspace out (`cp -r` to a temp dir) before
exercising any mutating command on it.

The vocabulary is declared explicitly (`vocabulary: agile-scrum`) even
though it would auto-derive from `methodology: scrum`. The redundancy
is intentional: it makes the example self-documenting for anyone
reading the config without consulting the resolver precedence chain.
