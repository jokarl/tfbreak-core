# Test BC004: type changed from string to number
# Using no default to avoid triggering RC006
variable "instance_count" {
  type        = number
  description = "Number of instances"
}
