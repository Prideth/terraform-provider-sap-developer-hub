package developerhub

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// The SubscriptionsType entity type below is copied verbatim from SAP's
// documented EDMX in
// create-or-update-or-read-an-application-using-subscription-key-e2645b5.md.
// ApplicationsType and AttributesType are test fixtures built from the
// properties SAP's documented payloads use; their type names are not
// documented and are deliberately different from the entity set names, to
// prove matching goes through the EntityContainer, not through type names.
const testMetadata = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
 <edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <Schema Namespace="developer" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
   <EntityType Name="SubscriptionsType">
    <Key><PropertyRef Name="id"/></Key>
    <Property Name="id" Type="Edm.String" Nullable="false" MaxLength="256"/>
    <Property Name="reg_id" Type="Edm.String" MaxLength="256"/>
    <Property Name="app_id" Type="Edm.String" MaxLength="256"/>
    <Property Name="product_id" Type="Edm.String" MaxLength="256"/>
    <Property Name="developer_id" Type="Edm.String" MaxLength="256"/>
    <Property Name="ratePlan_id" Type="Edm.String" MaxLength="256"/>
    <Property Name="validFrom" Type="Edm.DateTime"/>
    <Property Name="validTo" Type="Edm.DateTime"/>
    <Property Name="app_name" Type="Edm.String" MaxLength="255"/>
    <Property Name="isSubscribed" Type="Edm.Boolean"/>
    <Property Name="status" Type="Edm.String" MaxLength="255"/>
    <Property Name="comment" Type="Edm.String" MaxLength="2048"/>
    <Property Name="created_by" Type="Edm.String" MaxLength="255"/>
    <Property Name="created_at" Type="Edm.DateTime"/>
    <Property Name="modified_by" Type="Edm.String" MaxLength="255"/>
    <Property Name="modified_at" Type="Edm.DateTime"/>
    <NavigationProperty Name="ToApplication" Relationship="developer.Subscriptions_ApplicationsType" FromRole="SubscriptionsDependent" ToRole="ApplicationsDependent"/>
    <NavigationProperty Name="ToRatePlan" Relationship="developer.Subscriptions_RatePlansType" FromRole="SubscriptionsDependent" ToRole="RatePlansDependent"/>
    <NavigationProperty Name="ToAPIProduct" Relationship="developer.Subscriptions_APIProductsType" FromRole="SubscriptionsDependent" ToRole="APIProductsPrincipal"/>
   </EntityType>
   <EntityType Name="TestApplicationType">
    <Key><PropertyRef Name="id"/></Key>
    <Property Name="id" Type="Edm.String" Nullable="false"/>
    <Property Name="version" Type="Edm.String"/>
    <Property Name="title" Type="Edm.String"/>
    <Property Name="description" Type="Edm.String"/>
    <Property Name="callbackurl" Type="Edm.String"/>
    <Property Name="developer_id" Type="Edm.String"/>
    <Property Name="app_key" Type="Edm.String"/>
    <Property Name="app_secret" Type="Edm.String"/>
    <NavigationProperty Name="ToAttributes" Relationship="developer.Rel1" FromRole="A" ToRole="B"/>
   </EntityType>
   <EntityType Name="TestAttributeType">
    <Property Name="name" Type="Edm.String"/>
    <Property Name="value" Type="Edm.String"/>
    <Property Name="entityType" Type="Edm.String"/>
    <Property Name="entityId" Type="Edm.String"/>
   </EntityType>
   <EntityContainer Name="APIMgmt">
    <EntitySet Name="Applications" EntityType="developer.TestApplicationType"/>
    <EntitySet Name="Attributes" EntityType="developer.TestAttributeType"/>
    <EntitySet Name="Subscriptions" EntityType="developer.SubscriptionsType"/>
   </EntityContainer>
  </Schema>
 </edmx:DataServices>
</edmx:Edmx>`

func TestValidateContract_DocumentedMetadataSatisfiesContract(t *testing.T) {
	meta, err := ParseServiceMetadata([]byte(testMetadata))
	if err != nil {
		t.Fatalf("ParseServiceMetadata: %v", err)
	}
	if problems := ValidateContract(meta); len(problems) != 0 {
		t.Fatalf("expected no contract problems, got: %v", problems)
	}
}

func TestValidateContract_ReportsRemovedProperty(t *testing.T) {
	changed := strings.Replace(testMetadata, `<Property Name="status" Type="Edm.String" MaxLength="255"/>`, "", 1)
	meta, err := ParseServiceMetadata([]byte(changed))
	if err != nil {
		t.Fatalf("ParseServiceMetadata: %v", err)
	}
	problems := ValidateContract(meta)
	if len(problems) != 1 || !strings.Contains(problems[0], `"status"`) {
		t.Fatalf("expected exactly one problem about the removed status property, got: %v", problems)
	}
}

func TestValidateContract_ReportsRemovedNavigationProperty(t *testing.T) {
	changed := strings.Replace(testMetadata, `<NavigationProperty Name="ToAttributes" Relationship="developer.Rel1" FromRole="A" ToRole="B"/>`, "", 1)
	meta, err := ParseServiceMetadata([]byte(changed))
	if err != nil {
		t.Fatalf("ParseServiceMetadata: %v", err)
	}
	problems := ValidateContract(meta)
	if len(problems) != 1 || !strings.Contains(problems[0], `"ToAttributes"`) {
		t.Fatalf("expected exactly one problem about ToAttributes, got: %v", problems)
	}
}

func TestValidateContract_ReportsMissingEntitySet(t *testing.T) {
	changed := strings.Replace(testMetadata, `<EntitySet Name="Subscriptions" EntityType="developer.SubscriptionsType"/>`, "", 1)
	meta, err := ParseServiceMetadata([]byte(changed))
	if err != nil {
		t.Fatalf("ParseServiceMetadata: %v", err)
	}
	problems := ValidateContract(meta)
	if len(problems) != 1 || !strings.Contains(problems[0], `entity set "Subscriptions" is missing`) {
		t.Fatalf("expected exactly one missing-entity-set problem, got: %v", problems)
	}
}

func TestParseServiceMetadata_RejectsMalformedXML(t *testing.T) {
	if _, err := ParseServiceMetadata([]byte("<edmx:Edmx><unclosed>")); err == nil {
		t.Fatal("expected an error for malformed XML")
	}
}

func TestGetServiceMetadata(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/odata/1.0/data.svc/$metadata" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/xml" {
			t.Fatalf("expected Accept: application/xml, got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(testMetadata))
	})
	defer server.Close()

	meta, err := client.GetServiceMetadata(context.Background())
	if err != nil {
		t.Fatalf("GetServiceMetadata: %v", err)
	}
	if _, ok := meta.EntitySets["Subscriptions"]; !ok {
		t.Fatal("expected the Subscriptions entity set to be parsed")
	}
}

// TestContract_CoversEveryWireField derives every JSON field name the client
// actually sends or reads from the wire structs themselves, so adding a field
// to a struct without adding it to Contract (and therefore without it ever
// being checked against a live tenant) fails here.
func TestContract_CoversEveryWireField(t *testing.T) {
	wireTypes := map[string][]any{
		"Applications":  {Application{}},
		"Attributes":    {Attribute{}},
		"Subscriptions": {Subscription{}, subscriptionWriteRequest{}},
	}
	for _, set := range Contract {
		declared := map[string]bool{}
		for _, p := range set.Properties {
			declared[p] = true
		}
		for _, n := range set.NavigationProperties {
			declared[n] = true
		}
		for _, v := range wireTypes[set.Name] {
			for _, field := range jsonFieldNames(v) {
				if !declared[field] {
					t.Errorf("%T sends/reads %q, but Contract for %s does not declare it", v, field, set.Name)
				}
			}
		}
		delete(wireTypes, set.Name)
	}
	for name := range wireTypes {
		t.Errorf("wire types exist for %s, but Contract has no entry for it", name)
	}
}

func jsonFieldNames(v any) []string {
	var names []string
	rt := reflect.TypeOf(v)
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}
		names = append(names, name)
	}
	return names
}
