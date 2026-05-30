// Package defaultmodel embeds the hero-embed-v1 model weights into the
// binary. The files are produced by tools/distill-embeddings.py (one-time
// export from minishlab/potion-base-8M, pruned of dead subword tokens).
package defaultmodel

import _ "embed"

//go:embed vocab.txt
var Vocab []byte

//go:embed weights.bin
var Weights []byte

//go:embed config.json
var Config []byte
