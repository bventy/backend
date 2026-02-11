#!/bin/bash
set -e

PORT=8082
BASE_URL="http://localhost:$PORT"
DB_PASS="9^B:&,Oe76H\d16p8?"

echo "--------------------------------------------------"
echo "🚀 Starting Verification (Port $PORT)"
echo "--------------------------------------------------"

echo "1. Health Check"
curl -s $BASE_URL/health | jq .

# --- Auth ---

echo -e "\n2. Signup (Default Role: User)"
signup_res=$(curl -s -X POST $BASE_URL/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "myuser@example.com", "password": "password123"}')
USER_ID=$(echo $signup_res | jq -r .user.id)
echo "User ID: $USER_ID"

echo -e "\n3. Login"
TOKEN=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "myuser@example.com", "password": "password123"}' | jq -r .token)
echo "Token obtained"

echo -e "\n4. Get Me (Check Default Role)"
curl -s -X GET $BASE_URL/me \
  -H "Authorization: Bearer $TOKEN" | jq .

# --- Access Control ---

echo -e "\n5. Try Admin Route (Should Fail 403 Forbidden)"
curl -s -X GET $BASE_URL/admin/vendors/pending \
  -H "Authorization: Bearer $TOKEN" | jq .

# --- Role Promotion (Bootstrap via SQL) ---

echo -e "\n6. BOOTSTRAP: Promoting user to SUPER_ADMIN via SQL"
export PGPASSWORD=$DB_PASS
psql -U rootonceonkar -d bventy_mv1 -h localhost -c "UPDATE users SET role='super_admin' WHERE id='$USER_ID';"
# Also give permission needed for admin routes for later tests if logic requires 'vendor.verify' explicitly,
# but super_admin bypasses RequirePermission in my middleware logic!

echo -e "\n7. Get Me (Check Role Update)"
# Re-login to get updated claims if token stored role (It does!)
TOKEN=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "myuser@example.com", "password": "password123"}' | jq -r .token)
curl -s -X GET $BASE_URL/me \
  -H "Authorization: Bearer $TOKEN" | jq .

# --- Onboarding ---

echo -e "\n8. Vendor Onboarding"
VENDOR_RES=$(curl -s -X POST $BASE_URL/vendor/onboard \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Super Catering",
    "category": "Catering",
    "city": "New York",
    "bio": "Best food in NYC",
    "whatsapp_link": "wa.me/123"
  }')
VENDOR_ID=$(echo $VENDOR_RES | jq -r .vendor_id)
SLUG=$(echo $VENDOR_RES | jq -r .slug)
echo "Vendor Created: ID=$VENDOR_ID, Slug=$SLUG"

echo -e "\n9. Organizer Onboarding"
curl -s -X POST $BASE_URL/organizer/onboard \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "NYC Events",
    "city": "New York"
  }' | jq .

# --- Admin Flow ---

echo -e "\n10. Admin: List Pending Vendors"
# Note: super_admin has 'admin.manage'? 
# Middleware check: RequirePermission('vendor.verify').
# My logic: "Allow super_admin to bypass permission check" -> YES.
curl -s -X GET $BASE_URL/admin/vendors/pending \
  -H "Authorization: Bearer $TOKEN" | jq .

echo -e "\n11. Admin: Verify Vendor"
curl -s -X POST $BASE_URL/admin/vendors/$VENDOR_ID/verify \
  -H "Authorization: Bearer $TOKEN" | jq .

# --- Public Market ---

echo -e "\n12. Public: List Verified Vendors"
curl -s -X GET $BASE_URL/vendors | jq .

echo -e "\n13. Public: Get Vendor By Slug"
curl -s -X GET $BASE_URL/vendors/slug/$SLUG | jq .

echo -e "\n✅ Verification Complete!"
