output "host_main" {
  value       = aws_instance.this.public_ip
  sensitive   = true
  description = "The public IP address of the instance"
}

output "entrypoint" {
  value       = "Scan this instance for vulnerabilities ${aws_instance.this.public_ip}"
  sensitive   = true
}
