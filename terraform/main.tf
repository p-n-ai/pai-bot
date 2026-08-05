# P&AI Bot — Single EC2 + Docker Compose deployment
# Region: ap-southeast-5 (Malaysia)
#
# The server only needs Docker + docker compose. Registry operations and
# controlled release orchestration happen in GitHub Actions.
#
# Usage:
#   cd terraform
#   terraform init
#   terraform plan -var="ssh_cidr_blocks=[\"YOUR_IP/32\"]" -var="ssh_public_key=$(cat ~/.ssh/pai-bot-deploy.pub)"
#   terraform apply

terraform {
  required_version = ">= 1.7"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# --- SSH Key Pair ---

removed {
  from = tls_private_key.deploy

  lifecycle {
    destroy = false
  }
}

removed {
  from = local_file.private_key

  lifecycle {
    destroy = false
  }
}

resource "aws_key_pair" "deploy" {
  key_name   = "${var.project}-key"
  public_key = trimspace(var.ssh_public_key)
}

# --- Security Group ---

resource "aws_security_group" "app" {
  name        = "${var.project}-sg"
  description = "P&AI Bot: SSH + HTTP + HTTPS"

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.ssh_cidr_blocks
  }

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name    = "${var.project}-sg"
    Project = var.project
  }
}

# --- EC2 Instance ---

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

resource "aws_instance" "app" {
  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.instance_type
  key_name               = aws_key_pair.deploy.key_name
  vpc_security_group_ids = [aws_security_group.app.id]

  root_block_device {
    volume_size = var.volume_size_gb
    volume_type = "gp3"
    encrypted   = true
  }

  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required" # IMDSv2 only
  }

  user_data = templatefile("${path.module}/user-data.sh", {
    app_dir = var.app_dir
  })

  tags = {
    Name    = "${var.project}-server"
    Project = var.project
  }
}
