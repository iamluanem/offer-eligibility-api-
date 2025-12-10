# Offer Eligibility API

A high-performance, production-ready API for managing merchant offers and determining user eligibility based on transaction history. Built with Go 1.22+ and SQLite for persistence.

## Features

- **Create/Upsert Offers**: Merchants can create promotional offers with flexible eligibility rules
- **Transaction Ingestion**: Efficiently ingest multiple transactions in a single request
- **Eligibility Calculation**: Real-time calculation of which offers users qualify for based on:
  - Active offer status and time windows
  - Transaction matching (by merchant ID or MCC code)
  - Minimum transaction count within lookback period

## Architecture

The API follows clean architecture principles with clear separation of concerns:

- **Models**: Data structures and request/response types
- **Database**: SQLite persistence layer with optimized indexes
- **Service**: Business logic and eligibility calculations
- **Handler**: HTTP request/response handling
- **Main**: Application entry point and server setup

## Prerequisites

- Go 1.22 or higher
- SQLite3 (usually pre-installed on macOS/Linux)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd offer-eligibility-api-
```

2. Install dependencies:
```bash
go mod download
```

## Running the Server

### Basic Usage

Start the server with default settings (port 8080, database at `./offer_eligibility.db`):

```bash
go run cmd/api/main.go
```

### Custom Configuration

You can customize the port and database path using command-line flags:

```bash
go run cmd/api/main.go -port 3000 -db ./custom_path.db
```

Available flags:
- `-port`: Server port (default: 8080)
- `-db`: Database file path (default: ./offer_eligibility.db)

### Production Build

Build the binary:

```bash
go build -o offer-eligibility-api cmd/api/main.go
```

Run the binary:

```bash
./offer-eligibility-api -port 8080 -db ./offer_eligibility.db
```

## API Endpoints

### 1. Create/Update Offer

**POST** `/offers`

Creates a new offer or updates an existing one (upsert).

**Request Body:**
```json
{
  "id": "7f5e5f2b-8a75-4d5e-9c6e-5c6b1e7e9a01",
  "merchant_id": "a2d1e1a9-8b0c-4a6a-9b3a-2f9f1e0d9c11",
  "mcc_whitelist": ["5812", "5814"],
  "active": true,
  "min_txn_count": 3,
  "lookback_days": 30,
  "starts_at": "2025-10-01T00:00:00Z",
  "ends_at": "2025-10-31T23:59:59Z"
}
```

**Response:** `201 Created`
```json
{
  "id": "7f5e5f2b-8a75-4d5e-9c6e-5c6b1e7e9a01",
  "merchant_id": "a2d1e1a9-8b0c-4a6a-9b3a-2f9f1e0d9c11",
  "mcc_whitelist": ["5812", "5814"],
  "active": true,
  "min_txn_count": 3,
  "lookback_days": 30,
  "starts_at": "2025-10-01T00:00:00Z",
  "ends_at": "2025-10-31T23:59:59Z"
}
```

### 2. Ingest Transactions

**POST** `/transactions`

Ingests one or more transactions for one or more users.

**Request Body:**
```json
{
  "transactions": [
    {
      "id": "e4a3b6b7-0c2e-4e0b-9b7f-2a6c1d9e8f11",
      "user_id": "9b8a7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d",
      "merchant_id": "a2d1e1a9-8b0c-4a6a-9b3a-2f9f1e0d9c11",
      "mcc": "5812",
      "amount_cents": 1250,
      "approved_at": "2025-10-20T12:34:56Z"
    },
    {
      "id": "f9c2a1b0-3d4e-5f6a-7b8c-9d0e1f2a3b4c",
      "user_id": "9b8a7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d",
      "merchant_id": "77777777-1111-2222-3333-444444444444",
      "mcc": "5814",
      "amount_cents": 890,
      "approved_at": "2025-10-20T13:10:00Z"
    }
  ]
}
```

**Response:** `201 Created`
```json
{
  "inserted": 2
}
```

### 3. Get Eligible Offers

**GET** `/users/{user_id}/eligible-offers?now=2025-10-21T10:00:00Z`

Returns all active offers that the user currently qualifies for.

**Query Parameters:**
- `now` (optional): RFC3339 timestamp. If not provided, uses server time.

**Example Request:**
```bash
curl "http://localhost:8080/users/9b8a7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d/eligible-offers?now=2025-10-21T10:00:00Z"
```

**Response:** `200 OK`
```json
{
  "user_id": "9b8a7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d",
  "eligible_offers": [
    {
      "offer_id": "7f5e5f2b-8a75-4d5e-9c6e-5c6b1e7e9a01",
      "reason": ">= 3 matching transactions in last 30 days (found 3)"
    }
  ]
}
```

**Empty Response:** `200 OK`
```json
{
  "user_id": "9b8a7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d",
  "eligible_offers": []
}
```

### Health Check

**GET** `/health`

Returns server health status.

**Response:** `200 OK`
```
OK
```

## Quick Start Testing

### Step 1: Start the Server

First, start the API server in a terminal:

```bash
go run cmd/api/main.go
```

You should see:
```
Starting HTTP server on :8080
Database: ./offer_eligibility.db
```

**Keep this terminal open** - the server will continue running.

### Step 2: Test the API

Now you can test the API. Choose one of the options below:

#### Option 1: Automated Test Script (Recommended)

Open a **new terminal** and run:

```bash
cd offer-eligibility-api-
./test-api.sh
```

**Note:** The test script uses `localhost:3000` by default. If your server is running on port 8080, edit `test-api.sh` and change the `API_URL` variable to `http://localhost:8080`.

This script will:
1. Check server health
2. Create an offer
3. Create 3 matching transactions
4. Check eligible offers for a user

#### Option 2: Manual Testing

If you prefer to test manually, open a **new terminal** and follow these steps:

##### 1. Health Check

```bash
curl http://localhost:8080/health
```

Expected response: `OK`

#### 2. Create an Offer

```bash
curl -X POST http://localhost:8080/offers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "28722f71-6c76-4b23-8dbc-ccd2c3795171",
    "merchant_id": "0b29823e-667e-4bf5-84d7-8eb39973a401",
    "mcc_whitelist": ["5812", "5814"],
    "active": true,
    "min_txn_count": 3,
    "lookback_days": 30,
    "starts_at": "2025-10-01T00:00:00Z",
    "ends_at": "2025-10-31T23:59:59Z"
  }'
```

Expected response: The created offer object (JSON)

##### 3. Create Transactions

```bash
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "transactions": [{
      "id": "510dbcad-af66-4c66-b3dc-a4a0c8fd5c1d",
      "user_id": "d5e5c023-f9b1-4eac-b9bd-f538ccca040d",
      "merchant_id": "0b29823e-667e-4bf5-84d7-8eb39973a401",
      "mcc": "5812",
      "amount_cents": 1250,
      "approved_at": "2025-10-20T12:34:56Z"
    }]
  }'
```

Expected response: `{"inserted": 1}`

##### 4. Check Eligible Offers

```bash
curl "http://localhost:8080/users/d5e5c023-f9b1-4eac-b9bd-f538ccca040d/eligible-offers?now=2025-10-21T10:00:00Z"
```

Expected response: JSON with eligible offers for the user

**Note:** 
- Make sure the server is running before testing (see Step 1 above)
- Replace `localhost:8080` with your server address and port if different
- The UUIDs in the examples are valid UUID v4 format and can be reused for testing

## Running Unit Tests

Run all unit tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

Run tests for a specific package:

```bash
go test ./internal/service
```

Run tests with coverage:

```bash
go test -cover ./...
```

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Note:** Unit tests verify the eligibility logic with isolated test databases. For end-to-end API testing, use the test script described in the Quick Start Testing section above.

## Storage Choice: SQLite

I chose **SQLite** for persistence because:

1. **Simplicity**: No external dependencies or setup required - just a file
2. **Performance**: Excellent for read-heavy workloads with proper indexing
3. **ACID Compliance**: Ensures data integrity
4. **Portability**: Database is a single file that can be easily backed up or moved
5. **Production Ready**: SQLite is battle-tested and used in many production systems

The database schema includes:
- **Offers table**: Stores all merchant offers with indexes on active status and time ranges
- **Transactions table**: Stores user transactions with composite indexes for efficient eligibility queries
- **Optimized indexes**: Specifically designed for the eligibility query pattern (user_id + approved_at, merchant_id, mcc)

The database file persists across server restarts, ensuring data durability.

## What Was Intentionally Skipped

While building a production-ready API, I focused on core functionality and intentionally skipped:

1. **Authentication/Authorization**: No API keys or user authentication (as per requirements)
2. **Rate Limiting**: Not implemented (would be needed in production)
3. **Request Validation**: Basic validation only - UUID format validation skipped (as per requirements)
4. **Graceful Shutdown**: Basic signal handling implemented, but no connection draining
5. **Metrics/Monitoring**: No Prometheus metrics or structured logging
6. **API Versioning**: No version prefix in URLs
7. **Pagination**: Not needed for current use cases
8. **Idempotency**: Transaction ingestion is not idempotent (duplicate IDs will fail)
9. **Concurrent Transaction Handling**: No explicit locking for concurrent eligibility checks

See `DESIGN.md` for more details on what would be added with more time.

## Project Structure

```
offer-eligibility-api-/
├── cmd/
│   └── api/
│       └── main.go          # Application entry point
├── internal/
│   ├── database/
│   │   └── database.go      # Database layer (SQLite)
│   ├── handler/
│   │   └── handler.go       # HTTP handlers
│   ├── models/
│   │   └── models.go        # Data models
│   └── service/
│       ├── service.go       # Business logic
│       └── service_test.go # Unit tests
├── go.mod                   # Go module definition
├── go.sum                   # Go module checksums
├── README.md                # This file
└── DESIGN.md                # Architecture and design decisions
```

## Error Handling

The API returns standard HTTP status codes:

- `200 OK`: Successful GET request
- `201 Created`: Successful POST request
- `400 Bad Request`: Invalid request body or parameters
- `500 Internal Server Error`: Server errors (not exposed to users)

Error responses follow this format:
```json
{
  "error": "error message here"
}
```

## Time Handling

All timestamps are handled in **UTC** using **RFC3339** format:
- Input: Accepts RFC3339 timestamps
- Storage: Stored as RFC3339 strings in SQLite
- Output: Returns RFC3339 timestamps
- Query Parameter: `now` parameter accepts RFC3339 format

## Performance Considerations

- **Database Indexes**: Optimized indexes on frequently queried fields
- **Batch Inserts**: Transactions are inserted in a single database transaction
- **Efficient Queries**: Eligibility queries use indexed lookups
- **Connection Pooling**: SQLite connection is reused across requests

## Security Considerations

- **Input Validation**: Basic validation on all inputs
- **SQL Injection Prevention**: All queries use parameterized statements
- **CORS**: Configured for development (should be restricted in production)
- **Error Messages**: Generic error messages to avoid information leakage

## License

This project is provided as-is for evaluation purposes.

