#!/bin/bash
set -e

PORT=8082
BASE_URL="http://localhost:$PORT"

echo "1. Health Check"
curl -s $BASE_URL/health | jq .

echo -e "\n\n2. Auth - Signup Vendor"
curl -s -X POST $BASE_URL/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "vendor@example.com", "password": "password123", "role": "vendor", "full_name": "Vendor Full Name"}' | jq .

echo -e "\n3. Auth - Login Vendor"
VENDOR_TOKEN=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "vendor@example.com", "password": "password123"}' | jq -r .token)
echo "Vendor Token obtained"

echo -e "\n4. Onboard Vendor"
VENDOR_ID=$(curl -s -X POST $BASE_URL/vendor/onboard \
  -H "Authorization: Bearer $VENDOR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Best Vendor", "category": "Catering", "city": "New York", "bio": "We provide best food", "whatsapp_link": "wa.me/1234567890"}' | jq -r .vendor_id)
echo "Vendor ID: $VENDOR_ID"

echo -e "\n5. Auth - Signup Admin"
curl -s -X POST $BASE_URL/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "adminpassword", "role": "admin", "full_name": "Admin Full Name"}' | jq .

echo -e "\n6. Auth - Login Admin"
ADMIN_TOKEN=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "adminpassword"}' | jq -r .token)
echo "Admin Token obtained"

echo -e "\n7. Admin - List Pending Vendors"
curl -s -X GET $BASE_URL/admin/vendors/pending \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

echo -e "\n8. Admin - Verify Vendor"
curl -s -X POST $BASE_URL/admin/vendors/$VENDOR_ID/verify \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

echo -e "\n9. Admin - List Pending Vendors (Should be empty)"
curl -s -X GET $BASE_URL/admin/vendors/pending \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
