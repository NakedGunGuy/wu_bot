#!/bin/bash

# War Universe Bot - Monitoring Script
# This script checks the health of your bot and dependencies

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print section header
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# Function to check status
check_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
        return 0
    else
        echo -e "${RED}✗ $3${NC}"
        return 1
    fi
}

# System Information
print_header "System Information"
echo "Hostname: $(hostname)"
echo "OS: $(lsb_release -d | cut -f2)"
echo "Kernel: $(uname -r)"
echo "Uptime: $(uptime -p)"

# CPU and Memory
print_header "Resource Usage"
echo "CPU Usage:"
top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print "  Usage: " 100 - $1"%"}'

echo ""
echo "Memory Usage:"
free -h | awk 'NR==2{printf "  Total: %s\n  Used: %s (%.2f%%)\n  Free: %s\n", $2,$3,$3*100/$2,$4}'

echo ""
echo "Disk Usage:"
df -h / | awk 'NR==2{printf "  Total: %s\n  Used: %s (%s)\n  Free: %s\n", $2,$3,$5,$4}'

# Check Node.js
print_header "Node.js Status"
if command -v node &> /dev/null; then
    check_status 0 "Node.js $(node --version) is installed" "Node.js not found"
else
    check_status 1 "" "Node.js not found"
fi

# Check Java
print_header "Java Status"
if command -v java &> /dev/null; then
    JAVA_VERSION=$(java -version 2>&1 | head -n 1 | cut -d'"' -f2)
    check_status 0 "Java $JAVA_VERSION is installed" "Java not found"
else
    check_status 1 "" "Java not found"
fi

# Check MongoDB
print_header "MongoDB Status"
if systemctl is-active --quiet mongod; then
    check_status 0 "MongoDB is running" "MongoDB is not running"
    echo "  Status: $(systemctl show mongod --no-page --property=ActiveState --value)"
    echo "  Uptime: $(systemctl show mongod --no-page --property=ActiveEnterTimestamp --value)"
else
    check_status 1 "" "MongoDB is not running"
fi

# Check PM2
print_header "PM2 Status"
if command -v pm2 &> /dev/null; then
    check_status 0 "PM2 is installed" "PM2 not found"
    echo ""
    pm2 status
else
    check_status 1 "" "PM2 not found"
fi

# Check Bot Process
print_header "Bot Process Status"
if pm2 list | grep -q "wu-bot"; then
    BOT_STATUS=$(pm2 jlist | jq -r '.[] | select(.name=="wu-bot") | .pm2_env.status')
    if [ "$BOT_STATUS" = "online" ]; then
        check_status 0 "Bot is running (Status: $BOT_STATUS)" "Bot is not running"

        # Get more details
        CPU=$(pm2 jlist | jq -r '.[] | select(.name=="wu-bot") | .monit.cpu')
        MEM=$(pm2 jlist | jq -r '.[] | select(.name=="wu-bot") | .monit.memory')
        UPTIME=$(pm2 jlist | jq -r '.[] | select(.name=="wu-bot") | .pm2_env.pm_uptime')
        RESTARTS=$(pm2 jlist | jq -r '.[] | select(.name=="wu-bot") | .pm2_env.restart_time')

        echo "  CPU: ${CPU}%"
        echo "  Memory: $(numfmt --to=iec --format='%.2f' $MEM)"
        echo "  Uptime: $(date -d @$((UPTIME/1000)) -u +%H:%M:%S)"
        echo "  Restarts: $RESTARTS"
    else
        check_status 1 "" "Bot is not running (Status: $BOT_STATUS)"
    fi
else
    check_status 1 "" "Bot process not found in PM2"
fi

# Check Port
print_header "Network Status"
PORT=${PORT:-4646}
if netstat -tuln | grep -q ":$PORT "; then
    check_status 0 "Bot is listening on port $PORT" "Bot is not listening on port $PORT"
else
    check_status 1 "" "Bot is not listening on port $PORT"
fi

# Check API Response
echo ""
echo "Testing API endpoint..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$PORT 2>/dev/null)
if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 400 ]; then
    check_status 0 "API is responding (HTTP $HTTP_CODE)" "API is not responding"
else
    check_status 1 "" "API is not responding (HTTP $HTTP_CODE)"
fi

# Check Logs
print_header "Recent Logs (Last 10 lines)"
if [ -f "logs/combined.log" ]; then
    tail -n 10 logs/combined.log
elif pm2 list | grep -q "wu-bot"; then
    pm2 logs wu-bot --lines 10 --nostream
else
    echo "No logs found"
fi

# Check for errors in logs
print_header "Recent Errors"
if [ -f "logs/err.log" ]; then
    ERROR_COUNT=$(wc -l < logs/err.log)
    if [ $ERROR_COUNT -gt 0 ]; then
        echo -e "${RED}Found $ERROR_COUNT error entries${NC}"
        echo "Last 5 errors:"
        tail -n 5 logs/err.log
    else
        check_status 0 "No errors in log file" ""
    fi
else
    echo "No error log file found"
fi

# MongoDB Stats
print_header "MongoDB Database Stats"
if systemctl is-active --quiet mongod; then
    mongosh --quiet --eval "
    db = db.getSiblingDB('wu_bot');
    print('Collections:');
    db.getCollectionNames().forEach(function(collection) {
        var count = db.getCollection(collection).countDocuments();
        print('  ' + collection + ': ' + count + ' documents');
    });
    " 2>/dev/null || echo "Could not connect to MongoDB"
else
    echo "MongoDB is not running"
fi

# Recommendations
print_header "Recommendations"
RECOMMENDATIONS=0

# Check memory usage
MEM_PERCENT=$(free | grep Mem | awk '{print int($3/$2 * 100)}')
if [ $MEM_PERCENT -gt 80 ]; then
    echo -e "${YELLOW}⚠ Memory usage is high ($MEM_PERCENT%). Consider adding swap space or upgrading RAM.${NC}"
    RECOMMENDATIONS=$((RECOMMENDATIONS + 1))
fi

# Check disk usage
DISK_PERCENT=$(df / | tail -1 | awk '{print int($5)}')
if [ $DISK_PERCENT -gt 80 ]; then
    echo -e "${YELLOW}⚠ Disk usage is high ($DISK_PERCENT%). Consider cleaning up logs or expanding storage.${NC}"
    RECOMMENDATIONS=$((RECOMMENDATIONS + 1))
fi

# Check PM2 restarts
if pm2 list | grep -q "wu-bot"; then
    RESTARTS=$(pm2 jlist | jq -r '.[] | select(.name=="wu-bot") | .pm2_env.restart_time' 2>/dev/null)
    if [ -n "$RESTARTS" ] && [ "$RESTARTS" -gt 5 ]; then
        echo -e "${YELLOW}⚠ Bot has restarted $RESTARTS times. Check logs for recurring issues.${NC}"
        RECOMMENDATIONS=$((RECOMMENDATIONS + 1))
    fi
fi

if [ $RECOMMENDATIONS -eq 0 ]; then
    echo -e "${GREEN}✓ No recommendations at this time${NC}"
fi

# Summary
print_header "Summary"
echo "Monitoring complete at $(date)"
echo ""
echo "For detailed logs: pm2 logs wu-bot"
echo "To restart bot: pm2 restart wu-bot"
echo "For live monitoring: pm2 monit"
echo ""
