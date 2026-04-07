#!/bin/bash
set -euo pipefail

exec > >(tee /var/log/pai-bot-setup.log) 2>&1

echo "=== P&AI Bot EC2 Setup ==="

# --- System updates ---
apt-get update -y

# --- Install Docker Engine ---
apt-get install -y ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  > /etc/apt/sources.list.d/docker.list

apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

usermod -aG docker ubuntu

# --- Create app directory ---
mkdir -p ${app_dir}
chown ubuntu:ubuntu ${app_dir}

# --- Install git + jq + AWS CLI ---
apt-get install -y git jq unzip
curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o /tmp/awscliv2.zip
unzip -q /tmp/awscliv2.zip -d /tmp && /tmp/aws/install && rm -rf /tmp/aws /tmp/awscliv2.zip

# --- SSM Agent (snap-based on Ubuntu) ---
snap install amazon-ssm-agent --classic
snap start amazon-ssm-agent

echo "=== Setup complete ==="
