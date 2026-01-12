# Quick Start Guide - Hetzner Deployment

This is a simplified guide to get your War Universe Bot running on a Hetzner server in under 15 minutes.

## Prerequisites

- Hetzner VPS with Ubuntu 20.04+
- SSH access to your server
- Basic terminal knowledge

## Installation (One-Command Setup)

SSH into your server and run this one-liner:

```bash
curl -fsSL https://raw.githubusercontent.com/AlloryDante/War-Universe-PacketBot/main/install.sh | bash
```

Or follow the manual steps below.

## Manual Setup

### 1. SSH into Your Server

```bash
ssh root@your_server_ip
```

### 2. Install Dependencies

```bash
# Update system
apt update && apt upgrade -y

# Install Node.js 20
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt install -y nodejs

# Install Java JDK 22
cd /tmp
wget https://download.oracle.com/java/22/latest/jdk-22_linux-x64_bin.deb
apt install -y ./jdk-22_linux-x64_bin.deb

# Install MongoDB
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor
echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] http://repo.mongodb.org/apt/debian bullseye/mongodb-org/7.0 main" | tee /etc/apt/sources.list.d/mongodb-org-7.0.list
apt update && apt install -y mongodb-org
systemctl start mongod && systemctl enable mongod

# Install PM2
npm install -g pm2
```

### 3. Upload Your Bot

```bash
# Create directory
mkdir -p /opt/wu_bot
cd /opt/wu_bot

# Clone from Git (or upload via SCP)
git clone https://github.com/AlloryDante/War-Universe-PacketBot.git .
```

### 4. Configure Environment

```bash
# Copy and edit .env file
nano .env
```

Minimum required configuration:

```env
MONGODB_URI=mongodb://localhost:27017/wu_bot
PORT=4646
NODE_ENV=production
FRONTEND_URL_PRODUCTION=http://your_server_ip:4646
JWT_SECRET=your_random_secret_here
```

Generate JWT secret:

```bash
node -e "console.log(require('crypto').randomBytes(64).toString('hex'))"
```

### 5. Install & Start

```bash
# Install dependencies
npm install --production

# Start with PM2
pm2 start server.js --name wu-bot

# Save PM2 config
pm2 save

# Enable PM2 on startup
pm2 startup
```

### 6. Configure Firewall

```bash
# Allow port 4646
ufw allow 4646/tcp
ufw enable
```

### 7. Verify

```bash
# Check status
pm2 status

# View logs
pm2 logs wu-bot

# Test connection
curl http://localhost:4646
```

## Done!

Your bot is now running at: `http://your_server_ip:4646`

## Quick Commands

```bash
pm2 logs wu-bot          # View logs
pm2 restart wu-bot       # Restart
pm2 stop wu-bot          # Stop
pm2 status               # Check status
pm2 monit                # Monitor resources
```

## Adding SSL/Domain (Optional)

```bash
# Install Nginx and Certbot
apt install -y nginx certbot python3-certbot-nginx

# Copy the nginx config example
cp nginx-config-example.conf /etc/nginx/sites-available/wu-bot

# Edit with your domain
nano /etc/nginx/sites-available/wu-bot

# Enable site
ln -s /etc/nginx/sites-available/wu-bot /etc/nginx/sites-enabled/
nginx -t && systemctl restart nginx

# Get SSL certificate
certbot --nginx -d yourdomain.com
```

## Troubleshooting

**Bot won't start:**
```bash
pm2 logs wu-bot  # Check the logs
```

**MongoDB connection error:**
```bash
systemctl status mongod
systemctl restart mongod
```

**Port already in use:**
```bash
netstat -tuln | grep 4646
# Kill the process using that port or change PORT in .env
```

## Need Help?

- Full documentation: See `DEPLOYMENT.md`
- Check logs: `pm2 logs wu-bot`
- MongoDB logs: `/var/log/mongodb/mongod.log`
- System logs: `journalctl -xe`
