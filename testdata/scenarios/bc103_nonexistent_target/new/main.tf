terraform {
  required_providers {
    null = {
      source = "hashicorp/null"
    }
  }
}

# Resource removed, moved block points to non-existent target

moved {
  from = null_resource.old_name
  to   = null_resource.new_name
}
# Note: null_resource.new_name does not exist
