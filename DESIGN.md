# Design Document

## Overview

This document explains the architecture, design decisions, and implementation details of the Offer Eligibility API.

## Architecture

The API follows a **layered architecture** with clear separation of concerns:

```
┌─────────────────┐
│   HTTP Layer    │  (Handler)
├─────────────────┤
│  Business Logic │  (Service)
├─────────────────┤
│  Data Access    │  (Database)
├─────────────────┤
│   Persistence   │  (SQLite)
└─────────────────┘
```

### Layer Responsibilities

1. **Handler Layer** (`internal/handler`): HTTP request/response handling, JSON encoding/decoding, parameter extraction
2. **Service Layer** (`internal/service`): Business logic, validation, eligibility calculations
3. **Database Layer** (`internal/database`): Data persistence, SQL queries, transaction management
4. **Models Layer** (`internal/models`): Data structures and DTOs

## Data Storage

### Choice: SQLite

I chose **SQLite** for the following reasons:

1. **Zero Configuration**: No external database server required
2. **File-Based**: Easy to backup, move, or version control
3. **ACID Compliant**: Ensures data integrity
4. **Performance**: Excellent for read-heavy workloads with proper indexing
5. **Production Ready**: Used by many production systems (GitHub, Android, iOS, etc.)

### Schema Design

#### Offers Table

```sql
CREATE TABLE offers (
    id TEXT PRIMARY KEY,
    merchant_id TEXT NOT NULL,
    mcc_whitelist TEXT NOT NULL,  -- JSON array of MCC codes
    active INTEGER NOT NULL,
    min_txn_count INTEGER NOT NULL,
    lookback_days INTEGER NOT NULL,
    starts_at TEXT NOT NULL,      -- RFC3339 timestamp
    ends_at TEXT NOT NULL,         -- RFC3339 timestamp
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
)
```

**Design Decisions:**
- `mcc_whitelist` stored as JSON string for flexibility
- `active` stored as INTEGER (0/1) for SQLite compatibility
- Timestamps stored as TEXT in RFC3339 format for readability and portability
- No foreign keys (simplified for MVP)

#### Transactions Table

```sql
CREATE TABLE transactions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    merchant_id TEXT NOT NULL,
    mcc TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    approved_at TEXT NOT NULL,     -- RFC3339 timestamp
    created_at TEXT NOT NULL
)
```

**Indexes:**
- `idx_user_id`: Fast user lookups
- `idx_merchant_id`: Fast merchant filtering
- `idx_mcc`: Fast MCC code filtering
- `idx_approved_at`: Time-based queries
- `idx_user_approved_at`: **Composite index** for eligibility queries (most important)

The composite index `(user_id, approved_at)` is critical for performance, as eligibility queries filter by both user_id and time range.

### Data Serialization

**MCC Whitelist**: Stored as JSON array string (e.g., `["5812","5814"]`). This allows:
- Easy querying and filtering
- Flexible array length
- Human-readable in database tools

**Timestamps**: Stored as RFC3339 strings (e.g., `2025-10-21T10:00:00Z`). Benefits:
- Timezone-aware (always UTC)
- Sortable as strings
- Human-readable
- No precision loss

## Eligibility Calculation

### Algorithm

The eligibility calculation follows these steps:

1. **Get Active Offers**: Query offers where:
   - `active = true`
   - `starts_at <= now`
   - `ends_at >= now`

2. **For Each Active Offer**:
   - Calculate lookback window: `[now - lookback_days, now]`
   - Count matching transactions for the user where:
     - `user_id = target_user_id`
     - `approved_at >= lookback_start`
     - `approved_at <= now`
     - AND (`merchant_id = offer.merchant_id` OR `mcc IN offer.mcc_whitelist`)

3. **Check Eligibility**: If `count >= min_txn_count`, user is eligible

### Query Optimization

The eligibility query uses a composite index on `(user_id, approved_at)`:

```sql
SELECT COUNT(*) FROM transactions
WHERE user_id = ?
AND approved_at >= ?
AND approved_at <= ?
AND (merchant_id = ? OR mcc IN (...))
```

This query:
- Uses the composite index for efficient user_id + time range filtering
- Then filters by merchant_id or MCC (both indexed)
- Returns a count (no data transfer overhead)

### Performance Characteristics

- **Time Complexity**: O(O * T) where O = number of active offers, T = transactions per user
- **Space Complexity**: O(1) - only counts are stored
- **Database Queries**: 
  - 1 query to get active offers
  - N queries to count transactions (one per offer)
  - Could be optimized to 1 query with JOIN, but current approach is clearer and still fast with indexes

## Business Rules Implementation

### Offer Activation

An offer is **ACTIVE** if:
- `active == true` (boolean flag)
- `starts_at <= now <= ends_at` (time window check)

Implemented in `database.GetActiveOffers()` with SQL WHERE clause.

### Transaction Matching

A transaction **matches** an offer if:
- `transaction.merchant_id == offer.merchant_id` OR
- `transaction.mcc IN offer.mcc_whitelist`

Implemented in `database.CountMatchingTransactions()` with SQL OR condition.

### User Eligibility

A user is **eligible** for an ACTIVE offer if:
- Within the last `lookback_days` days (counting back from `now`)
- User has made at least `min_txn_count` matching transactions

Implemented in `service.GetEligibleOffers()` by combining active offer check with transaction count.

## Error Handling Strategy

### Validation Errors

- **Input Validation**: Performed in service layer
- **HTTP Status**: 400 Bad Request
- **Response Format**: `{"error": "descriptive message"}`

### Database Errors

- **Wrapped Errors**: All database errors are wrapped with context
- **HTTP Status**: 500 Internal Server Error (not exposed to users)
- **Logging**: Errors logged at handler level (would add structured logging in production)

### Error Propagation

Errors flow: `Database → Service → Handler → HTTP Response`

Each layer adds context but doesn't expose internal details to users.

## Testing Strategy

### Unit Tests

Located in `internal/service/service_test.go`, covering:

1. **User Qualifies**: User has enough matching transactions
2. **User Doesn't Qualify - Not Enough Transactions**: Below minimum threshold
3. **User Doesn't Qualify - Offer Inactive**: Offer exists but is inactive
4. **User Doesn't Qualify - Out of Time Window**: Transactions outside lookback period
5. **Multiple Offers**: User qualifies for multiple offers simultaneously

### Test Database

Each test:
- Creates a temporary SQLite database file
- Runs test against isolated database
- Cleans up database file after test

This ensures:
- Tests are isolated and don't interfere with each other
- No external dependencies
- Fast execution

## What Would Be Added With More Time

### High Priority

1. **Idempotency**
   - Add idempotency keys to transaction ingestion
   - Prevent duplicate transaction processing
   - Implement: Store idempotency keys with TTL

2. **Pagination**
   - Add pagination to eligible offers endpoint
   - Implement cursor-based pagination for large result sets
   - Add `limit` and `cursor` query parameters

3. **Request Validation**
   - UUID format validation
   - MCC code format validation (4-digit numeric)
   - Timestamp validation and range checks
   - Use a validation library (e.g., `go-playground/validator`)

4. **Structured Logging**
   - Replace `log` package with structured logger (e.g., `zap` or `logrus`)
   - Add request ID to all log entries
   - Log request/response for debugging

### Medium Priority

5. **Metrics & Monitoring**
   - Prometheus metrics for:
     - Request rate and latency
     - Database query performance
     - Error rates
   - Health check endpoint with dependency checks

6. **Connection Pooling**
   - Configure SQLite connection pool size
   - Add connection pool metrics
   - Handle connection exhaustion gracefully

7. **Graceful Shutdown**
   - Wait for in-flight requests to complete
   - Close database connections gracefully
   - Implement timeout for shutdown

8. **Caching**
   - Cache active offers (TTL: 1 minute)
   - Cache eligibility results (TTL: 30 seconds)
   - Use Redis or in-memory cache

### Low Priority

9. **API Versioning**
   - Add `/v1/` prefix to all endpoints
   - Support multiple API versions simultaneously

10. **Rate Limiting**
    - Per-IP rate limiting
    - Per-user rate limiting
    - Use token bucket or sliding window algorithm

11. **Database Migrations**
    - Use migration tool (e.g., `golang-migrate`)
    - Version control schema changes
    - Support rollback

12. **Integration Tests**
    - End-to-end API tests
    - Test with real HTTP requests
    - Use test fixtures for data

13. **Documentation**
    - OpenAPI/Swagger specification
    - Interactive API documentation
    - Code examples in multiple languages

## Scalability Considerations

### Current Limitations

- **Single SQLite File**: Not suitable for distributed systems
- **No Sharding**: All data in one database
- **No Replication**: Single point of failure

### Scaling Path

1. **Read Replicas**: Add read-only SQLite replicas for eligibility queries
2. **PostgreSQL Migration**: Move to PostgreSQL for better concurrency
3. **Caching Layer**: Add Redis for frequently accessed data
4. **Horizontal Scaling**: Use database sharding by user_id
5. **Event Sourcing**: Consider event-driven architecture for transaction ingestion

### Performance Optimizations

- **Batch Eligibility Checks**: Allow checking multiple users at once
- **Precomputed Eligibility**: Background job to precompute eligibility
- **Materialized Views**: Pre-aggregate transaction counts per user/offer

## Security Considerations

### Current Implementation

- **SQL Injection Prevention**: All queries use parameterized statements
- **Input Validation**: Basic validation on all inputs
- **Error Messages**: Generic errors to avoid information leakage

### Production Additions

1. **Authentication**: API keys or OAuth2
2. **Authorization**: Role-based access control
3. **Rate Limiting**: Prevent abuse
4. **Input Sanitization**: More thorough validation
5. **HTTPS**: TLS encryption (handled by reverse proxy)
6. **Audit Logging**: Log all data modifications
7. **Data Encryption**: Encrypt sensitive fields at rest

## Code Quality

### Principles Followed

1. **Clean Code**: Clear naming, small functions, single responsibility
2. **Error Handling**: Explicit error handling, no silent failures
3. **Type Safety**: Strong typing, no `interface{}` abuse
4. **Documentation**: Public functions documented
5. **Testing**: Core business logic thoroughly tested

### Code Organization

- **Package Structure**: Clear separation by layer
- **Dependency Direction**: Dependencies flow inward (Handler → Service → Database)
- **No Circular Dependencies**: Clean dependency graph
- **Internal Packages**: Use `internal/` to prevent external imports

## Conclusion

This API is designed to be:
- **Correct**: Implements all business rules accurately
- **Fast**: Optimized queries with proper indexes
- **Maintainable**: Clean code with clear structure
- **Testable**: Isolated tests with good coverage
- **Extensible**: Easy to add features without major refactoring

The design prioritizes simplicity and correctness over premature optimization, while leaving clear paths for future enhancements.

