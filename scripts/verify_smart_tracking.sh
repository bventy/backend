#!/bin/bash

# Configuration
API_URL="http://localhost:8080/track/activity"
VENDOR_ID="00000000-0000-0000-0000-000000000001" # Replace with a real vendor ID if testing on live
AUTH_TOKEN="Bearer <replace_with_token>"

echo "Testing Smart View Tracking..."

# Send 50 view requests
for i in {1..50}
do
   curl -X POST $API_URL \
     -H "Content-Type: application/json" \
     -H "Authorization: $AUTH_TOKEN" \
     -d "{\"entity_type\": \"vendor\", \"entity_id\": \"$VENDOR_ID\", \"action_type\": \"view\"}" \
     -s > /dev/null
   
   if [ $((i % 10)) -eq 0 ]; then
     echo "Sent $i tracking requests..."
   fi
done

echo "Done. Please check the 'views_count' in the database for vendor $VENDOR_ID."
echo "If views < 100, it should have increased by exactly 50."
echo "If views >= 100, it should have increased probabilistically (e.g. 1 in 10 chance for +10)."
