# AWS Deployment (EC2 + Docker Compose)

Single EC2 instance running Docker Compose in `ap-southeast-5` (Malaysia).
Cost: ~$20-25/mo.

The server only needs Docker + docker compose. All AWS operations (ECR push, secrets) happen in GitHub Actions CI. This makes the server portable to any VPS.

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.5
- AWS CLI configured with access to ap-southeast-5

## Deploy Infrastructure

```bash
cd terraform
terraform init
terraform plan -var='ssh_cidr_blocks=["YOUR_IP/32"]'
terraform apply -var='ssh_cidr_blocks=["YOUR_IP/32"]'
```

**Important:** `ssh_cidr_blocks` has no default — you must set it to restrict SSH access.

## First-Time Server Setup

```bash
# SSH in (command from terraform output)
ssh -i terraform/pai-bot-key.pem ubuntu@<PUBLIC_IP>

# Create app directory (done by user-data, but verify)
ls /opt/pai-bot

# Verify Docker is running
docker info
```

After the EC2 instance is provisioned, set up GitHub Actions secrets:
- `DEPLOY_HOST` — public IP from terraform output
- `DEPLOY_USER` — `ubuntu`
- `DEPLOY_KEY` — contents of `terraform/pai-bot-key.pem`
- `DEPLOY_DIR` — `/opt/pai-bot`
- `AWS_ROLE_ARN` — GitHub Actions OIDC role ARN
- `AWS_REGION` — `ap-southeast-5`
- `ECR_REGISTRY` — `<account>.dkr.ecr.ap-southeast-5.amazonaws.com`

Secrets are stored in AWS Secrets Manager (`pai-bot/production-env`).

## How Deploys Work

1. CI runs tests (Go + admin lint/test)
2. CI builds Docker images and pushes to ECR with git SHA tags
3. CI fetches secrets from Secrets Manager, writes `.env` via SSH
4. CI copies compose files + migrations via SCP
5. Server runs `scripts/deploy-remote.sh`: ECR pull → migrate → rollout → health check

The server never needs AWS CLI — CI generates a short-lived ECR token and passes it via SSH.

## Caddy (HTTPS)

Caddy runs in Docker and handles TLS automatically when `DOMAIN` is set in Secrets Manager. No nginx, no certbot.

## Deploy to any VPS

To deploy to a non-AWS VPS:
1. Set up a server with Docker + docker compose
2. Push images to any registry (Docker Hub, GHCR, etc.) instead of ECR
3. Pass the registry token to `scripts/deploy-remote.sh` via SSH
4. Write `.env` with your secrets however you prefer

The deploy script (`scripts/deploy-remote.sh`) has zero AWS dependency.

## Teardown

```bash
terraform destroy
```
