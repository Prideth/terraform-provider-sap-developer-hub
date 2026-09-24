package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	tfprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestResources_ImplementConfigureAndImport mirrors
// terraform-provider-integration-suite's structural consistency test: every
// registered resource must implement ResourceWithConfigure and
// ResourceWithImportState, and must have a non-empty schema.
func TestResources_ImplementConfigureAndImport(t *testing.T) {
	p := New("test")()
	for _, factory := range p.Resources(context.Background()) {
		r := factory()

		if _, ok := r.(resource.ResourceWithConfigure); !ok {
			t.Errorf("resource %T does not implement resource.ResourceWithConfigure", r)
		}
		if _, ok := r.(resource.ResourceWithImportState); !ok {
			t.Errorf("resource %T does not implement resource.ResourceWithImportState", r)
		}

		var metaResp resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "developerhub"}, &metaResp)
		if metaResp.TypeName == "" {
			t.Errorf("resource %T returned an empty TypeName", r)
		}

		var schemaResp resource.SchemaResponse
		r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("resource %s schema has diagnostics errors: %v", metaResp.TypeName, schemaResp.Diagnostics)
		}
		if len(schemaResp.Schema.Attributes) == 0 && len(schemaResp.Schema.Blocks) == 0 {
			t.Errorf("resource %s has an empty schema", metaResp.TypeName)
		}
	}
}

// TestDataSources_ImplementConfigure is the data source counterpart.
func TestDataSources_ImplementConfigure(t *testing.T) {
	p := New("test")()
	for _, factory := range p.DataSources(context.Background()) {
		d := factory()

		if _, ok := d.(datasource.DataSourceWithConfigure); !ok {
			t.Errorf("data source %T does not implement datasource.DataSourceWithConfigure", d)
		}

		var metaResp datasource.MetadataResponse
		d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "developerhub"}, &metaResp)
		if metaResp.TypeName == "" {
			t.Errorf("data source %T returned an empty TypeName", d)
		}

		var schemaResp datasource.SchemaResponse
		d.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("data source %s schema has diagnostics errors: %v", metaResp.TypeName, schemaResp.Diagnostics)
		}
		if len(schemaResp.Schema.Attributes) == 0 {
			t.Errorf("data source %s has an empty schema", metaResp.TypeName)
		}
	}
}

func TestProvider_Metadata(t *testing.T) {
	p := New("1.2.3")()

	var resp tfprovider.MetadataResponse
	p.Metadata(context.Background(), tfprovider.MetadataRequest{}, &resp)

	if resp.TypeName != "developerhub" {
		t.Errorf("expected TypeName %q, got %q", "developerhub", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("expected Version %q, got %q", "1.2.3", resp.Version)
	}
}

func TestProvider_Schema(t *testing.T) {
	p := New("test")()

	var resp tfprovider.SchemaResponse
	p.Schema(context.Background(), tfprovider.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema has diagnostics errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"url", "token_url", "client_id", "client_secret"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("expected provider schema to have attribute %q", attr)
		}
	}
	if !resp.Schema.Attributes["client_secret"].IsSensitive() {
		t.Error("expected client_secret to be marked Sensitive")
	}
}
