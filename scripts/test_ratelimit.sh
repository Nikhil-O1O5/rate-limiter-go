#!/bin/bash

BASE_URL="http://localhost:8080"
USER_ID="test-user-1"

echo "=== Test 1: Missing X-User-ID header ==="
curl -s -w "\nHTTP %{http_code}\n" "$BASE_URL/feed"

echo ""
echo "=== Test 2: /hash — capacity 3, should 429 on 4th request ==="
for i in $(seq 1 5); do
  RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-User-ID: $USER_ID" \
    -H "Content-Type: application/json" \
    -X POST "$BASE_URL/hash" \
    -d '{"password":"secret"}')
  echo "  /hash request $i: HTTP $RESPONSE"
done

echo ""
echo "=== Test 3: /feed — capacity 20, same user should still have tokens ==="
for i in $(seq 1 5); do
  RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-User-ID: $USER_ID" \
    "$BASE_URL/feed")
  echo "  /feed request $i: HTTP $RESPONSE"
done

echo ""
echo "=== Test 4: Different user on /hash — full bucket ==="
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "X-User-ID: other-user" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/hash" \
  -d '{"password":"secret"}')
echo "  other-user /hash: HTTP $RESPONSE"

echo ""
echo "=== Test 5: Check X-RateLimit-Limit and X-RateLimit-Remaining headers ==="
echo "  --- /hash (capacity 3) ---"
curl -s -D - -o /dev/null \
  -H "X-User-ID: header-test-user" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/hash" \
  -d '{"password":"secret"}' | grep -E "X-RateLimit|Retry-After|HTTP/"

echo "  --- /feed (capacity 20) ---"
curl -s -D - -o /dev/null \
  -H "X-User-ID: header-test-user" \
  "$BASE_URL/feed" | grep -E "X-RateLimit|Retry-After|HTTP/"
