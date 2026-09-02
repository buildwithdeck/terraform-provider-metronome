package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestMain delegates to the plugin-testing harness so sweepers registered with
// resource.AddTestSweepers run via `go test -sweep=all`. Metronome has no API to
// create tenants, so unlike the Clerk provider there is no ephemeral-app
// lifecycle here: tests share the Sandbox tenant and clean up by archiving.
func TestMain(m *testing.M) {
	resource.TestMain(m)
}
