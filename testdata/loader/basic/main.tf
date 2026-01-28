resource "null_resource" "example" {
}

resource "local_file" "config" {
  filename = "/tmp/config.txt"
  content  = "hello"
}

module "submodule" {
  source  = "./submodule"
  version = "1.0.0"
}
