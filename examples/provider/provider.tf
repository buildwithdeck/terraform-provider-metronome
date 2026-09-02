# Authenticate with a Metronome API token (Metronome app → Developer → API tokens).
# Point the provider at Deck's Sandbox tenant by using a token minted there.

provider "metronome" {
  # bearer_token = "..."                        # or set METRONOME_BEARER_TOKEN
  # base_url     = "https://api.metronome.com"  # optional; or set METRONOME_BASE_URL
}
