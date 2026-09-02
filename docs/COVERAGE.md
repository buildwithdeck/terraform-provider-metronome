# Coverage: what this provider manages, and what the Metronome API allows

Source of truth for scope. Hand-maintained; flip a row's status in the PR that ships it. The full gap assessment, with the API constraints behind every row, is Linear issue V2-3634 (Deck V2, project *Metronome Terraform Provider*).

Facts that shape every row: the Go SDK (`metronome-go/v3`) covers 123 of the 159 public endpoints, and the 36 missing ones are reachable through `client.Post` (8 are deliberately excluded by Metronome). Almost nothing hard-deletes: `terraform destroy` archives, irreversibly, and archived objects stay readable with `archived_at` set. Alerts, custom field keys and packages are immutable; billable metrics are name-only mutable; rates are an append-only schedule; product updates are scheduled on hour boundaries; contracts are edited through a tenant-gated delta API.

Legend: SDK = method on `client.V1.` / `client.V2.`. RAW = not in the SDK, called via `rawPost`. Delete = what `terraform destroy` does. FN = ForceNew attributes. Status: ⬜ planned · 🔧 in progress · ✅ shipped.

## Phase 0: Foundation

| # | Item | Linear | Status |
| -- | -- | -- | -- |
| 0a | Repo scaffold from the Clerk skeleton | V2-3635 | ✅ |
| 0b | Sandbox tenant API token, `METRONOME_BEARER_TOKEN` secret, Stripe test connection | V2-3636 | ✅ |
| 0c | Shared plumbing: sweepers, `hourFloor`, `customFieldsDiff`, read-only httptest harness | V2-3637 | ✅ |
| 0d | API spike resolving open behaviour questions (see `API-NOTES.md`) | V2-3660 | ✅ |
| 0e | Raise the Sandbox active-customer cap (trial limit 5) — blocks customer, contract, customer billing config, customer-scoped alerts | V2-3672 | ⬜ |

## Phase 1: Catalog primitives

| Resource | Create | Read | Update | Delete | FN | Linear | Status |
| -- | -- | -- | -- | -- | -- | -- | -- |
| `metronome_custom_field_key` | `CustomFields.AddKey` | `CustomFields.ListKeys` filtered | none | `CustomFields.RemoveKey` (hard, drops values) | all | V2-3638 | ⬜ |
| `metronome_notification` | `V2.Notifications.Offset.New` | `.Get` | `.Edit` | `.Archive` | none | V2-3639 | ⬜ |
| `metronome_billable_metric` | `BillableMetrics.New` | `.Get` | `.Update` (name only) | `.Archive` | all but `name` | V2-3640 | ⬜ |
| `metronome_alert` | `Alerts.New` (+`uniqueness_key`) | `Customers.Alerts.Get` when `customer_id` set, else state | none | `Alerts.Archive` (`release_uniqueness_key=true`) | all | V2-3641 | ⬜ |

## Phase 2: Pricing

| Resource | Create | Read | Update | Delete | FN | Linear | Status |
| -- | -- | -- | -- | -- | -- | -- | -- |
| data `metronome_credit_types` | | `PricingUnits.List` | | | | V2-3642 | ⬜ |
| `metronome_product` | `Contracts.Products.New` | `.Get` (`current` view) | `.Update` (`starting_at` = hour floor) | `.Archive` | `type` | V2-3661 | ⬜ |
| `metronome_rate_card` | `Contracts.RateCards.New` | `.Get` | `.Update` (metadata, cosmetic) | `.Archive` | `fiat_credit_type_id` | V2-3662 | ⬜ |
| `metronome_rate` | `Contracts.RateCards.Rates.Add` | `.Rates.List` at `starting_at`, filtered | re-`Add` or FN (spike) | state-only, warns | `rate_card_id`, `product_id`, `pricing_group_values`, `starting_at` | V2-3664 | ⬜ |

## Phase 3: Packaging

| Resource | Create | Read | Update | Delete | FN | Linear | Status |
| -- | -- | -- | -- | -- | -- | -- | -- |
| data `metronome_billing_providers` | | `Settings.BillingProviders.List` | | | | V2-3643 | ⬜ |
| `metronome_package` | `Packages.New` | `.Get` | none | `.Archive` | all | V2-3665 | ⬜ |

## Phase 4: Customers and contracts

| Resource | Create | Read | Update | Delete | FN | Linear | Status |
| -- | -- | -- | -- | -- | -- | -- | -- |
| `metronome_customer` | `Customers.New` | `.Get` | `.SetName`, `.SetIngestAliases`, `.UpdateConfig`, custom fields | `.SetIngestAliases([])` then `.Archive` | none | V2-3644 | ⬜ |
| `metronome_customer_billing_provider_configuration` | `Customers.SetBillingConfigurations` | `.GetBillingConfigurations` | none | `.ArchiveBillingConfigurations` | all | V2-3663 | ⬜ |
| `metronome_contract` | `Contracts.New` (+`uniqueness_key`) | `V2.Contracts.Get` | `UpdateEndDate`, `SetUsageFilter`, custom fields; else `ModifyPlan` error | `.Archive` | none (errors) | V2-3666 | ⬜ |
| data `metronome_customer` | | `Customers.Get` by id or `.List` by ingest alias | | | | V2-3645 | ⬜ |
| data `metronome_billable_metric` | | `BillableMetrics.List` + name filter | | | | V2-3646 | ⬜ |
| data `metronome_product` | | `Contracts.Products.List` + name filter | | | | V2-3647 | ⬜ |
| data `metronome_rate_card` | | `Contracts.RateCards.List` + name/alias filter | | | | V2-3648 | ⬜ |

## Phase 5: Account-level configuration

| Resource | Create | Read | Update | Delete | FN | Linear | Status |
| -- | -- | -- | -- | -- | -- | -- | -- |
| `metronome_billing_provider` | `Settings.BillingProviders.New` | `.List` filtered by `delivery_method_id` | none | state-only (no API) | all | V2-3649 | ⬜ |
| `metronome_stripe_billing_settings` | RAW `POST /v1/client/billing-config/stripe` | write-only, trust state | RAW `PATCH` | RAW `DELETE` (hard) | none | V2-3650 | ⬜ |
| `metronome_webhook_secret` | RAW `POST /v1/client/config/webhook_secret` | state | re-set | no-op | none | V2-3651 | ⬜ |
| data `metronome_services` | | `Services.List` | | | | V2-3652 | ⬜ |

## Backlog (each issue names its gate)

| Resource | Parked because | Linear |
| -- | -- | -- |
| `metronome_named_schedule` (customer / contract / rate card) | Tenant-gated; SDK service names inverted; `listNamedSchedules` is `x-stainless-skip` | V2-3653 |
| `metronome_rate_card_product_order` | Read path unconfirmed (spike) | V2-3654 |
| `metronome_customer_commit` | Spec steers to `contracts/edit`; no get-by-id; archive is an SDK gap | V2-3655 |
| `metronome_customer_credit` | same | V2-3656 |
| `metronome_customer_revenue_system_configuration` | Entire entity is an SDK gap; niche | V2-3657 |
| `metronome_tax_provider_credentials` (Avalara + Anrok) | Upsert-only write-only secrets; PLG / threshold-billing only | V2-3658 |
| data `metronome_notification_system_events` | Enumeration only, low value | V2-3659 |

## Not Terraform material

| Surface | Reason |
| -- | -- |
| Usage: `ingest`, `usage`, `usage/groups`, `events/search`, seats | Runtime data; events are immutable |
| Invoices: list / get / pdf / breakdowns / void / regenerate / addCharge / historical / pro-services / issue date | Generated artefacts; void and regenerate are actions |
| Balances, ledger, net balance, manual ledger entry | Runtime state; ledger is append-only |
| Dashboards `getEmbeddableUrl` | Time-limited action |
| Audit logs | Read-only stream |
| Approval requests | `x-stainless-skip`, feature-flagged |
| Plans 1.0: plans, customer plans, credit grants, recharge settings, `migrateToContracts`, customer `costs` | Deprecated by Metronome; plans are UI-only |
| `composite/createCustomerWithContract`, `customers/setBillableStatus`, `threshold-billing/*`, `commits/retire`, `disableTrueup`, `integrations/log`, `rotateDeltaStreamSecret` | Tenant-gated, Stainless-skipped, or operational actions |
| API tokens, webhook endpoint URL, custom pricing units, contract-editing enablement | UI-only, no API |

## SDK gaps this provider calls via `rawPost`

`POST/PATCH/DELETE /v1/client/billing-config/stripe` · `POST /v1/client/config/webhook_secret` · `POST /v2/contracts/commits/archive` · `POST /v2/contracts/credits/archive` · `POST /v1/{set,get,archive}CustomerRevenueSystemConfigurations` · `POST /v1/upsertAnrokApiToken`
