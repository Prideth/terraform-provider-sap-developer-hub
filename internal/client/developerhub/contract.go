package developerhub

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
)

const metadataPath = "/odata/1.0/data.svc/$metadata"

// RequiredEntitySet is one entity set of the Developer Hub OData service this
// provider depends on, with every property and navigation property it reads
// or writes. Each name is taken from SAP's documentation, cited in
// DESIGN.md §5/§9.
type RequiredEntitySet struct {
	Name                 string
	Properties           []string
	NavigationProperties []string
}

// Contract is the complete set of OData entity sets, properties and
// navigation properties this provider relies on. ValidateContract checks it
// against a live tenant's $metadata document, so every acceptance run
// verifies the provider against the newest version of the API actually
// deployed, not just against the documentation it was written from
// (DESIGN.md §20). Keep it in sync with applications.go, attributes.go and
// subscriptions.go: anything added there must be added here.
var Contract = []RequiredEntitySet{
	{
		Name:                 "Applications",
		Properties:           []string{"id", "version", "title", "description", "callbackurl", "developer_id", "app_key", "app_secret"},
		NavigationProperties: []string{"ToAttributes"},
	},
	{
		Name:       "Attributes",
		Properties: []string{"name", "value", "entityType", "entityId"},
	},
	{
		Name:                 "Subscriptions",
		Properties:           []string{"id", "app_id", "product_id", "developer_id", "isSubscribed", "status"},
		NavigationProperties: []string{"ToApplication", "ToAPIProduct"},
	},
}

// ServiceMetadata is the subset of an OData EDMX document ValidateContract
// needs: which entity sets exist and which properties their entity types
// declare. Element matching is by local name only, so it works regardless of
// which EDM namespace version the service declares.
type ServiceMetadata struct {
	EntitySets map[string]EntityTypeInfo
}

// EntityTypeInfo lists the declared properties of one entity type.
type EntityTypeInfo struct {
	TypeName             string
	Properties           map[string]bool
	NavigationProperties map[string]bool
}

type edmx struct {
	DataServices struct {
		Schemas []edmSchema `xml:"Schema"`
	} `xml:"DataServices"`
}

type edmSchema struct {
	Namespace   string `xml:"Namespace,attr"`
	EntityTypes []struct {
		Name       string `xml:"Name,attr"`
		Properties []struct {
			Name string `xml:"Name,attr"`
		} `xml:"Property"`
		NavigationProperties []struct {
			Name string `xml:"Name,attr"`
		} `xml:"NavigationProperty"`
	} `xml:"EntityType"`
	EntityContainers []struct {
		Name       string `xml:"Name,attr"`
		EntitySets []struct {
			Name       string `xml:"Name,attr"`
			EntityType string `xml:"EntityType,attr"`
		} `xml:"EntitySet"`
	} `xml:"EntityContainer"`
}

// ParseServiceMetadata parses an OData v1/v2 EDMX document. Go's
// encoding/xml never resolves external entities, so parsing a document from
// the network carries no XXE risk.
func ParseServiceMetadata(data []byte) (*ServiceMetadata, error) {
	var doc edmx
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing OData $metadata document: %w", err)
	}

	types := map[string]EntityTypeInfo{}
	for _, schema := range doc.DataServices.Schemas {
		for _, et := range schema.EntityTypes {
			info := EntityTypeInfo{
				TypeName:             et.Name,
				Properties:           map[string]bool{},
				NavigationProperties: map[string]bool{},
			}
			for _, p := range et.Properties {
				info.Properties[p.Name] = true
			}
			for _, n := range et.NavigationProperties {
				info.NavigationProperties[n.Name] = true
			}
			types[et.Name] = info
			if schema.Namespace != "" {
				types[schema.Namespace+"."+et.Name] = info
			}
		}
	}

	meta := &ServiceMetadata{EntitySets: map[string]EntityTypeInfo{}}
	for _, schema := range doc.DataServices.Schemas {
		for _, container := range schema.EntityContainers {
			for _, set := range container.EntitySets {
				info, ok := types[set.EntityType]
				if !ok {
					info, ok = types[localName(set.EntityType)]
				}
				if !ok {
					info = EntityTypeInfo{TypeName: set.EntityType, Properties: map[string]bool{}, NavigationProperties: map[string]bool{}}
				}
				meta.EntitySets[set.Name] = info
			}
		}
	}
	return meta, nil
}

func localName(qualified string) string {
	if i := strings.LastIndex(qualified, "."); i >= 0 {
		return qualified[i+1:]
	}
	return qualified
}

// ValidateContract reports every entity set, property or navigation property
// in Contract that meta does not declare. An empty result means the live
// service still offers everything this provider uses.
func ValidateContract(meta *ServiceMetadata) []string {
	var problems []string
	for _, required := range Contract {
		info, ok := meta.EntitySets[required.Name]
		if !ok {
			problems = append(problems, fmt.Sprintf("entity set %q is missing", required.Name))
			continue
		}
		for _, p := range required.Properties {
			if !info.Properties[p] {
				problems = append(problems, fmt.Sprintf("entity set %q (type %s) has no property %q", required.Name, info.TypeName, p))
			}
		}
		for _, n := range required.NavigationProperties {
			if !info.NavigationProperties[n] {
				problems = append(problems, fmt.Sprintf("entity set %q (type %s) has no navigation property %q", required.Name, info.TypeName, n))
			}
		}
	}
	sort.Strings(problems)
	return problems
}

// GetServiceMetadata fetches and parses the Developer Hub OData service's
// $metadata document. $metadata is mandated by the OData protocol for every
// OData service, so this is not a Developer Hub-specific endpoint.
func (c *Client) GetServiceMetadata(ctx context.Context) (*ServiceMetadata, error) {
	body, err := c.getXML(ctx, metadataPath)
	if err != nil {
		return nil, err
	}
	return ParseServiceMetadata(body)
}
