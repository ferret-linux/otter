// Package docsfs embeds otter's documentation (the repository's docs/
// directory) into the binary so `otter docs` can browse it without
// depending on the docs/ directory existing on disk at runtime.
//
// This lives at the repo root — rather than under internal/ alongside the
// command that uses it — because Go's //go:embed can only reach paths
// within or below the directory containing the directive, and docs/ is a
// root-level sibling. Duplicating docs/ into an internal/ package would
// avoid that constraint but risks the copy drifting from the real docs/;
// this keeps a single source of truth instead.
package docsfs

import "embed"

// FS is the embedded contents of docs/, rooted at "docs" (i.e. paths look
// like "docs/getting-started.md", "docs/commands/otter-list.md").
//
//go:embed docs
var FS embed.FS
