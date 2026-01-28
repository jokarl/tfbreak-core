resource "aws_s3_bucket" "new_name" {
  bucket = "my-bucket"
}

moved {
  from = aws_s3_bucket.old_name
  to   = aws_s3_bucket.new_name
}

moved {
  from = module.old_module
  to   = module.new_module
}
