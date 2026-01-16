# Finance Buddy Backend

## Testing

### Test Database Fixtures

Two database fixtures available:

- **`test_db_session`** - SQLite in-memory database (fast, no external dependencies)
- **`test_db_session_postgres`** - PostgreSQL via testcontainers (requires Docker)

### When to Use PostgreSQL Tests

Use `test_db_session_postgres` to test PostgreSQL-specific behavior:
- Database-specific features (e.g., JSONB, array types)
- PostgreSQL-specific constraints or behaviors
- Integration testing with production-like environment

For most tests, use `test_db_session` (SQLite) - faster and no Docker required.

### Requirements

PostgreSQL tests require:
- Docker daemon running
- `testcontainers[postgres]` installed (included in dev dependencies)

### Running Tests

```bash
# All tests (SQLite + PostgreSQL)
pytest

# SQLite tests only (faster, no Docker required)
pytest -k "not postgres"

# PostgreSQL tests only
pytest -k postgres
```

### Example

See `tests/test_models_account_postgres.py` for PostgreSQL test example.
