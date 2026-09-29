#!/bin/bash

BASE_URL="http://localhost:8080"
USER_ID="test-user-1"

echo "=== Test 1: Missing X-User-ID header ==="
curl -s -w "\nHTTP %{http_code}\n" "$BASE_URL/feed"

echo ""
echo "=== Test 2: First request (should be allowed) ==="
curl -s -w "\nHTTP %{http_code}\n" \
  -H "X-User-ID: $USER_ID" \
  "$BASE_URL/feed"

echo ""
echo "=== Test 3: Hammer /feed 15 times — expect 429 after 10 ==="
for i in $(seq 1 15); do
  RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-User-ID: $USER_ID" \
    "$BASE_URL/feed")
  echo "  Request $i: HTTP $RESPONSE"
done

echo ""
echo "=== Test 4: Different user — should have full bucket ==="
curl -s -w "\nHTTP %{http_code}\n" \
  -H "X-User-ID: other-user" \
  "$BASE_URL/feed"

echo ""
echo "=== Test 5: Wait 5s for tokens to refill, then retry ==="
echo "  Waiting 5 seconds..."
sleep 5
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "X-User-ID: $USER_ID" \
  "$BASE_URL/feed")
echo "  After refill: HTTP $RESPONSE"

echo ""
echo "=== Test 6: Check Retry-After and X-RateLimit-Remaining headers ==="
curl -s -D - -o /dev/null \
  -H "X-User-ID: $USER_ID" \
  "$BASE_URL/feed" | grep -E "X-RateLimit|Retry-After|HTTP/"
