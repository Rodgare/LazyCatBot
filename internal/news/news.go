package news

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"strings"
)

//go:embed news.md
var raw string

// Content is the announcement body written in news.md.
func Content() string {
	return strings.TrimSpace(raw)
}

// Version is derived from the content, so editing news.md and deploying makes
// the bot deliver the new text exactly once.
func Version() string {
	sum := sha256.Sum256([]byte(Content()))
	return hex.EncodeToString(sum[:8])
}
