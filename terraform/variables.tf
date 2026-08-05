variable "project" {
  description = "Project name used for tagging and naming"
  type        = string
  default     = "pai-bot"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "ap-southeast-5"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.small"
}

variable "volume_size_gb" {
  description = "Root EBS volume size in GB"
  type        = number
  default     = 30
}

variable "ssh_cidr_blocks" {
  description = "CIDR blocks allowed to SSH — must be explicitly set, no default"
  type        = list(string)
}

variable "ssh_public_key" {
  description = "Externally generated OpenSSH public key used for server access"
  type        = string

  validation {
    condition     = can(regex("^(ssh-ed25519|ssh-rsa|ecdsa-sha2-nistp(256|384|521)) ", trimspace(var.ssh_public_key)))
    error_message = "ssh_public_key must be a valid OpenSSH public key."
  }
}

variable "ssh_private_key_path" {
  description = "Local private key path used in the SSH command output"
  type        = string
  default     = "~/.ssh/pai-bot-deploy"

  validation {
    condition     = trimspace(var.ssh_private_key_path) != ""
    error_message = "ssh_private_key_path must not be empty."
  }
}

variable "app_dir" {
  description = "Application directory on the server"
  type        = string
  default     = "/opt/pai-bot"
}
