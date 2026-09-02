# Metronome API behaviour notes (Sandbox spike, 2026-09-02)

Observed against https://api.metronome.com with a Sandbox tenant. Status codes and
bodies are as returned. Spec = https://docs.metronome.com/openapi.json.

## Sandbox constraints
- `POST /v1/customers` → 404 `{"code":"NotFound","message":"Trial accounts are limited to 5 active
  customers. To add more customers, contact Metronome's Sales team."}`. The Sandbox holds 5 active
  and 9 archived customers, so archived customers do NOT count. Tracked in Linear V2-3672; until
  Metronome raises the cap, acceptance tests that create a customer cannot run on this tenant.
- `GET /v1/customers?only_archived=true` lists archived customers. Archive body is `{id}`; clear
  aliases first with `POST /v1/customers/{id}/setIngestAliases {"ingest_aliases":[]}`.
- Archiving is the only delete; archived objects stay readable.

## Rates (`metronome_rate`)
- `POST /v1/contract-pricing/rate-cards/addRate` required: `rate_card_id, product_id,
  starting_at, entitled, rate_type`. Response: `{rate_type, price, credit_type}` — no id.
- Re-adding for the same (rate_card_id, product_id, starting_at) → 200 and REPLACES;
  `getRates`/`getRateSchedule` return one entry with the new price.
- Resource identity must be composite; update = re-add; removal = set `ending_before`.

## Products (`metronome_product`)
- `products/update` requires `starting_at` on an hour boundary; otherwise 400
  `Product update effective at is required to be on an hour boundary`.
- Past `starting_at` (hour floor, or hours earlier) → 200 and visible in `current` at once.
- `products/get` → `{id, type, initial, current, updates[], custom_fields, archived_at}`.
  `current` = most recently CREATED update with `starting_at <= now` (an older-effective
  update created later wins). Future updates do not affect `current`.
  `updates[]` is in `created_at` order and carries only the changed fields.

## Rate card product order (`metronome_rate_card_product_order`)
- Body: `{rate_card_id, product_order: [ids]}` (not `product_ids`). Partial lists accepted.
- Response `{data:{id: rate_card_id}}`. No read path: `rate-cards/get` returns
  `aliases, created_at, created_by, custom_fields, fiat_credit_type, id, name(, archived_at)`;
  `getRateSchedule` item order does not follow `product_order`. Treat as write-only state.

## Commit / credit listing (V2-3655 / V2-3656)
- `contracts/customerCommits/list` filters: `customer_id` (required), `commit_id`,
  `access_type`, `covering_date`, `starting_at`, `effective_before`,
  `include_contract_commits`, `include_archived`, `include_ledgers`, `include_balance`,
  `limit`, `next_page`. Credits: `credit_id`, `include_contract_credits`.
- Unknown fields (`id`, `contract_id`) are silently ignored. Malformed `commit_id` → 400.
- Contract-level commits require `include_contract_commits: true`.

## Contract edit (`metronome_contract`)
- `POST /v2/contracts/edit` required `customer_id, contract_id`; responses 200/400 in spec;
  200 body `{data:{id, edit}}`. Unknown customer → 404 `{"code":"NotFound"}` before any
  feature check, so enablement is unknown until tested on a real contract.
- `POST /v2/contracts/get` unknown customer → 400 `{"code":"CustomerNotFound"}`.
- Contract object keys: archived_at, billing_provider_configuration_schedule, commits,
  created_at, created_by, credits, custom_fields, customer_billing_provider_configuration,
  customer_id, discounts, ending_before, has_more, hierarchy_configuration, id,
  multiplier_override_prioritization, name, net_payment_terms_days, netsuite_sales_order_id,
  overrides, package_id, prepaid_balance_threshold_configuration, priority,
  professional_services, rate_card_id, recurring_commits, recurring_credits,
  reseller_royalties, revenue_system_configuration_schedule, salesforce_opportunity_id,
  scheduled_charges, scheduled_charges_on_usage_invoices, spend_threshold_configuration,
  spend_trackers, starting_at, subscriptions, total_contract_value, transitions,
  uniqueness_key, usage_filter, usage_statement_schedule.

## Packages (`metronome_package`)
- `packages/create` required: `name`. Only on packages: `aliases, billing_provider,
  contract_name, delivery_method, duration`. Only on `contracts/create`:
  `billing_provider_configuration, custom_fields, customer_id, discounts, ending_before,
  hierarchy_configuration, netsuite_sales_order_id, package_alias, package_id,
  professional_services, reseller_royalties, revenue_system_configuration,
  salesforce_opportunity_id, starting_at, total_contract_value, transition, usage_filter`.
- Response `{data:{id}}`. Duplicate `uniqueness_key` → 409 `This uniqueness key has already been used`.
- `aliases: [{name}]` works; alias `starting_at` / `ending_before` are optional.
- `packages/get` top keys: aliases, commits, created_at, created_by, credits, id,
  multiplier_override_prioritization, name, overrides, rate_card_id, recurring_commits,
  recurring_credits, scheduled_charges, subscriptions, uniqueness_key,
  usage_statement_schedule (+ archived_at once archived; contract_name in list items).
- `uniqueness_key` is read back prefixed: send `x`, read `package:x`. Normalize in Read.
- `credits[].access_schedule.schedule_items[]` require `amount, starting_at_offset, duration`
  (RelativeDate `{value, unit: DAYS|WEEKS|MONTHS|YEARS}`). Read-back returns RelativeDate
  values as strings and `product: {id, name}` instead of `product_id`.
- `recurring_credits` with every spec-required field (`product_id, access_amount, priority,
  commit_duration, starting_at_offset`), with and without optionals, is rejected 400
  `instance failed to match all required schemas (matched only 1 out of 2)`. Server/spec drift;
  open question for Metronome before the recurring-credit block ships.
- `packages/archive` 200; again → 422 `already archived`. `packages/list` defaults to
  NOT_ARCHIVED (`archive_filter: ARCHIVED|ALL`), page size 10, `limit`/`next_page` are
  QUERY parameters.

## Custom field keys (`metronome_custom_field_key`)
- `customFields/addKey` `{entity, key, enforce_uniqueness}` → 200 `{data:{id}}`; the id
  is never returned by any read. Duplicate → 400 `Duplicate custom field key`.
- `listKeys` `{entities:[...]}` → items `{entity, key, enforce_uniqueness}` only.
- `removeKey` → 200 with a `null` body; key disappears (no archived marker); again → 404
  `Custom field key not found`.
- Provider id = `<entity>/<key>`; existence check = presence in listKeys.

## Customer billing provider configuration (`metronome_customer_billing_provider_configuration`)
- `setCustomerBillingProviderConfigurations` body `{data:[{customer_id, billing_provider,
  delivery_method_id | delivery_method, configuration, tax_provider,
  unbillable_invoices_configuration}]}`. Stripe config keys: `stripe_customer_id`,
  `stripe_collection_method`.
- Unknown customer → 500 `Unexpected internal error` (not 404).
- `getCustomerBillingProviderConfigurations` → 200 `{data:[]}` for an unknown customer.
  Item fields: id, billing_provider, customer_id, configuration, delivery_method_id,
  delivery_method, delivery_method_configuration, archived_at, unbillable_invoices_configuration.
- `archiveCustomerBillingProviderConfigurations` body: `{customer_id,
  customer_billing_provider_configuration_ids}`.
- `stripe_customer_id` validation: NOT VERIFIED (no customer slot).

## Alerts (`metronome_alert`)
- `alerts/create` required `alert_type, name, threshold`; response `{data:{id}}` only.
  Duplicate `uniqueness_key` → 409 `{message, conflicting_id, conflicting_value}`.
- `alert_type` enum: spend_threshold_reached, monthly_invoice_total_spend_threshold_reached,
  usage_threshold_reached, low_remaining_days_for_commit_segment_reached,
  low_remaining_commit_balance_reached, low_remaining_commit_percentage_reached,
  low_remaining_days_for_contract_credit_segment_reached,
  low_remaining_contract_credit_balance_reached,
  low_remaining_contract_credit_percentage_reached,
  low_remaining_contract_credit_and_commit_balance_reached, invoice_total_reached,
  low_remaining_seat_balance_reached.
- Read path is `customer-alerts/get {customer_id, alert_id}` → `{alert, customer_status,
  triggered_by}`; alert keys: credit_type, id, name, status, threshold, type,
  uniqueness_key, updated_at. Works for GLOBAL alerts with any existing customer id
  (`alert.customer_id` is absent). Unknown customer → 404.
- `alerts/archive {id, release_uniqueness_key}` → 200; alert stays readable with
  `status: archived`. Without `release_uniqueness_key: true` the key is held forever
  (re-archive → 400 `Alert already archived`). Destroy MUST pass it.

## Account-level billing provider (`metronome_billing_provider`) — spec only
- `POST /v1/setUpBillingProvider`: "Set up account-level configuration for a billing
  provider. Once configured, individual contracts across customers can be mapped to this
  configuration using the returned delivery_method_id."
- `billing_provider`: aws_marketplace | azure_marketplace | gcp_marketplace.
  `delivery_method`: direct_to_billing_provider | aws_sqs | aws_sns. `configuration`
  is provider-specific (aws_external_id/aws_iam_role_arn; azure_client_id/
  raw_azure_client_secret/azure_tenant_id; gcp_provider_id/
  raw_gcp_workload_identity_federation_config).
- Responses: 200 `{data:{delivery_method_id}}`, 400, 409 "Conflict error" `{message}`.
  No idempotency or update semantics documented; assume 409 on re-create.
- Stripe is configured via `POST/PATCH/DELETE /v1/client/billing-config/stripe`
  (`stripe_api_key` required, `anrok_api_key` optional, empty 200 body).
- `POST /v1/listConfiguredBillingProviders {}` items: billing_provider, delivery_method_id,
  delivery_method, delivery_method_configuration (secrets omitted).

## Blocked on customer slot (re-run once V2-3672 is resolved)
Create `tf-acc-*` customer + contract first (`POST /v1/customers`, `POST /v1/contracts/create` with
one prepaid commit), then:
1. **Q4** `POST /v1/contracts/customerCommits/list {customer_id, commit_id, include_contract_commits:true}`
   and `customerCredits/list {customer_id, credit_id, include_contract_credits:true}`: confirm a single
   item comes back and record its fields. (V2-3655, V2-3656)
2. **Q5** `POST /v2/contracts/edit {customer_id, contract_id, add_commits: []}`: 200 means contract
   editing is enabled on the tenant; a 4xx with a feature message means it is not. Also call
   `POST /v2/contracts/getEditHistory`. (V2-3666)
3. **Q8** `POST /v1/setCustomerBillingProviderConfigurations {data:[{customer_id, billing_provider:
   "stripe", delivery_method_id: <Sandbox Stripe id>, configuration:{stripe_customer_id:
   "cus_tfacc_doesnotexist", stripe_collection_method:"charge_automatically"}}]}`: accepted (no
   validation) or rejected. Then `getCustomerBillingProviderConfigurations` and
   `archiveCustomerBillingProviderConfigurations`. (V2-3663)
4. **Q9** customer-scoped `POST /v1/alerts/create {customer_id, alert_type:"spend_threshold_reached",
   threshold, name, uniqueness_key}` then `customer-alerts/get`; archive with
   `release_uniqueness_key:true`. (V2-3641)
5. Cleanup: `setIngestAliases {ingest_aliases:[]}` then `POST /v1/customers/archive`; contract
   archive with `void_invoices:true`.
