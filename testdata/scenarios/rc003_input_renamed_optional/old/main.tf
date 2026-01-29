# Test RC003: optional variable renamed
# Old state: optional variable "timeout"
variable "timeout" {
  type        = string
  description = "Request timeout"
  default     = "30s"
}
