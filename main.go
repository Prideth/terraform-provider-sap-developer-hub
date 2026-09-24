package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/provider"
)

// version is set via -ldflags "-X main.version=..." by GoReleaser at
// release time. It is "dev" for local builds.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "start the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/Prideth/sap-developer-hub",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
