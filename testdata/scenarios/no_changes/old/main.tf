variable "my_var" {
  type    = string
  default = "value"
}

output "my_output" {
  value = var.my_var
}

resource "null_resource" "example" {
}
