package provider

import (
	"context"
	"fmt"
	"strings"

	metronome "github.com/Metronome-Industries/metronome-go/v3"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure interface compliance.
var (
	_ resource.Resource                = &customFieldKeyResource{}
	_ resource.ResourceWithConfigure   = &customFieldKeyResource{}
	_ resource.ResourceWithImportState = &customFieldKeyResource{}
)

func NewCustomFieldKeyResource() resource.Resource {
	return &customFieldKeyResource{}
}

type customFieldKeyResource struct {
	client *metronome.Client
}

// customFieldKeyModel is the Terraform state model for metronome_custom_field_key.
// The composite ID is "<entity>/<key>" — both are immutable, so import can
// reconstruct the full state without any API round-trip.
type customFieldKeyModel struct {
	ID                types.String `tfsdk:"id"`
	Entity            types.String `tfsdk:"entity"`
	Key               types.String `tfsdk:"key"`
	EnforceUniqueness types.Bool   `tfsdk:"enforce_uniqueness"`
}

// validCustomFieldEntities is the exhaustive list of Metronome entity types
// that support custom fields. It mirrors the SDK enum V1CustomFieldAddKeyParamsEntity.
var validCustomFieldEntities = []string{
	"alert", "billable_metric", "charge", "commit", "contract_credit",
	"contract_product", "contract", "credit_grant", "customer_plan", "customer",
	"discount", "invoice", "plan", "professional_service", "product", "rate_card",
	"scheduled_charge", "subscription", "package_commit", "package_credit",
	"package_subscription", "package_scheduled_charge",
}

// stringOneOfValidator is a validator.String that rejects any value not in the
// allowed set. We implement it inline to avoid pulling in a new dependency.
type stringOneOfValidator struct {
	allowed []string
}

func (v stringOneOfValidator) Description(_ context.Context) string {
	return fmt.Sprintf("Value must be one of: %s", strings.Join(v.allowed, ", "))
}

func (v stringOneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v stringOneOfValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	for _, a := range v.allowed {
		if val == a {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid entity type",
		fmt.Sprintf("%q is not a valid Metronome entity type. Must be one of: %s", val, strings.Join(v.allowed, ", ")),
	)
}

// ── resource.Resource ──────────────────────────────────────────────────────────

func (r *customFieldKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_field_key"
}

func (r *customFieldKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registers a custom field key for a Metronome entity type (e.g. customer, product, contract). Custom field keys define the allowed metadata keys that can later be set on individual entity instances with SetValues. All attributes are immutable: any change destroys and recreates the key. Deleting the key hard-removes it from the allowlist and permanently drops all values set for it across every entity instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier in the form \"<entity>/<key>\". Used for import.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"entity": schema.StringAttribute{
				Required:    true,
				Description: "The Metronome entity type this key applies to. Must be one of: alert, billable_metric, charge, commit, contract_credit, contract_product, contract, credit_grant, customer_plan, customer, discount, invoice, plan, professional_service, product, rate_card, scheduled_charge, subscription, package_commit, package_credit, package_subscription, package_scheduled_charge.",
				Validators: []validator.String{
					stringOneOfValidator{allowed: validCustomFieldEntities},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The custom field key name. Once created this value cannot be changed.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enforce_uniqueness": schema.BoolAttribute{
				Required:    true,
				Description: "When true, the API will reject attempts to set a value for this key on a second entity instance that already has this key set to a different value. Set to false to allow multiple entity instances to share the same key without uniqueness enforcement.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// ── resource.ResourceWithConfigure ────────────────────────────────────────────

func (r *customFieldKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected ProviderData, got: %T", req.ProviderData),
		)
		return
	}
	r.client = pd.Client
}

// ── CRUD ───────────────────────────────────────────────────────────────────────

func (r *customFieldKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customFieldKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.V1.CustomFields.AddKey(ctx, metronome.V1CustomFieldAddKeyParams{
		Entity:            metronome.V1CustomFieldAddKeyParamsEntity(plan.Entity.ValueString()),
		Key:               plan.Key.ValueString(),
		EnforceUniqueness: plan.EnforceUniqueness.ValueBool(),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "create", "custom field key", err)
		return
	}

	plan.ID = types.StringValue(plan.Entity.ValueString() + "/" + plan.Key.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customFieldKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customFieldKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := findCustomFieldKey(ctx, r.client, state.Entity.ValueString(), state.Key.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "custom field key", err)
		return
	}

	// Key has been removed upstream — drop from state.
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.EnforceUniqueness = types.BoolValue(found.EnforceUniqueness)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is intentionally an error: every attribute carries RequiresReplace,
// so the framework will always destroy+create instead. We surface an actionable
// error here in case it is somehow reached.
func (r *customFieldKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Custom field keys are immutable",
		"All attributes of metronome_custom_field_key require replacement. Terraform should never call Update on this resource; this is a provider bug.",
	)
}

// Delete calls RemoveKey, which hard-deletes the key and all values set for it.
func (r *customFieldKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customFieldKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.V1.CustomFields.RemoveKey(ctx, metronome.V1CustomFieldRemoveKeyParams{
		Entity: metronome.V1CustomFieldRemoveKeyParamsEntity(state.Entity.ValueString()),
		Key:    state.Key.ValueString(),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "delete", "custom field key", err)
		return
	}
}

// ── resource.ResourceWithImportState ──────────────────────────────────────────

// ImportState parses the import ID "<entity>/<key>" and populates state
// directly — no API call needed because the subsequent Read will refresh the rest.
func (r *customFieldKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected \"<entity>/<key>\", got %q. Example: terraform import metronome_custom_field_key.example customer/x_account_id", req.ID),
		)
		return
	}

	state := customFieldKeyModel{
		ID:     types.StringValue(req.ID),
		Entity: types.StringValue(parts[0]),
		Key:    types.StringValue(parts[1]),
		// EnforceUniqueness will be populated by the subsequent Read.
		EnforceUniqueness: types.BoolNull(),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// findCustomFieldKey pages through ListKeys filtered by entity and returns the
// matching key or nil when not found. It is the sole read path for this resource.
func findCustomFieldKey(ctx context.Context, client *metronome.Client, entity, key string) (*metronome.V1CustomFieldListKeysResponse, error) {
	it := client.V1.CustomFields.ListKeysAutoPaging(ctx, metronome.V1CustomFieldListKeysParams{
		Entities: []string{entity},
	})
	for it.Next() {
		k := it.Current()
		if string(k.Entity) == entity && k.Key == key {
			return &k, nil
		}
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}
