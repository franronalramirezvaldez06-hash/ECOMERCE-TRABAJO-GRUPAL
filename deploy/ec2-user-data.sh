#!/bin/bash
set -euo pipefail
exec > >(tee -a /var/log/ecommerce-bootstrap.log) 2>&1

dnf install -y docker git openssl
systemctl enable --now docker
fallocate -l 2G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
echo '/swapfile swap swap defaults 0 0' >> /etc/fstab

TOKEN=$(curl -fsS -X PUT -H 'X-aws-ec2-metadata-token-ttl-seconds: 21600' http://169.254.169.254/latest/api/token)
IP=$(curl -fsS -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/public-ipv4)
DOMAIN="${IP//./-}.sslip.io"
mkdir -p /opt/ecommerce
git clone https://github.com/franronalramirezvaldez06-hash/ECOMERCE-TRABAJO-GRUPAL.git /opt/ecommerce/source
cd /opt/ecommerce/source
git checkout 10e9f0becb419c935438ef968dbac5c30278e495

umask 077
DB_PASSWORD=$(openssl rand -hex 24)
JWT_SECRET=$(openssl rand -hex 32)
ADMIN_PASSWORD=$(openssl rand -hex 18)
printf 'ADMIN_EMAIL=admin@ecommerce.local\nADMIN_PASSWORD=%s\n' "$ADMIN_PASSWORD" > /root/ecommerce-admin.txt
printf 'POSTGRES_USER=ecommerce\nPOSTGRES_PASSWORD=%s\nPOSTGRES_DB=ecommerce\n' "$DB_PASSWORD" > /root/ecommerce-db.env
printf 'APP_ENV=production\nHTTP_PORT=8080\nDATABASE_URL=postgres://ecommerce:%s@db:5432/ecommerce?sslmode=disable\nJWT_SECRET=%s\nADMIN_EMAIL=admin@ecommerce.local\nADMIN_PASSWORD=%s\nALLOWED_ORIGINS=https://%s\nENABLE_SIMULATED_GATEWAYS=true\n' "$DB_PASSWORD" "$JWT_SECRET" "$ADMIN_PASSWORD" "$DOMAIN" > /root/ecommerce-api.env

docker network create ecommerce
docker volume create ecommerce-db
docker volume create ecommerce-caddy-data
docker volume create ecommerce-caddy-config
docker run -d --name ecommerce-db --network ecommerce --network-alias db --restart unless-stopped --env-file /root/ecommerce-db.env -v ecommerce-db:/var/lib/postgresql/data postgres:16-alpine
for i in $(seq 1 60); do
  if docker exec ecommerce-db pg_isready -U ecommerce -d ecommerce; then break; fi
  sleep 2
done
docker build -t ecommerce-api ./backend
docker build --build-arg "NEXT_PUBLIC_API_URL=https://$DOMAIN" -t ecommerce-web ./frontend
docker run -d --name ecommerce-api --network ecommerce --network-alias api --restart unless-stopped --env-file /root/ecommerce-api.env ecommerce-api
docker run -d --name ecommerce-web --network ecommerce --network-alias web --restart unless-stopped ecommerce-web
cat > /opt/ecommerce/Caddyfile <<EOF
$DOMAIN {
  handle /api/* {
    reverse_proxy api:8080
  }
  handle /health {
    reverse_proxy api:8080
  }
  handle {
    reverse_proxy web:3000
  }
}
EOF
docker run -d --name ecommerce-proxy --network ecommerce --restart unless-stopped -p 80:80 -p 443:443 -v /opt/ecommerce/Caddyfile:/etc/caddy/Caddyfile:ro -v ecommerce-caddy-data:/data -v ecommerce-caddy-config:/config caddy:2
echo "URL=https://$DOMAIN" > /opt/ecommerce/url.txt
