#!/bin/bash
set -e

PORT=8082
BASE_URL="http://localhost:$PORT"
DB_PASS="9^B:&,Oe76H\d16p8?"

# Dynamic User
TIMESTAMP=$(date +%s)
EMAIL="v8user_${TIMESTAMP}@example.com"
echo "Using Email: $EMAIL"

echo "--------------------------------------------------"
echo "🚀 Starting V8 Verification (Port $PORT)"
echo "--------------------------------------------------"

echo "1. Health Check"
curl -s $BASE_URL/health | jq .

# --- Auth ---

echo -e "\n2. Signup (Default Role: User)"
signup_res=$(curl -s -X POST $BASE_URL/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\", \"password\": \"password123\", \"full_name\": \"V8 User Fix\", \"phone\": \"1234567890\", \"username\": \"v8user_$TIMESTAMP\"}")
USER_ID=$(echo $signup_res | jq -r .user.id)
echo "User ID: $USER_ID"

if [ "$USER_ID" == "null" ]; then
  echo "Signup Failed: $signup_res"
  exit 1
fi

echo -e "\n3. Login"
login_res=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\", \"password\": \"password123\"}")
TOKEN=$(echo $login_res | jq -r .token)
echo "Token obtained: ${TOKEN:0:10}..."

if [ "$TOKEN" == "null" ]; then
  echo "Login Failed: $login_res"
  exit 1
fi

# --- Groups ---

echo -e "\n5. Create Group"
GROUP_RES=$(curl -s -X POST $BASE_URL/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Tech Meetups Fix $TIMESTAMP\",
    \"city\": \"San Francisco\",
    \"description\": \"Tech lovers\"
  }")
GROUP_ID=$(echo $GROUP_RES | jq -r .group_id)
# Also print full response if ID null
if [ "$GROUP_ID" == "null" ]; then
  echo "DEBUG GROUP RES: $GROUP_RES"
fi
echo "Group Created: $GROUP_ID"

# --- Events ---

echo -e "\n8. Create Group Event"
EVENT_GROUP_RES=$(curl -s -X POST $BASE_URL/events \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"Tech Conference Fix\",
    \"city\": \"SF Center\",
    \"event_date\": \"2026-11-20\",
    \"organizer_group_id\": \"$GROUP_ID\"
  }")
EVENT_GROUP_ID=$(echo $EVENT_GROUP_RES | jq -r .event_id)
echo "Group Event: $EVENT_GROUP_ID"
if [ "$EVENT_GROUP_ID" == "null" ]; then
  echo "DEBUG EVENT RES: $EVENT_GROUP_RES"
fi

# --- Vendor Onboarding ---

echo -e "\n10. Vendor Onboarding"
VENDOR_RES=$(curl -s -X POST $BASE_URL/vendor/onboard \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"business_name\": \"Pro Audio Fix $TIMESTAMP\",
    \"category\": \"Sound\",
    \"city\": \"New York\",
    \"bio\": \"Loudest sound\",
    \"whatsapp_link\": \"wa.me/999\"
  }")
VENDOR_ID=$(echo $VENDOR_RES | jq -r .vendor_id)
echo "Vendor Created: $VENDOR_ID"

# --- Admin (Bootstrap) ---
echo -e "\n13. BOOTSTRAP: Promote to Admin AND Grant Permission"
export PGPASSWORD=$DB_PASS
psql -U rootonceonkar -d bventy_mv1 -h localhost -c "UPDATE users SET role='admin' WHERE id='$USER_ID';" -q

# Fetch Permission ID for 'vendor.verify'
PERM_ID=$(psql -U rootonceonkar -d bventy_mv1 -h localhost -t -c "SELECT id FROM permissions WHERE code='vendor.verify';" | xargs)
echo "Permission ID: $PERM_ID"

# Grant Permission
psql -U rootonceonkar -d bventy_mv1 -h localhost -c "INSERT INTO user_permissions (user_id, permission_id) VALUES ('$USER_ID', '$PERM_ID');" -q

echo -e "\n14. Admin: Verify Vendor"
# Re-login for role update
login_res_admin=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\", \"password\": \"password123\"}")
TOKEN=$(echo $login_res_admin | jq -r .token)

curl -s -X POST $BASE_URL/admin/vendors/$VENDOR_ID/verify \
  -H "Authorization: Bearer $TOKEN" | jq .

echo -e "\n15. Public: List Vendors"
curl -s -X GET $BASE_URL/vendors | jq .

echo -e "\n✅ V8 Verification Complete!"
