#!/bin/bash
set -euo pipefail

exec > >(tee /var/log/pai-bot-setup.log) 2>&1

echo "=== P&AI Bot EC2 Setup ==="

# --- System updates ---
apt-get update -y
apt-get upgrade -y

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

# --- Install Nginx ---
apt-get install -y nginx
systemctl enable nginx

# --- Create app directory ---
mkdir -p ${app_dir}
chown ubuntu:ubuntu ${app_dir}

# --- Install git ---
apt-get install -y git

echo "=== Setup complete ==="
