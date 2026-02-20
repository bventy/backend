#!/bin/bash
set -e

PORT=8082
BASE_URL="http://localhost:$PORT"
DB_URL="postgresql://neondb_owner:npg_ABuQl7cj5heW@ep-wispy-brook-a1ij8hbi-pooler.ap-southeast-1.aws.neon.tech/bventy_mv1?sslmode=require"

# Tools
function json_val() {
    echo "$1" | jq -r "$2"
}

echo "=== Admin API Verification (Neon DB) ==="

# 1. Create Users
TIMESTAMP=$(date +%s)
USER_EMAIL="user_${TIMESTAMP}@example.com"
ADMIN_EMAIL="admin_${TIMESTAMP}@example.com"
SUPER_EMAIL="super_${TIMESTAMP}@example.com"
PASS="password"

echo "Creating Regular User..."
curl -s -X POST $BASE_URL/auth/signup -H "Content-Type: application/json" -d "{\"email\": \"$USER_EMAIL\", \"password\": \"$PASS\", \"full_name\": \"User V9\"}" > /dev/null
USER_TOKEN=$(curl -s -X POST $BASE_URL/auth/login -H "Content-Type: application/json" -d "{\"email\": \"$USER_EMAIL\", \"password\": \"$PASS\"}" | jq -r .token)

echo "Creating Admin..."
curl -s -X POST $BASE_URL/auth/signup -H "Content-Type: application/json" -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$PASS\", \"full_name\": \"Admin V9\"}" > /dev/null
# Promote to Admin via DB
psql "$DB_URL" -c "UPDATE users SET role = 'admin' WHERE email = '$ADMIN_EMAIL';"
ADMIN_TOKEN=$(curl -s -X POST $BASE_URL/auth/login -H "Content-Type: application/json" -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$PASS\"}" | jq -r .token)

echo "Creating Super Admin..."
curl -s -X POST $BASE_URL/auth/signup -H "Content-Type: application/json" -d "{\"email\": \"$SUPER_EMAIL\", \"password\": \"$PASS\", \"full_name\": \"Super V9\"}" > /dev/null
# Promote to Super Admin via DB
psql "$DB_URL" -c "UPDATE users SET role = 'super_admin' WHERE email = '$SUPER_EMAIL';"
SUPER_TOKEN=$(curl -s -X POST $BASE_URL/auth/login -H "Content-Type: application/json" -d "{\"email\": \"$SUPER_EMAIL\", \"password\": \"$PASS\"}" | jq -r .token)

echo "Tokens obtained."

# 2. Test Access Control
echo -e "\n--- Testing Access Control ---"
echo "Trying /admin/stats with User Token (Expect 403)..."
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $USER_TOKEN" $BASE_URL/admin/stats)
if [ "$STATUS" -eq 403 ]; then echo "SUCCESS: Got 403"; else echo "FAILURE: Got $STATUS"; exit 1; fi

echo "Trying /admin/stats with Admin Token (Expect 200)..."
STATS=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" $BASE_URL/admin/stats)
echo "Stats: $STATS"
if echo "$STATS" | grep -q "total_users"; then echo "SUCCESS: Got Stats"; else echo "FAILURE"; exit 1; fi

# 3. Test Vendor Moderation
echo -e "\n--- Testing Vendor Moderation ---"
# Create pending vendor with User
echo "User creating vendor..."
RESP=$(curl -s -X POST $BASE_URL/vendor/onboard \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"business_name\": \"Vendor $TIMESTAMP\", \"slug\": \"vendor-$TIMESTAMP\", \"category\": \"Photo\", \"city\": \"Test City\", \"whatsapp_link\": \"wa.me/123\"}")
echo "Vendor Create Resp: $RESP"

# Get Pending Vendors as Admin
echo "Admin fetching pending vendors..."
PENDING=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" $BASE_URL/admin/vendors/pending)
echo "Pending Vendors: $PENDING"
VENDOR_ID=$(echo $PENDING | jq -r ".[] | select(.slug==\"vendor-$TIMESTAMP\") | .id")
echo "Found Vendor ID: $VENDOR_ID"

if [ -z "$VENDOR_ID" ] || [ "$VENDOR_ID" == "null" ]; then echo "FAILURE: Could not find pending vendor"; exit 1; fi

# Approve Vendor
echo "Admin approving vendor..."
curl -s -X PATCH $BASE_URL/admin/vendors/$VENDOR_ID/approve -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

# Verify status in DB
STATUS_DB=$(psql "$DB_URL" -t -c "SELECT status FROM vendor_profiles WHERE id='$VENDOR_ID'" | xargs)
if [ "$STATUS_DB" == "verified" ]; then echo "SUCCESS: Vendor verified"; else echo "FAILURE: Status is $STATUS_DB"; exit 1; fi 

# 4. Test User Role Management
echo -e "\n--- Testing User Role Management ---"
USER_ID=$(curl -s -H "Authorization: Bearer $USER_TOKEN" $BASE_URL/me | jq -r .id)
echo "User ID to promote: $USER_ID"

# Admin tries to promote (Should fail if restricted to super_admin)
# Route has `middleware.RequireRole("super_admin")`
echo "Admin trying to promote User (Expect 403)..."
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH $BASE_URL/admin/users/$USER_ID/role \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role": "staff"}')

if [ "$STATUS" -eq 403 ]; then echo "SUCCESS: Admin blocked (403)"; else echo "FAILURE: Admin got $STATUS"; exit 1; fi

# Super Admin promotes
echo "Super Admin promoting User to Staff..."
curl -s -X PATCH $BASE_URL/admin/users/$USER_ID/role \
  -H "Authorization: Bearer $SUPER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role": "staff"}' | jq .

# Verify Role
ROLE_DB=$(psql "$DB_URL" -t -c "SELECT role FROM users WHERE id='$USER_ID'" | xargs)
if [ "$ROLE_DB" == "staff" ]; then echo "SUCCESS: User promoted to staff"; else echo "FAILURE: Role is $ROLE_DB"; exit 1; fi

echo -e "\n=== Verification Complete ==="
