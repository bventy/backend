#!/bin/bash
set -e

PORT=8082
BASE_URL="http://localhost:$PORT"

echo "=== Slug Uniqueness Verification ==="

# 1. Signup User
EMAIL="user_slug_$(date +%s)@example.com"
curl -s -X POST $BASE_URL/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\", \"password\": \"password\", \"role\": \"user\", \"full_name\": \"Slug Tester\"}" > /dev/null

TOKEN=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\", \"password\": \"password\"}" | jq -r .token)
echo "Token obtained."

# 2. Create Group 1
NAME="Slug Test"
CITY="Unique City"
echo "Creating Group 1: $NAME in $CITY"
RES1=$(curl -s -X POST $BASE_URL/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$NAME\", \"city\": \"$CITY\", \"description\": \"Desc 1\"}")
SLUG1=$(echo $RES1 | jq -r .slug)
echo "Group 1 Created. Slug: $SLUG1"

if [ "$SLUG1" == "slug-test-unique-city" ]; then
    echo "Slug 1 matches expected base."
else
    echo "Slug 1 unexpected: $SLUG1"
fi

# 3. Create Group 2 (Conflict)
echo "Creating Group 2: $NAME in $CITY (Should succeed with new slug)"
RES2=$(curl -s -X POST $BASE_URL/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$NAME\", \"city\": \"$CITY\", \"description\": \"Desc 2\"}")
SLUG2=$(echo $RES2 | jq -r .slug)
echo "Group 2 Created. Slug: $SLUG2"

if [ "$SLUG2" != "$SLUG1" ]; then
    echo "SUCCESS: Slugs are different."
else
    echo "FAILURE: Slugs are the same ($SLUG2)."
    exit 1
fi

if [[ "$SLUG2" == "$SLUG1-"* ]]; then
     echo "SUCCESS: New slug has suffix."
else
     echo "FAILURE: New slug format unexpected."
     exit 1
fi

echo "=== Verification Complete ==="
