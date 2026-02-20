#!/bin/bash
set -e

PORT=8082
BASE_URL="http://localhost:$PORT"

# Tools
function json_val() {
    echo "$1" | jq -r "$2"
}

echo "=== Admin API Debug Verification v2 ==="

# 1. Create Users
TIMESTAMP=$(date +%s)
USER_EMAIL="user_debug2_${TIMESTAMP}@example.com"
ADMIN_EMAIL="admin_debug2_${TIMESTAMP}@example.com"
SUPER_EMAIL="super_debug2_${TIMESTAMP}@example.com"
PASS="password"

echo "Creating Admin ($ADMIN_EMAIL)..."
curl -s -X POST $BASE_URL/auth/signup -H "Content-Type: application/json" -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$PASS\", \"full_name\": \"Admin V9\"}"

# Check if user exists via DB
echo -e "\nChecking DB for Admin user..."
PGPASSWORD='9^B:&,Oe76H\d16p8?' psql -h localhost -p 5432 -U rootonceonkar -d bventy_mv1 -c "SELECT id, email, role FROM users WHERE email = '$ADMIN_EMAIL';"

# Promote to Admin via DB
echo "Promoting to Admin..."
PGPASSWORD='9^B:&,Oe76H\d16p8?' psql -h localhost -p 5432 -U rootonceonkar -d bventy_mv1 -c "UPDATE users SET role = 'admin' WHERE email = '$ADMIN_EMAIL';"

# Check Role again
echo "Checking DB role after update..."
PGPASSWORD='9^B:&,Oe76H\d16p8?' psql -h localhost -p 5432 -U rootonceonkar -d bventy_mv1 -c "SELECT email, role FROM users WHERE email = '$ADMIN_EMAIL';"

ADMIN_TOKEN=$(curl -s -X POST $BASE_URL/auth/login -H "Content-Type: application/json" -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$PASS\"}" | jq -r .token)
echo "Admin Token: $ADMIN_TOKEN"

echo -e "\nTesting /admin/stats with Admin Token..."
curl -v -H "Authorization: Bearer $ADMIN_TOKEN" $BASE_URL/admin/stats

echo -e "\n=== Verification Complete ==="
