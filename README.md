# certrenew

A minimal CLI tool for renewing Let's Encrypt certificates via the DNS-Route53 challenge and reloading nginx.

## How it works

1. Runs `certbot/dns-route53` as a Docker container to renew the certificate
2. Restarts the nginx container to pick up the new cert
3. Verifies the new certificate via TLS dial and prints the expiry date

## Usage

```
certrenew [-config /path/to/config.json] [-dry-run]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `/etc/certrenew/config.json` | Path to config file |
| `-dry-run` | `false` | Print actions without executing |

## Configuration

Copy `config.example.json` and fill in your values:

```json
{
  "letsencrypt_dir": "/mnt/cell_block_d/apps/nginx-proxy/certs/letsencrypt",
  "cert_name": "iillmaticc.link",
  "nginx_container": "nginx-proxy",
  "domain": "iillmaticc.link",
  "aws_access_key_id": "YOUR_ACCESS_KEY_ID",
  "aws_secret_access_key": "YOUR_SECRET_ACCESS_KEY"
}
```

**Secure your config file:**
```bash
chmod 600 /mnt/cell_block_d/apps/certrenew/config.json
```

## TrueNAS Deployment

Create the config directory and file on the host:

```bash
mkdir -p /mnt/cell_block_d/apps/certrenew
cp config.example.json /mnt/cell_block_d/apps/certrenew/config.json
vim /mnt/cell_block_d/apps/certrenew/config.json
chmod 600 /mnt/cell_block_d/apps/certrenew/config.json
```

Deploy as a TrueNAS custom app using `docker-compose.yaml`. The container exits after completion (`restart: "no"`). Trigger a renewal by restarting the container from the TrueNAS UI or via SSH:

```bash
docker restart certrenew
```

## Requirements

- Docker socket access (`/var/run/docker.sock`) — required to run certbot and restart nginx
- AWS IAM credentials with Route53 permissions:
  - `route53:ListHostedZones`
  - `route53:GetChange`
  - `route53:ChangeResourceRecordSets`

## Building locally

```bash
go build -o certrenew ./cmd/certrenew
./certrenew -dry-run
```
