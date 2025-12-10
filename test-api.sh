#!/bin/bash

API_URL="http://localhost:3000"

OFFER_ID="28722f71-6c76-4b23-8dbc-ccd2c3795171"
MERCHANT_ID="0b29823e-667e-4bf5-84d7-8eb39973a401"
USER_ID="d5e5c023-f9b1-4eac-b9bd-f538ccca040d"
TXN1_ID="510dbcad-af66-4c66-b3dc-a4a0c8fd5c1d"
TXN2_ID="f0e78b4a-473b-48f2-9b16-5f64b789fcff"
TXN3_ID="42f8015e-9e2d-45df-a573-9cd7673cbfc0"

echo "🧪 Testing Offer Eligibility API"
echo "=================================="
echo ""

echo "1️⃣  Health Check"
echo "----------------"
curl -s "$API_URL/health"
echo ""
echo ""

echo "2️⃣  Creating Offer"
echo "------------------"
curl -X POST "$API_URL/offers" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"$OFFER_ID\",
    \"merchant_id\": \"$MERCHANT_ID\",
    \"mcc_whitelist\": [\"5812\", \"5814\"],
    \"active\": true,
    \"min_txn_count\": 3,
    \"lookback_days\": 30,
    \"starts_at\": \"2025-10-01T00:00:00Z\",
    \"ends_at\": \"2025-10-31T23:59:59Z\"
  }"
echo ""
echo ""

echo "3️⃣  Creating Transactions"
echo "----------------------"
curl -X POST "$API_URL/transactions" \
  -H "Content-Type: application/json" \
  -d "{
    \"transactions\": [
      {
        \"id\": \"$TXN1_ID\",
        \"user_id\": \"$USER_ID\",
        \"merchant_id\": \"$MERCHANT_ID\",
        \"mcc\": \"5812\",
        \"amount_cents\": 1250,
        \"approved_at\": \"2025-10-20T12:34:56Z\"
      },
      {
        \"id\": \"$TXN2_ID\",
        \"user_id\": \"$USER_ID\",
        \"merchant_id\": \"$MERCHANT_ID\",
        \"mcc\": \"5812\",
        \"amount_cents\": 890,
        \"approved_at\": \"2025-10-19T13:10:00Z\"
      },
      {
        \"id\": \"$TXN3_ID\",
        \"user_id\": \"$USER_ID\",
        \"merchant_id\": \"88888888-8888-4888-8888-888888888888\",
        \"mcc\": \"5814\",
        \"amount_cents\": 1500,
        \"approved_at\": \"2025-10-18T10:00:00Z\"
      }
    ]
  }"
echo ""
echo ""

echo "4️⃣  Checking Eligible Offers"
echo "----------------------------------"
curl -s "$API_URL/users/$USER_ID/eligible-offers?now=2025-10-21T10:00:00Z" | python3 -m json.tool 2>/dev/null || curl -s "$API_URL/users/$USER_ID/eligible-offers?now=2025-10-21T10:00:00Z"
echo ""
echo ""

echo "✅ Tests completed!"
