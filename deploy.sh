#!/bin/bash

# War Universe Bot - Deployment Script for Hetzner Server
# This script automates the deployment process

set -e  # Exit on error

echo "=========================================="
echo "War Universe Bot - Deployment Script"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo -e "${RED}Please do not run this script as root. Run as regular user with sudo privileges.${NC}"
    exit 1
fi

# Function to print success messages
success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error messages
error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to print info messages
info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Check if Node.js is installed
check_nodejs() {
    if ! command -v node &> /dev/null; then
        error "Node.js is not installed. Please install Node.js v18 or later."
        exit 1
    fi
    success "Node.js $(node --version) is installed"
}

# Check if Java is installed
check_java() {
    if ! command -v java &> /dev/null; then
        error "Java is not installed. Please install Java JDK 22 or later."
        exit 1
    fi
    success "Java $(java -version 2>&1 | head -n 1) is installed"
}

# Check if MongoDB is running
check_mongodb() {
    if ! systemctl is-active --quiet mongod; then
        error "MongoDB is not running. Please start MongoDB: sudo systemctl start mongod"
        exit 1
    fi
    success "MongoDB is running"
}

# Install PM2 if not installed
install_pm2() {
    if ! command -v pm2 &> /dev/null; then
        info "Installing PM2..."
        sudo npm install -g pm2
        success "PM2 installed"
    else
        success "PM2 is already installed"
    fi
}

# Install dependencies
install_dependencies() {
    info "Installing Node.js dependencies..."
    npm install --production
    success "Dependencies installed"
}

# Create logs directory
create_logs_dir() {
    if [ ! -d "logs" ]; then
        mkdir -p logs
        success "Logs directory created"
    else
        success "Logs directory exists"
    fi
}

# Check .env file
check_env_file() {
    if [ ! -f ".env" ]; then
        error ".env file not found. Please create it from .env.example"
        exit 1
    fi
    success ".env file exists"
}

# Start bot with PM2
start_bot_pm2() {
    info "Starting bot with PM2..."

    # Stop if already running
    if pm2 list | grep -q "wu-bot"; then
        pm2 delete wu-bot
    fi

    # Start using ecosystem config if exists, otherwise use simple start
    if [ -f "ecosystem.config.js" ]; then
        pm2 start ecosystem.config.js
    else
        pm2 start server.js --name wu-bot
    fi

    success "Bot started with PM2"
}

# Configure PM2 startup
setup_pm2_startup() {
    info "Configuring PM2 to start on system boot..."
    pm2 save
    sudo env PATH=$PATH:/usr/bin pm2 startup systemd -u $USER --hp $HOME
    success "PM2 startup configured"
}

# Display PM2 status
show_status() {
    echo ""
    echo "=========================================="
    echo "Bot Status:"
    echo "=========================================="
    pm2 status
    echo ""
    echo "=========================================="
    echo "Recent Logs:"
    echo "=========================================="
    pm2 logs wu-bot --lines 20 --nostream
}

# Main deployment process
main() {
    echo ""
    info "Starting deployment checks..."
    echo ""

    check_nodejs
    check_java
    check_mongodb
    check_env_file

    echo ""
    info "Installing dependencies..."
    echo ""

    install_pm2
    create_logs_dir
    install_dependencies

    echo ""
    info "Starting bot..."
    echo ""

    start_bot_pm2
    setup_pm2_startup

    echo ""
    success "Deployment completed successfully!"
    echo ""

    show_status

    echo ""
    echo "=========================================="
    echo "Useful Commands:"
    echo "=========================================="
    echo "View logs:       pm2 logs wu-bot"
    echo "Stop bot:        pm2 stop wu-bot"
    echo "Restart bot:     pm2 restart wu-bot"
    echo "Bot status:      pm2 status"
    echo "Monitor:         pm2 monit"
    echo "=========================================="
}

# Run main function
main
