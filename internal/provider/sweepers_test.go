package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	metronome "github.com/Metronome-Industries/metronome-go/v3"
	"github.com/Metronome-Industries/metronome-go/v3/option"
	"github.com/Metronome-Industries/metronome-go/v3/shared"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Sweepers archive leftovers from interrupted acceptance runs. They match ONLY
// objects whose name starts with tf-acc- (or tf_acc_ for custom field keys) and
// run only via `go test ./internal/provider/ -sweep=all` (see GNUmakefile
// `sweep`), never during a test run. Run them when no acceptance job is active:
// Metronome objects carry no reliable created_at across entity types, so there
// is no age guard.
const sweepPrefix = "tf-acc-"

func init() {
	resource.AddTestSweepers("metronome_custom_field_key", &resource.Sweeper{Name: "metronome_custom_field_key", F: sweepCustomFieldKeys})
	resource.AddTestSweepers("metronome_billable_metric", &resource.Sweeper{Name: "metronome_billable_metric", F: sweepBillableMetrics})
	resource.AddTestSweepers("metronome_product", &resource.Sweeper{Name: "metronome_product", F: sweepProducts})
	resource.AddTestSweepers("metronome_rate_card", &resource.Sweeper{Name: "metronome_rate_card", F: sweepRateCards})
	resource.AddTestSweepers("metronome_package", &resource.Sweeper{Name: "metronome_package", F: sweepPackages})
	resource.AddTestSweepers("metronome_customer", &resource.Sweeper{Name: "metronome_customer", F: sweepCustomers})
	resource.AddTestSweepers("metronome_notification", &resource.Sweeper{Name: "metronome_notification", F: sweepNotifications})
}

func sweepClient() (*metronome.Client, error) {
	token := os.Getenv("METRONOME_BEARER_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("METRONOME_BEARER_TOKEN must be set to sweep")
	}
	opts := []option.RequestOption{option.WithBearerToken(token)}
	if u := os.Getenv("METRONOME_BASE_URL"); u != "" {
		opts = append(opts, option.WithBaseURL(u))
	}
	c := metronome.NewClient(opts...)
	return &c, nil
}

func isSweepable(name string) bool {
	return strings.HasPrefix(name, sweepPrefix) || strings.HasPrefix(name, "tf_acc_")
}

func sweepBillableMetrics(_ string) error {
	client, err := sweepClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	it := client.V1.BillableMetrics.ListAutoPaging(ctx, metronome.V1BillableMetricListParams{})
	for it.Next() {
		m := it.Current()
		if !isSweepable(m.Name) || !m.ArchivedAt.IsZero() {
			continue
		}
		log.Printf("[sweep] archiving billable metric %s (%s)", m.Name, m.ID)
		if _, err := client.V1.BillableMetrics.Archive(ctx, metronome.V1BillableMetricArchiveParams{ID: shared.IDParam{ID: m.ID}}); err != nil {
			log.Printf("[sweep] WARN billable metric %s: %v", m.ID, err)
		}
	}
	return it.Err()
}

func sweepProducts(_ string) error {
	client, err := sweepClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	it := client.V1.Contracts.Products.ListAutoPaging(ctx, metronome.V1ContractProductListParams{})
	for it.Next() {
		p := it.Current()
		if !isSweepable(p.Current.Name) || !p.ArchivedAt.IsZero() {
			continue
		}
		log.Printf("[sweep] archiving product %s (%s)", p.Current.Name, p.ID)
		if _, err := client.V1.Contracts.Products.Archive(ctx, metronome.V1ContractProductArchiveParams{ProductID: p.ID}); err != nil {
			log.Printf("[sweep] WARN product %s: %v", p.ID, err)
		}
	}
	return it.Err()
}

func sweepRateCards(_ string) error {
	client, err := sweepClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	// The list item carries no archived_at; archiving an already-archived card
	// is reported by the API and only logged here.
	it := client.V1.Contracts.RateCards.ListAutoPaging(ctx, metronome.V1ContractRateCardListParams{Body: map[string]any{}})
	for it.Next() {
		rc := it.Current()
		if !isSweepable(rc.Name) {
			continue
		}
		log.Printf("[sweep] archiving rate card %s (%s)", rc.Name, rc.ID)
		if _, err := client.V1.Contracts.RateCards.Archive(ctx, metronome.V1ContractRateCardArchiveParams{ID: shared.IDParam{ID: rc.ID}}); err != nil {
			log.Printf("[sweep] WARN rate card %s: %v", rc.ID, err)
		}
	}
	return it.Err()
}

func sweepPackages(_ string) error {
	client, err := sweepClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	it := client.V1.Packages.ListAutoPaging(ctx, metronome.V1PackageListParams{})
	for it.Next() {
		p := it.Current()
		if !isSweepable(p.Name) || !p.ArchivedAt.IsZero() {
			continue
		}
		log.Printf("[sweep] archiving package %s (%s)", p.Name, p.ID)
		if _, err := client.V1.Packages.Archive(ctx, metronome.V1PackageArchiveParams{PackageID: p.ID}); err != nil {
			log.Printf("[sweep] WARN package %s: %v", p.ID, err)
		}
	}
	return it.Err()
}

// sweepCustomers clears ingest aliases BEFORE archiving: archived customers keep
// their aliases and aliases 409 on reuse, so skipping this leaks test aliases forever.
func sweepCustomers(_ string) error {
	client, err := sweepClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	it := client.V1.Customers.ListAutoPaging(ctx, metronome.V1CustomerListParams{})
	for it.Next() {
		c := it.Current()
		if !isSweepable(c.Name) || !c.ArchivedAt.IsZero() {
			continue
		}
		log.Printf("[sweep] archiving customer %s (%s)", c.Name, c.ID)
		if len(c.IngestAliases) > 0 {
			if err := client.V1.Customers.SetIngestAliases(ctx, metronome.V1CustomerSetIngestAliasesParams{CustomerID: c.ID, IngestAliases: []string{}}); err != nil {
				log.Printf("[sweep] WARN clear aliases %s: %v", c.ID, err)
				continue
			}
		}
		if _, err := client.V1.Customers.Archive(ctx, metronome.V1CustomerArchiveParams{ID: shared.IDParam{ID: c.ID}}); err != nil {
			log.Printf("[sweep] WARN customer %s: %v", c.ID, err)
		}
	}
	return it.Err()
}

func sweepCustomFieldKeys(_ string) error {
	client, err := sweepClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	it := client.V1.CustomFields.ListKeysAutoPaging(ctx, metronome.V1CustomFieldListKeysParams{})
	for it.Next() {
		k := it.Current()
		if !isSweepable(k.Key) {
			continue
		}
		log.Printf("[sweep] removing custom field key %s/%s", k.Entity, k.Key)
		if err := client.V1.CustomFields.RemoveKey(ctx, metronome.V1CustomFieldRemoveKeyParams{Entity: metronome.V1CustomFieldRemoveKeyParamsEntity(k.Entity), Key: k.Key}); err != nil {
			log.Printf("[sweep] WARN custom field key %s/%s: %v", k.Entity, k.Key, err)
		}
	}
	return it.Err()
}

func sweepNotifications(_ string) error {
	client, err := sweepClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	it := client.V2.Notifications.Offset.ListAutoPaging(ctx, metronome.V2NotificationOffsetListParams{})
	for it.Next() {
		n := it.Current()
		if !isSweepable(n.Name) || !n.ArchivedAt.IsZero() {
			continue
		}
		log.Printf("[sweep] archiving notification %s (%s)", n.Name, n.ID)
		if _, err := client.V2.Notifications.Offset.Archive(ctx, metronome.V2NotificationOffsetArchiveParams{ID: n.ID}); err != nil {
			log.Printf("[sweep] WARN notification %s: %v", n.ID, err)
		}
	}
	return it.Err()
}
