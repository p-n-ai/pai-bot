output "instance_id" {
  description = "EC2 instance ID"
  value       = aws_instance.app.id
}

output "public_ip" {
  description = "Public IP of the EC2 instance"
  value       = aws_instance.app.public_ip
}

output "ssh_command" {
  description = "SSH command to connect"
  value       = "ssh -i terraform/${var.project}-key.pem ubuntu@${aws_instance.app.public_ip}"
}

output "health_check_url" {
  description = "Health check URL (via Caddy)"
  value       = "http://${aws_instance.app.public_ip}/healthz"
}
