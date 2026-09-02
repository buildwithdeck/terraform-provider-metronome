# Terraform Provider for Metronome

Manage [Metronome](https://metronome.com) usage-based billing configuration as code: billable metrics, products, rate cards and rates, packages, alerts, notifications, customers and contracts.

Built with the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) on top of the official [Metronome Go SDK](https://github.com/Metronome-Industries/metronome-go). Sibling of [terraform-provider-clerk](https://github.com/buildwithdeck/terraform-provider-clerk) and follows the same conventions.

> **Status: scaffold.** No resources yet. [docs/COVERAGE.md](docs/COVERAGE.md) lists what is planned and what the Metronome API allows.

## Usage

```hcl
terraform {
  required_providers {
    metronome = {
      source = "buildwithdeck/metronome"
    }
  }
}

provider "metronome" {
  # bearer_token = "..."   # or set METRONOME_BEARER_TOKEN
}
```

Create the token in the Metronome app under **Developer → API tokens**. Metronome exposes one API host; production and Sandbox tenants are distinguished by the token you use. `base_url` / `METRONOME_BASE_URL` override the host, which is only needed for tests against a mock server.

## What Terraform can and cannot do with Metronome

Three facts shape every resource in this provider:

- **Nothing hard-deletes.** `terraform destroy` archives. Archiving is irreversible and the object stays readable with `archived_at` set. The provider drops archived objects from state on the next refresh.
- **Several objects are immutable or append-only.** Alerts, custom field keys and packages cannot be updated (every change is a replacement). Billable metrics only allow a rename. Rates are an append-only schedule. Product changes are scheduled on hour boundaries.
- **Contracts are edited through a tenant-gated delta API.** The provider creates contracts and supports narrow updates (end date, usage filter, custom fields). Other diffs are rejected at plan time instead of replacing the contract, because replacement voids invoices.

Per-resource details: [docs/COVERAGE.md](docs/COVERAGE.md).

## Development

Requirements: Go (version in `go.mod`). Terraform CLI is needed for acceptance tests; `make generate` downloads one automatically if none is on `PATH`.

```sh
make build      # go build ./...
make test       # unit tests, no credentials needed
make generate   # regenerate docs/ from the provider schema and examples/
make testacc    # acceptance tests, needs METRONOME_BEARER_TOKEN
```

### Acceptance tests

Acceptance tests run against Deck's Metronome **Sandbox** tenant. Never point `TF_ACC` at a production token: tests archive what they create, and archiving cannot be undone. Every test object is named `tf-acc-<prefix>-<random>`, and `go test ./internal/provider/ -sweep=all` archives leftovers older than an hour.

```sh
TF_ACC=1 METRONOME_BEARER_TOKEN=<sandbox token> go test ./internal/provider/ -v -timeout 600s
```

CI builds on every PR and runs the acceptance suite when the `METRONOME_BEARER_TOKEN` repository secret is present. Fork PRs run acceptance only after a maintainer applies the `safe-to-test` label.

## Contributing

One resource per PR, following the checklist in [CLAUDE.md](CLAUDE.md). Work is tracked in Linear under the *Metronome Terraform Provider* project.

## License

[MPL-2.0](LICENSE)
