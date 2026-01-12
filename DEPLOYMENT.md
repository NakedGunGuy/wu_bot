# Hetzner Server Deployment Guide

This guide will help you deploy the War Universe Bot to your Hetzner server so it runs continuously.

## Prerequisites

- A Hetzner server (VPS) running Ubuntu 20.04+ or Debian 11+
- SSH access to your server
- Domain name (optional, for SSL/HTTPS)

## Step 1: Initial Server Setup

SSH into your Hetzner server:

```bash
ssh root@your_server_ip
```

Update the system:

```bash
apt update && apt upgrade -y
```

## Step 2: Install Node.js

Install Node.js (v18 or later):

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt install -y nodejs
node --version  # Should show v20.x.x
npm --version
```

## Step 3: Install Java JDK 22

Download and install Java JDK 22:

```bash
cd /tmp
wget https://download.oracle.com/java/22/latest/jdk-22_linux-x64_bin.deb
apt install -y ./jdk-22_linux-x64_bin.deb
java -version  # Should show version 22
```

## Step 4: Install MongoDB

Install MongoDB:

```bash
# Import MongoDB public GPG key
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor

# Create list file for MongoDB
echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] http://repo.mongodb.org/apt/debian bullseye/mongodb-org/7.0 main" | tee /etc/apt/sources.list.d/mongodb-org-7.0.list

# Update package database and install MongoDB
apt update
apt install -y mongodb-org

# Start MongoDB
systemctl start mongod
systemctl enable mongod
systemctl status mongod
```

## Step 5: Create Application User

Create a dedicated user for running the bot:

```bash
adduser --system --group --home /opt/wu_bot wu_bot
```

## Step 6: Upload Your Bot Files

Upload your bot files to the server. From your local machine:

```bash
# Using SCP
scp -r C:\Users\Perko\Downloads\wu_bot-main\wu_bot-main root@your_server_ip:/opt/wu_bot/

# Or using rsync
rsync -avz --exclude 'node_modules' C:\Users\Perko\Downloads\wu_bot-main\wu_bot-main/ root@your_server_ip:/opt/wu_bot/
```

Or clone from your git repository directly on the server:

```bash
cd /opt
git clone https://github.com/AlloryDante/War-Universe-PacketBot.git wu_bot
```

## Step 7: Configure Environment Variables

Create and configure the `.env` file:

```bash
cd /opt/wu_bot
nano .env
```

Update with your production values:

```env
# Discord OAuth (if using)
DISCORD_CLIENT_ID=your_discord_client_id
DISCORD_CLIENT_SECRET=your_discord_client_secret

# MongoDB
MONGODB_URI=mongodb://localhost:27017/wu_bot

# Server
PORT=4646
NODE_ENV=production

# Frontend
FRONTEND_URL_PRODUCTION=https://yourdomain.com
JWT_SECRET=generate_a_very_long_random_secret_here
```

Generate a secure JWT secret:

```bash
node -e "console.log(require('crypto').randomBytes(64).toString('hex'))"
```

## Step 8: Install Dependencies

```bash
cd /opt/wu_bot
npm install --production
chown -R wu_bot:wu_bot /opt/wu_bot
```

## Step 9: Install PM2 Process Manager

PM2 will keep your bot running continuously and restart it if it crashes:

```bash
npm install -g pm2
```

## Step 10: Start the Bot with PM2

Start the bot:

```bash
pm2 start server.js --name wu-bot --user wu_bot
```

Configure PM2 to start on system boot:

```bash
pm2 startup systemd
# Run the command that PM2 outputs

pm2 save
```

## Step 11: Useful PM2 Commands

```bash
# View bot status
pm2 status

# View logs
pm2 logs wu-bot

# View real-time logs
pm2 logs wu-bot --lines 100

# Restart the bot
pm2 restart wu-bot

# Stop the bot
pm2 stop wu-bot

# Monitor resources
pm2 monit

# Delete from PM2
pm2 delete wu-bot
```

## Step 12: Configure Firewall

Allow necessary ports:

```bash
# If using ufw
ufw allow 22/tcp    # SSH
ufw allow 4646/tcp  # Bot API
ufw enable

# Check status
ufw status
```

## Step 13: Set Up Nginx Reverse Proxy (Optional)

If you want to access your bot via a domain with SSL:

Install Nginx:

```bash
apt install -y nginx certbot python3-certbot-nginx
```

Create Nginx configuration:

```bash
nano /etc/nginx/sites-available/wu-bot
```

Add this configuration:

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    location / {
        proxy_pass http://localhost:4646;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

Enable the site:

```bash
ln -s /etc/nginx/sites-available/wu-bot /etc/nginx/sites-enabled/
nginx -t
systemctl restart nginx
```

Get SSL certificate:

```bash
certbot --nginx -d yourdomain.com
```

## Step 14: Monitor Your Bot

Check if the bot is running:

```bash
pm2 status
pm2 logs wu-bot
curl http://localhost:4646
```

Check MongoDB:

```bash
mongosh
> show dbs
> use wu_bot
> show collections
```

## Troubleshooting

### Bot won't start

```bash
# Check logs
pm2 logs wu-bot

# Check if port is already in use
netstat -tuln | grep 4646

# Check permissions
ls -la /opt/wu_bot
```

### MongoDB connection issues

```bash
# Check MongoDB status
systemctl status mongod

# Check MongoDB logs
tail -f /var/log/mongodb/mongod.log
```

### Out of memory

```bash
# Check memory usage
free -h

# Add swap space if needed
fallocate -l 2G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
echo '/swapfile none swap sw 0 0' >> /etc/fstab
```

## Updating the Bot

```bash
cd /opt/wu_bot
git pull  # If using git
npm install --production
pm2 restart wu-bot
```

## Backup

Regular MongoDB backups:

```bash
# Create backup directory
mkdir -p /opt/backups

# Backup script
mongodump --db wu_bot --out /opt/backups/wu_bot_$(date +%Y%m%d)

# Create cron job for daily backups
crontab -e
```

Add this line:

```
0 2 * * * mongodump --db wu_bot --out /opt/backups/wu_bot_$(date +\%Y\%m\%d) && find /opt/backups -type d -mtime +7 -exec rm -rf {} +
```

## Security Recommendations

1. Change SSH port from default 22
2. Set up SSH key authentication and disable password login
3. Keep system and dependencies updated
4. Use strong passwords for MongoDB if exposing it
5. Regularly backup your database
6. Monitor server resources and logs
7. Set up fail2ban to prevent brute force attacks

```bash
apt install -y fail2ban
systemctl enable fail2ban
systemctl start fail2ban
```

## Performance Optimization

Edit PM2 configuration for better performance:

```bash
pm2 start server.js --name wu-bot --max-memory-restart 500M --user wu_bot
pm2 save
```

This will automatically restart the bot if it uses more than 500MB of memory.

## Done!

Your War Universe Bot should now be running continuously on your Hetzner server. Access it at:

- Direct: `http://your_server_ip:4646`
- With Nginx/SSL: `https://yourdomain.com`
