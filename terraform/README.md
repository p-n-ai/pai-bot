# AWS Deployment (EC2 + Docker Compose)

Single EC2 instance running Docker Compose in `ap-southeast-5` (Malaysia).
Cost: ~$20-25/mo.

The server only needs Docker + docker compose. GitHub Actions owns registry
operations and release orchestration, then writes runtime configuration to the
server over SSH. This keeps the server portable to any VPS.

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.7
- AWS CLI configured with access to ap-southeast-5

## New Deployment

```bash
cd terraform
ssh-keygen -t ed25519 -f ~/.ssh/pai-bot-deploy
terraform init
terraform plan \\
  -var='ssh_cidr_blocks=["YOUR_IP/32"]' \\
  -var="ssh_public_key=$(cat ~/.ssh/pai-bot-deploy.pub)"
terraform apply \\
  -var='ssh_cidr_blocks=["YOUR_IP/32"]' \\
  -var="ssh_public_key=$(cat ~/.ssh/pai-bot-deploy.pub)"
```

Both `ssh_cidr_blocks` and `ssh_public_key` are required. Generate and store
the private key outside Terraform; only the public key enters Terraform state.

## Existing Deployment Migration

Do not generate a replacement key before this migration. Reuse the private key
that matches the public key on the running server.

```bash
cd terraform
install -m 0600 pai-bot-key.pem ~/.ssh/pai-bot-deploy
ssh-keygen -y -f ~/.ssh/pai-bot-deploy > ~/.ssh/pai-bot-deploy.pub
ssh -i ~/.ssh/pai-bot-deploy ubuntu@<PUBLIC_IP>
terraform init -upgrade
terraform plan \
  -var='ssh_cidr_blocks=["YOUR_IP/32"]' \
  -var="ssh_public_key=$(cat ~/.ssh/pai-bot-deploy.pub)"
```

Check the plan before you apply it. It must not replace the EC2 instance. If it
replaces the AWS key pair, proceed only when the successful SSH check proves
that the supplied private key already opens the server. The two `removed`
blocks remove the generated key resources from Terraform state without
deleting the existing private-key file.

Apply the same variables only after the plan is safe:

```bash
terraform apply \
  -var='ssh_cidr_blocks=["YOUR_IP/32"]' \
  -var="ssh_public_key=$(cat ~/.ssh/pai-bot-deploy.pub)"
```

Keep the old key until the copied key works in a separate SSH session. Old
Terraform state versions can still contain the private key. Protect that state
history, and rotate the key after this migration.

## Rotate an Existing Key

AWS key-pair replacement does not change `authorized_keys` on a running EC2
instance. Add and test the new public key before you change Terraform.

```bash
ssh-keygen -t ed25519 -f ~/.ssh/pai-bot-next
cat ~/.ssh/pai-bot-next.pub | \
  ssh -i ~/.ssh/pai-bot-deploy ubuntu@<PUBLIC_IP> \
  'umask 077; mkdir -p ~/.ssh; cat >> ~/.ssh/authorized_keys'
ssh -i ~/.ssh/pai-bot-next ubuntu@<PUBLIC_IP>
```

Keep that second SSH session open. Then run `terraform plan` with the new
public key and `-var='ssh_private_key_path=~/.ssh/pai-bot-next'`. Update the
`DEPLOY_KEY` secret before you remove the old key.

## First-Time Server Setup

```bash
# SSH in (command from terraform output)
ssh -i ~/.ssh/pai-bot-deploy ubuntu@<PUBLIC_IP>

# Create app directory (done by user-data, but verify)
ls /opt/pai-bot

# Verify Docker is running
docker info
```

After the EC2 instance is provisioned, configure these as repository or
organization secrets so the Nightly workflow can access them:

- `AWS_ROLE_ARN` — GitHub Actions OIDC role ARN
- `AWS_REGION` — `ap-southeast-5`
- `ECR_REGISTRY` — `<account>.dkr.ecr.ap-southeast-5.amazonaws.com`

Configure deploy-only values as repository secrets or in the GitHub
`production` environment:

- `DEPLOY_HOST` — public IP from terraform output
- `DEPLOY_USER` — `ubuntu`
- `DEPLOY_KEY` — contents of the externally managed `~/.ssh/pai-bot-deploy`
- `DEPLOY_DIR` — `/opt/pai-bot`

Configure the production runtime values consumed by
`.github/workflows/deploy.yml` in the same deployment settings. See
[Nightly candidates and stable releases](../docs/releases.md) for the complete
release contract and [Runtime AI settings](../docs/operations/runtime-ai-settings.md)
for production secret requirements.

## How Releases Work

1. CI validates a push to `main`.
2. The Nightly workflow builds or reuses SHA-addressed images and records their
   immutable digests in a candidate artifact.
3. A maintainer manually dispatches the Stable workflow with a successful
   candidate run ID and semantic version.
4. Stable verifies candidate provenance, image availability, and production
   secrets, then copies deployment files and writes `.env` over SSH.
5. The server runs `scripts/deploy-remote.sh`: backup → digest pull → migration
   → rollout → health and smoke checks.
6. GitHub publishes the semantic tag and Release only after deployment
   succeeds.

The server never needs AWS CLI. GitHub Actions generates a short-lived ECR
token and passes it to the deployment script. A merge to `main` does not deploy
production.

## Caddy (HTTPS)

Caddy runs in Docker and handles TLS automatically when the `DOMAIN` GitHub
secret is set. No nginx, no certbot.

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
