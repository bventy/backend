#!/bin/bash
set -e

PORT=8082
BASE_URL="http://localhost:$PORT"

# Tools
function json_val() {
    echo "$1" | jq -r "$2"
}

echo "=== V9 Group Management Verification ==="

# 1. Signup Owner
echo -e "\n1. Signing up Owner..."
OWNER_EMAIL="owner_v9_$(date +%s)@example.com"
curl -s -X POST $BASE_URL/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$OWNER_EMAIL\", \"password\": \"password\", \"role\": \"user\", \"full_name\": \"Owner V9\"}" > /dev/null

OWNER_TOKEN=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$OWNER_EMAIL\", \"password\": \"password\"}" | jq -r .token)
echo "Owner Token obtained."

# 2. Create Group
echo -e "\n2. Creating Group..."
GROUP_RES=$(curl -s -X POST $BASE_URL/groups \
  -H "Authorization: Bearer $OWNER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"V9 Community $(date +%s)\", \"city\": \"Cyberpunk City\", \"description\": \"Original Desc\"}")
GROUP_ID=$(echo $GROUP_RES | jq -r .group_id)
echo "Group Created: $GROUP_ID"

# 3. Signup Invitee
echo -e "\n3. Signing up Invitee..."
INVITEE_EMAIL="invitee_v9_$(date +%s)@example.com"
curl -s -X POST $BASE_URL/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$INVITEE_EMAIL\", \"password\": \"password\", \"role\": \"user\", \"full_name\": \"Invitee V9\"}" > /dev/null

INVITEE_TOKEN=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$INVITEE_EMAIL\", \"password\": \"password\"}" | jq -r .token)
echo "Invitee Token obtained."

INVITEE_ID=$(curl -s -H "Authorization: Bearer $INVITEE_TOKEN" $BASE_URL/me | jq -r .id)
echo "Invitee ID: $INVITEE_ID"

# 4. Invite Member
echo -e "\n4. Inviting Member..."
curl -s -X POST $BASE_URL/groups/$GROUP_ID/invite \
  -H "Authorization: Bearer $OWNER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$INVITEE_EMAIL\", \"role\": \"member\"}" | jq .

# Get Invite ID (Simulating user checking email/db, here we cheat and query DB or just List Invites if endpoint existed, but it doesn't. 
# Wait, AcceptInvite needs invite_id. How does user get it? By email. 
# For test, I need to fetch it from DB. Run psql command.
INVITE_ID=$(export PGPASSWORD='9^B:&,Oe76H\d16p8?' && psql -h localhost -p 5432 -U rootonceonkar -d bventy_mv1 -t -c "SELECT id FROM group_invites WHERE invited_email='$INVITEE_EMAIL' ORDER BY created_at DESC LIMIT 1" | xargs)
echo "Invite ID: $INVITE_ID"

# 5. Accept Invite
echo -e "\n5. Accepting Invite..."
curl -s -X POST $BASE_URL/groups/invites/accept \
  -H "Authorization: Bearer $INVITEE_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"invite_id\": \"$INVITE_ID\"}" | jq .

# 6. List Members (Owner)
echo -e "\n6. Listing Members (Owner)..."
MEMBERS=$(curl -s -H "Authorization: Bearer $OWNER_TOKEN" $BASE_URL/groups/$GROUP_ID/members)
echo $MEMBERS | jq .
COUNT=$(echo $MEMBERS | jq '. | length')
if [ "$COUNT" -eq 2 ]; then
    echo "SUCCESS: 2 members found"
else
    echo "FAILURE: Expected 2 members, got $COUNT"
    exit 1
fi

# 7. Promote to Manager
echo -e "\n7. Promoting Invitee to Manager..."
curl -s -X PATCH $BASE_URL/groups/$GROUP_ID/members/$INVITEE_ID \
  -H "Authorization: Bearer $OWNER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role": "manager"}' | jq .

# 8. Manager Updates Group
echo -e "\n8. Manager updating Group..."
curl -s -X PATCH $BASE_URL/groups/$GROUP_ID \
  -H "Authorization: Bearer $INVITEE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"description": "Updated by Manager"}' | jq .

# Verify update
NEW_DESC=$(curl -s -H "Authorization: Bearer $OWNER_TOKEN" $BASE_URL/groups/my | jq -r ".[] | select(.id==\"$GROUP_ID\") | .description")
# Wait, ListMyGroups doesn't return description in the compact list from requirement? 
# Requirement: 
# GET /groups/my should return: [ { id, name, city, role_in_group } ]
# It does NOT return description. 
# So I can't verify description via ListMyGroups. 
# But I can Verify via DB or if there was a GetGroup endpoint? GET /groups/:id is not specified in "REQUIRED BACKEND FEATURES".
# But checking DB is fine.
SAVED_DESC=$(export PGPASSWORD='9^B:&,Oe76H\d16p8?' && psql -h localhost -p 5432 -U rootonceonkar -d bventy_mv1 -t -c "SELECT description FROM groups WHERE id='$GROUP_ID'" | xargs)
if [ "$SAVED_DESC" == "Updated by Manager" ]; then
    echo "SUCCESS: Description updated"
else
    echo "FAILURE: Description match failed. Got '$SAVED_DESC'"
    exit 1
fi

# 9. Remove Member
echo -e "\n9. Removing Manager..."
curl -s -X DELETE $BASE_URL/groups/$GROUP_ID/members/$INVITEE_ID \
  -H "Authorization: Bearer $OWNER_TOKEN" | jq .

# 10. List Members again
echo -e "\n10. Listing Members (Expected 1)..."
MEMBERS_FINAL=$(curl -s -H "Authorization: Bearer $OWNER_TOKEN" $BASE_URL/groups/$GROUP_ID/members)
echo $MEMBERS_FINAL | jq .
COUNT_FINAL=$(echo $MEMBERS_FINAL | jq '. | length')
if [ "$COUNT_FINAL" -eq 1 ]; then
    echo "SUCCESS: Member removed, 1 remaining"
else
    echo "FAILURE: Expected 1 member, got $COUNT_FINAL"
    exit 1
fi

echo -e "\n=== Verification Complete ==="
