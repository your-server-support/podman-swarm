# Testing Guide

This document describes the testing strategy and how to run tests for Podman Swarm.

## Test Coverage

### Unit Tests

Currently implemented unit tests for:

1. **Storage (`internal/storage`)** - 13 tests
   - State persistence and recovery
   - Deployment, Service, Ingress, Pod operations
   - Concurrent access
   - Atomic writes
   - Backup functionality
   - State merging

2. **Security (`internal/security`)** - 19 tests
   - API token generation and validation
   - Token expiration and cleanup
   - Join token management
   - Token revocation
   - Concurrent token operations

3. **Parser (`internal/parser`)** - 8 tests
   - Kubernetes manifest parsing
   - Deployment parsing
   - Service parsing
   - Ingress parsing
   - Pod template extraction

### Integration Tests

Integration tests require actual cluster setup and are marked as skipped in unit tests:

- **Scheduler** - Requires real cluster
- **API** - Requires Podman and cluster
- **Cluster** - Requires network setup
- **DNS** - Requires network and service discovery
- **Ingress** - Requires network and routing

## Running Tests

### Run All Unit Tests

```bash
make test-unit
```

### Run Specific Package Tests

```bash
go test -v ./internal/storage
go test -v ./internal/security
go test -v ./internal/parser
```

### Run with Coverage

```bash
make test-coverage
```

This generates:
- `coverage.out` - Coverage data
- `coverage.html` - HTML coverage report

View coverage:

```bash
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Run Individual Test

```bash
go test -v ./internal/storage -run TestSaveAndGetDeployment
```

## Test Structure

### Storage Tests

```
internal/storage/storage_test.go
├── Setup/Teardown helpers
│   ├── setupTestStorage() - Creates temp storage
│   └── cleanup() - Cleans up temp directory
└── Test cases
    ├── CRUD operations (Save, Get, Delete, List)
    ├── Persistence across restarts
    ├── Backup functionality
    ├── State merging
    ├── Concurrent access
    └── Atomic writes
```

### Security Tests

```
internal/security/
├── auth_test.go - API token tests
│   ├── Token generation
│   ├── Token validation
│   ├── Token expiration
│   ├── Token revocation
│   └── Concurrent operations
└── token_test.go - Join token tests
    ├── Token generation
    ├── Token validation
    └── Token formatting
```

### Parser Tests

```
internal/parser/parser_test.go
├── Deployment parsing
├── Service parsing
├── Ingress parsing
├── Pod template extraction
└── Edge cases (default namespaces, multiple containers)
```

## Writing Tests

### Test Template

```go
func TestFeatureName(t *testing.T) {
    // Setup
    // ... create test data

    // Execute
    result, err := functionUnderTest(input)

    // Assert
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }

    // Cleanup (if needed)
}
```

### Best Practices

1. **Use table-driven tests** for multiple test cases
2. **Cleanup resources** in defer statements
3. **Use temporary directories** for file operations
4. **Mock external dependencies** (network, Podman API)
5. **Test error conditions** not just happy paths
6. **Use meaningful test names** that describe what's being tested

### Example Table-Driven Test

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "test", false},
        {"empty input", "", true},
        {"invalid input", "!", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Test Data

### Temporary Storage

Tests use temporary directories for storage:

```go
tmpDir, err := os.MkdirTemp("", "storage-test-*")
defer os.RemoveAll(tmpDir)
```

### Mock Data

Tests create minimal valid objects:

```go
deployment := &types.Deployment{
    Name:            "test-deployment",
    Namespace:       "default",
    DesiredReplicas: 3,
}
```

## Continuous Integration

Tests should run in CI/CD pipeline:

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      - run: make test-unit
      - run: make test-coverage
      - uses: codecov/codecov-action@v2
```

## Known Limitations

### Integration Tests

Full integration tests require:
- Running Podman daemon
- Network configuration
- Multiple nodes for cluster testing

These should be run separately:

```bash
# Start test cluster
./scripts/start-test-cluster.sh

# Run integration tests
go test -v -tags integration ./test/integration/...

# Cleanup
./scripts/stop-test-cluster.sh
```

### Platform-Specific Tests

Some features are Linux-only (Podman, network namespaces):

```go
//go:build linux
// +build linux

func TestLinuxFeature(t *testing.T) {
    // ...
}
```

## Troubleshooting

### Tests Fail with "Permission Denied"

Check file permissions on test data directory:

```bash
chmod 755 /tmp/storage-test-*
```

### Tests Hang

Increase test timeout:

```bash
go test -v -timeout 30s ./internal/storage
```

### Coverage Not Generated

Ensure coverage tools are installed:

```bash
go install golang.org/x/tools/cmd/cover@latest
```

## Contributing Tests

When adding new features:

1. **Write tests first** (TDD approach)
2. **Test both success and failure** cases
3. **Document test purpose** with comments
4. **Update this guide** if adding new test patterns

### Test Checklist

- [ ] Unit tests for new functions
- [ ] Edge cases covered
- [ ] Error conditions tested
- [ ] Concurrent access tested (if applicable)
- [ ] Documentation updated
- [ ] CI passes

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Testify Framework](https://github.com/stretchr/testify) (optional)

## Future Improvements

See [TODO.md](TODO.md) for planned testing improvements:

- [ ] End-to-end tests
- [ ] Performance benchmarks
- [ ] Chaos testing
- [ ] Load testing
- [ ] Integration test suite
