output "entrypoint" {
  value       = "Entrypoint ${aws_instance.this.public_ip}"
  sensitive   = true
  description = "Entrypoint for the scenario"
}

output "host_main" {
  value       = aws_instance.this.public_ip
  sensitive   = true
  description = "The public IP address of the instance"
}
