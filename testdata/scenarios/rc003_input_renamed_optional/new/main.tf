# Test RC003: optional variable renamed
# New state: optional variable renamed to "timeout_ms"
variable "timeout_ms" {
  type        = string
  description = "Request timeout in milliseconds"
  default     = "30000"
}
