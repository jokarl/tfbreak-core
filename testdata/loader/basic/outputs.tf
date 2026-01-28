output "simple_output" {
  value       = "hello"
  description = "A simple output"
}

output "sensitive_output" {
  value     = "secret"
  sensitive = true
}
