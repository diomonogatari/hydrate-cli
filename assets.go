package main

import "embed"

// assets bundles the install-time files (systemd units, zsh hook) into the
// binary so `hydrate init` can wire things up without the source tree present.
//
//go:embed packaging
var assets embed.FS
