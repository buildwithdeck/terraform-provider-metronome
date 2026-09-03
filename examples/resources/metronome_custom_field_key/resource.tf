# Register a custom field key that can be set on any Customer entity.
# Once created, all attributes are immutable — any change will destroy and
# recreate the key, which permanently drops all values already set for it.
resource "metronome_custom_field_key" "customer_account_id" {
  entity             = "customer"
  key                = "x_account_id"
  enforce_uniqueness = false
}

# A product-scoped key with uniqueness enforced.
resource "metronome_custom_field_key" "product_sku" {
  entity             = "product"
  key                = "sku"
  enforce_uniqueness = true
}
