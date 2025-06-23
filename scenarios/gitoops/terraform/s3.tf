resource "random_string" "random" {
  length  = 16
  special = false
  upper   = false
}


resource "aws_s3_bucket" "this" {
  bucket = "gitoops-${random_string.random.result}"
}


resource "aws_s3_bucket_public_access_block" "this" {
  bucket = aws_s3_bucket.this.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = true
  restrict_public_buckets = false
}


resource "aws_s3_bucket_policy" "this" {
  depends_on = [
    aws_s3_bucket_public_access_block.this
  ]

  bucket = aws_s3_bucket.this.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = "*"
        Action = [
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.this.arn,
          "${aws_s3_bucket.this.arn}/*"
        ]
      }
    ]
  })
}


resource "aws_s3_object" "state" {
  key    = "/gitcorp/terraform.tfstate"
  bucket = aws_s3_bucket.this.id
  content = templatefile("src/tfstate.tmpl", {
    ACCOUNT_ID = data.aws_caller_identity.current.account_id
    AWS_REGION = data.aws_region.current.name
  })
}
