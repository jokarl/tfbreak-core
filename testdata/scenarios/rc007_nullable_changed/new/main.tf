# Test RC007: nullable changed from true to false
# Using same default to avoid triggering RC006
variable "optional_config" {
  type        = string
  default     = "default_value"
  nullable    = false
  description = "An optional config that no longer accepts null"
}
