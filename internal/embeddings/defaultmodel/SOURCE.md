# hero-embed-v1 source

`hero-embed-v1` is a pruned export of the MIT-licensed
[`minishlab/potion-base-8M`](https://huggingface.co/minishlab/potion-base-8M)
Model2Vec model.

- Upstream revision: `bf8b056651a2c21b8d2565580b8569da283cab23`
- Upstream license declaration: `license: mit` in the model card at that revision
- Model2Vec MIT copyright: 2024 Thomas van Dongen
- Base-model lineage: the pinned model card identifies
  [`BAAI/bge-base-en-v1.5`](https://huggingface.co/BAAI/bge-base-en-v1.5),
  also MIT-licensed (FlagEmbedding copyright 2022 staoxiao)
- Captured license text: `potion-base-8M-MIT.txt`
- Transformation: `tools/distill-embeddings.py` exports the static tensor and
  removes BERT special tokens plus `##` subword rows that Hero's tokenizer
  cannot produce
- Local `weights.bin` SHA-256: `bf3444799ca188d35fbd424d73539b3aeb2b889324d71a44ea9913550e5cce49`
- Local `vocab.txt` SHA-256: `28f351e6d87bbceee58fa12c9a7df72ee4230f81fcd6f9dc8dbbb6838d17ef6b`
- Local `config.json` SHA-256: `a56e14a3eb050d5d63d87480fb5eb565895c1b1bf992eac810707c1a7154b046`

The MIT notice for this derivative must be included in Hero's distributed
third-party notices. The upstream model's license remains MIT; Hero's license
does not replace it.
