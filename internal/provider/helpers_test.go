package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	metronome "github.com/Metronome-Industries/metronome-go/v3"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestHourFloor(t *testing.T) {
	in := time.Date(2026, 9, 2, 20, 47, 13, 500, time.FixedZone("EDT", -4*3600))
	got := hourFloor(in)
	want := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC) // 20:47 EDT = 00:47 UTC next day
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Fatalf("hourFloor(%v) = %v, want %v UTC", in, got, want)
	}
}

func TestTfAccName(t *testing.T) {
	a, b := tfAccName("bm"), tfAccName("bm")
	if a == b || len(a) != len("tf-acc-bm-")+8 || a[:10] != "tf-acc-bm-" {
		t.Fatalf("unexpected names %q %q", a, b)
	}
}

func TestCustomFieldsDiff(t *testing.T) {
	client, calls := newMockClient(t, jsonHandler(http.StatusOK, `{}`))
	old := map[string]string{"a": "1", "b": "2", "c": "3"}
	new := map[string]string{"a": "1", "b": "9", "d": "4"}
	if err := customFieldsDiff(context.Background(), client, "customer", "cust-1", old, new); err != nil {
		t.Fatal(err)
	}
	got := calls()
	want := []string{"POST /v1/customFields/setValues", "POST /v1/customFields/deleteValues"}
	if !reflect.DeepEqual(routes(got), want) {
		t.Fatalf("routes = %v, want %v", routes(got), want)
	}
	set := bodyJSON(t, got[0])
	if set["entity"] != "customer" || set["entity_id"] != "cust-1" {
		t.Fatalf("setValues body = %v", set)
	}
	if cf, _ := set["custom_fields"].(map[string]any); len(cf) != 2 || cf["b"] != "9" || cf["d"] != "4" {
		t.Fatalf("setValues custom_fields = %v, want only changed/added keys b and d", set["custom_fields"])
	}
	del := bodyJSON(t, got[1])
	if keys, _ := del["keys"].([]any); len(keys) != 1 || keys[0] != "c" {
		t.Fatalf("deleteValues keys = %v, want [c]", del["keys"])
	}
}

func TestCustomFieldsDiff_NoChangeMakesNoCalls(t *testing.T) {
	client, calls := newMockClient(t, nil)
	same := map[string]string{"a": "1"}
	if err := customFieldsDiff(context.Background(), client, "product", "p-1", same, same); err != nil {
		t.Fatal(err)
	}
	if n := len(calls()); n != 0 {
		t.Fatalf("expected no API calls, got %d: %v", n, routes(calls()))
	}
}

func newReadResponse(t *testing.T) *resource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	s := schema.Schema{Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}}}
	raw := tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "abc"),
	})
	return &resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: raw}}
}

func TestReadGone(t *testing.T) {
	notFound := &metronome.Error{StatusCode: http.StatusNotFound}
	cases := []struct {
		name       string
		err        error
		archivedAt time.Time
		wantGone   bool
	}{
		{"live object", nil, time.Time{}, false},
		{"archived object", nil, time.Now(), true},
		{"404", notFound, time.Time{}, true},
		{"wrapped 404", fmt.Errorf("read: %w", notFound), time.Time{}, true},
		{"other API error", &metronome.Error{StatusCode: http.StatusInternalServerError}, time.Time{}, false},
		{"transport error", errors.New("dial tcp: refused"), time.Time{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := newReadResponse(t)
			gone := readGone(context.Background(), resp, tc.err, tc.archivedAt)
			if gone != tc.wantGone {
				t.Fatalf("readGone = %v, want %v", gone, tc.wantGone)
			}
			if resp.State.Raw.IsNull() != tc.wantGone {
				t.Fatalf("state removed = %v, want %v", resp.State.Raw.IsNull(), tc.wantGone)
			}
		})
	}
}
