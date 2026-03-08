# Integration Tests

Run:

```
INTEGRATION_DB_DSN=postgres://... go test -tags=integration ./internal/integration -v
```

If `INTEGRATION_DB_DSN` is not set, the tests will use `DB_DSN`.

The tests will truncate `user_credentials`, `users`, `tenants`, and `rbac_policies`.
