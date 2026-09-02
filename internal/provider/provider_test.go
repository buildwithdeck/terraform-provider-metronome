package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"metronome": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("METRONOME_BEARER_TOKEN") == "" {
		t.Fatal("METRONOME_BEARER_TOKEN must be set for acceptance tests (use a Sandbox tenant token)")
	}
}

// tfAccName returns tf-acc-<prefix>-<8 hex>. Every name, ingest alias and
// uniqueness_key in acceptance tests goes through it: the Sandbox tenant is
// shared and long-lived, and Metronome returns 409 on reuse.
func tfAccName(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("tf-acc-%s-%s", prefix, hex.EncodeToString(b))
}

func TestProviderSchema(t *testing.T) {
	var resp provider.SchemaResponse
	New("test")().Schema(context.Background(), provider.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	token, ok := resp.Schema.Attributes["bearer_token"]
	if !ok || !token.IsSensitive() || !token.IsOptional() {
		t.Fatalf("bearer_token must be optional and sensitive, got %#v", token)
	}
	if base, ok := resp.Schema.Attributes["base_url"]; !ok || !base.IsOptional() {
		t.Fatalf("base_url must be optional, got %#v", base)
	}
}
