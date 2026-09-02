package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
