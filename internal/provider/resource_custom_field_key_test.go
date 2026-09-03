package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"testing"

	metronome "github.com/Metronome-Industries/metronome-go/v3"
	"github.com/Metronome-Industries/metronome-go/v3/option"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ── validator unit tests ───────────────────────────────────────────────────────

func TestStringOneOfValidator_valid(t *testing.T) {
	v := stringOneOfValidator{allowed: validCustomFieldEntities}
	for _, entity := range validCustomFieldEntities {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue(entity),
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("entity %q unexpectedly failed validation: %v", entity, resp.Diagnostics)
		}
	}
}

func TestStringOneOfValidator_invalid(t *testing.T) {
	v := stringOneOfValidator{allowed: validCustomFieldEntities}
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringValue("foobar"),
	}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error for invalid entity type, got none")
	}
}

// ── httptest unit test ─────────────────────────────────────────────────────────

// TestCustomFieldKeyRead_OnlyHitsListKeys asserts that Read only calls
// POST /v1/customFields/listKeys and never touches a write endpoint.
func TestCustomFieldKeyRead_OnlyHitsListKeys(t *testing.T) {
	fixture := `{
		"data": [
			{"entity": "customer", "key": "x_account_id", "enforce_uniqueness": true}
		],
		"next_page": ""
	}`
	client, calls := newMockClient(t, jsonHandler(http.StatusOK, fixture))

	r := &customFieldKeyResource{client: client}

	s := customFieldKeySchemaForTest(t)
	ctx := context.Background()
	raw := tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "customer/x_account_id"),
		"entity":             tftypes.NewValue(tftypes.String, "customer"),
		"key":                tftypes.NewValue(tftypes.String, "x_account_id"),
		"enforce_uniqueness": tftypes.NewValue(tftypes.Bool, false),
	})
	stateIn := tfsdk.State{Schema: s, Raw: raw}
	readResp := &frameworkresource.ReadResponse{State: stateIn}

	r.Read(ctx, frameworkresource.ReadRequest{State: stateIn}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", readResp.Diagnostics)
	}

	got := routes(calls())
	want := []string{"POST /v1/customFields/listKeys"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Read routes = %v, want %v", got, want)
	}
}

// customFieldKeySchemaForTest returns the resource schema for unit-test state construction.
func customFieldKeySchemaForTest(t *testing.T) schema.Schema {
	t.Helper()
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"entity": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enforce_uniqueness": schema.BoolAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// ── acceptance tests ───────────────────────────────────────────────────────────

func TestAccCustomFieldKey_basic(t *testing.T) {
	t.Parallel()
	entity := "customer"
	key := tfAccName("cfk")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCustomFieldKeyDestroyed(entity, key),
		Steps: []resource.TestStep{
			// Create and verify.
			{
				Config: testAccCustomFieldKeyConfig(entity, key, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("metronome_custom_field_key.test", "entity", entity),
					resource.TestCheckResourceAttr("metronome_custom_field_key.test", "key", key),
					resource.TestCheckResourceAttr("metronome_custom_field_key.test", "enforce_uniqueness", "false"),
					resource.TestCheckResourceAttr("metronome_custom_field_key.test", "id", entity+"/"+key),
				),
			},
			// Import and verify.
			{
				ResourceName:      "metronome_custom_field_key.test",
				ImportState:       true,
				ImportStateId:     entity + "/" + key,
				ImportStateVerify: true,
			},
			// Changing enforce_uniqueness triggers replace (all attrs are ForceNew).
			{
				Config: testAccCustomFieldKeyConfig(entity, key, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"metronome_custom_field_key.test",
							plancheck.ResourceActionDestroyBeforeCreate,
						),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("metronome_custom_field_key.test", "enforce_uniqueness", "true"),
				),
			},
		},
	})
}

// ── helpers ────────────────────────────────────────────────────────────────────

func testAccCustomFieldKeyConfig(entity, key string, enforceUniqueness bool) string {
	return fmt.Sprintf(`
resource "metronome_custom_field_key" "test" {
  entity             = %q
  key                = %q
  enforce_uniqueness = %t
}
`, entity, key, enforceUniqueness)
}

func testAccCheckCustomFieldKeyDestroyed(entity, key string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		token := os.Getenv("METRONOME_BEARER_TOKEN")
		c := metronome.NewClient(option.WithBearerToken(token))
		found, err := findCustomFieldKey(context.Background(), &c, entity, key)
		if err != nil {
			return fmt.Errorf("CheckDestroy: list custom field keys: %w", err)
		}
		if found != nil {
			return fmt.Errorf("CheckDestroy: custom field key %s/%s still exists", entity, key)
		}
		return nil
	}
}
