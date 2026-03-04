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
  description = "CIDR blocks allowed to SSH (restrict to your IP)"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "app_dir" {
  description = "Application directory on the server"
  type        = string
  default     = "/opt/pai-bot"
}
