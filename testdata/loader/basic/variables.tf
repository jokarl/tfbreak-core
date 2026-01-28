variable "required_var" {
  type        = string
  description = "A required variable"
}

variable "optional_var" {
  type        = string
  default     = "default_value"
  description = "An optional variable"
}

variable "sensitive_var" {
  type      = string
  default   = "secret"
  sensitive = true
}
