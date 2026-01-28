resource "null_resource" "new_name" {
}

moved {
  from = null_resource.old_name
  to   = null_resource.new_name
}
