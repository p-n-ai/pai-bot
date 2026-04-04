# AWS Deployment (EC2 + Docker Compose)

Single EC2 instance running Docker Compose in `ap-southeast-5` (Malaysia).
Cost: ~$20-25/mo.

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

## First-Time Server Setup

```bash
# SSH in (command from terraform output)
ssh -i terraform/pai-bot-key.pem ubuntu@<PUBLIC_IP>

# Clone repo
cd /opt/pai-bot
git clone https://github.com/p-n-ai/pai-bot.git .

# Configure
cp .env.example .env
nano .env
# Set: LEARN_TELEGRAM_BOT_TOKEN, PAI_AUTH_SECRET, AI provider keys
# Change DB/cache/nats URLs to use container names: postgres, dragonfly, nats

# Set postgres password and start
export POSTGRES_PASSWORD=<strong-password>
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Set up nginx
sudo cp deploy/nginx/pai-bot.conf /etc/nginx/sites-available/pai-bot
sudo ln -s /etc/nginx/sites-available/pai-bot /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t && sudo systemctl reload nginx

# Verify
curl http://localhost/healthz
```

## Subsequent Deploys

```bash
DEPLOY_HOST=<IP> DEPLOY_KEY=terraform/pai-bot-key.pem ./scripts/deploy.sh
```

## SSL (after pointing a domain)

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

## Teardown

```bash
terraform destroy
```
