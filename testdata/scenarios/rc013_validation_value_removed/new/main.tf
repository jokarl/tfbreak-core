# Test RC013: validation value removed from contains()
# New state: "prod" removed from contains() list
variable "environment" {
  type        = string
  description = "Deployment environment"

  validation {
    condition     = contains(["dev", "staging"], var.environment)
    error_message = "Environment must be dev or staging."
  }
}
