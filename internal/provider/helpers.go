package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	metronome "github.com/Metronome-Industries/metronome-go/v3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// rawPost calls an endpoint the SDK does not wrap (the list lives in
// docs/COVERAGE.md). Declare the request/response structs next to the resource
// that needs them; do not grow a second HTTP client.
func rawPost(ctx context.Context, client *metronome.Client, path string, body any, out any) error {
	return client.Post(ctx, path, body, out)
}

// readGone removes the resource from state when it no longer exists for
// Terraform's purposes: the API returned 404, or the object is archived.
// Metronome archives instead of deleting and archived objects stay readable
// forever, so every Read must call this right after its Get. Returns true when
// the caller should return immediately.
func readGone(ctx context.Context, resp *resource.ReadResponse, err error, archivedAt time.Time) bool {
	if isNotFound(err) || (err == nil && !archivedAt.IsZero()) {
		tflog.Warn(ctx, "Resource is archived or gone upstream, removing from state")
		resp.State.RemoveResource(ctx)
		return true
	}
	return false
}

func isNotFound(err error) bool {
	var apiErr *metronome.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// addAPIError is the one error shape every CRUD method reports.
func addAPIError(diags *diag.Diagnostics, verb, thing string, err error) {
	diags.AddError(fmt.Sprintf("Unable to %s %s", verb, thing), err.Error())
}

// hourFloor truncates t to the hour in UTC. Product updates are scheduled on
// hour boundaries; passing the current hour floor makes an update effective now.
func hourFloor(t time.Time) time.Time {
	return t.UTC().Truncate(time.Hour)
}

// customFieldsDiff reconciles the custom fields on an entity: added or changed
// keys are set in one SetValues call, removed keys go in one DeleteValues call.
// entity is the Metronome entity type ("customer", "product", ...). Values are
// capped at 200 characters by the API; each call is transactional on its side.
func customFieldsDiff(ctx context.Context, client *metronome.Client, entity, entityID string, old, new map[string]string) error {
	set := map[string]string{}
	for k, v := range new {
		if ov, ok := old[k]; !ok || ov != v {
			set[k] = v
		}
	}
	var del []string
	for k := range old {
		if _, ok := new[k]; !ok {
			del = append(del, k)
		}
	}
	sort.Strings(del)

	if len(set) > 0 {
		err := client.V1.CustomFields.SetValues(ctx, metronome.V1CustomFieldSetValuesParams{
			Entity:       metronome.V1CustomFieldSetValuesParamsEntity(entity),
			EntityID:     entityID,
			CustomFields: set,
		})
		if err != nil {
			return fmt.Errorf("set custom fields: %w", err)
		}
	}
	if len(del) > 0 {
		err := client.V1.CustomFields.DeleteValues(ctx, metronome.V1CustomFieldDeleteValuesParams{
			Entity:   metronome.V1CustomFieldDeleteValuesParamsEntity(entity),
			EntityID: entityID,
			Keys:     del,
		})
		if err != nil {
			return fmt.Errorf("delete custom fields: %w", err)
		}
	}
	return nil
}
