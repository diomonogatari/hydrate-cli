// Package assets bundles the install-time files (systemd units, zsh hook) into
// the binary so `hydrate init` can wire things up without the source tree
// present. Living in its own package lets the embed sit next to the files while
// the main package moves under cmd/.
package assets

import "embed"

// FS holds the embedded install assets: systemd/* and zsh/*.
//
//go:embed systemd zsh
var FS embed.FS
