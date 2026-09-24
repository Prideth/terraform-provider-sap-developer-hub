//go:build tools

// Package tools pins build-time-only tool dependencies (currently
// tfplugindocs) in a module separate from the provider's own go.mod, so the
// tool's dependency requirements never affect the provider build.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
