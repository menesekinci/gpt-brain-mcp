# Testing Guide

## Unit Tests

Run all tests:

```bash
go test ./...
```

### Filesystem Security Tests

```bash
go test ./internal/fsx/...
```

Tests cover:
- Path traversal rejection (`../`, absolute escapes)
- Symlink escape prevention
- Sensitive file blocking
- Write directory enforcement

### Project Detection Tests

```bash
go test ./internal/project/...
```

Tests cover:
- Stack detection (Next.js, Go, Python, Rust)
- Project signal recognition
- Ignore rule application

## Integration Tests

Start the server and test via HTTP:

```bash
go build ./cmd/server
./server --addr 127.0.0.1:3939 &
```

### Health Check

```bash
curl http://127.0.0.1:3939/healthz
```

### MCP Initialize

```bash
curl -X POST http://127.0.0.1:3939/mcp/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {"name": "test", "version": "1.0"}
    }
  }'
```

### List Tools

```bash
curl -X POST http://127.0.0.1:3939/mcp/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {}
  }'
```

### Call list_roots

```bash
curl -X POST http://127.0.0.1:3939/mcp/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "list_roots",
      "arguments": {}
    }
  }'
```

## Security Test Cases

### Prompt Injection: Path Traversal

```bash
curl -X POST http://127.0.0.1:3939/mcp/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
      "name": "read_project_file",
      "arguments": {
        "project_id": "default:test-app",
        "path": "../../.ssh/id_rsa"
      }
    }
  }'
```

**Expected:** `Read blocked: blocked: path traversal`

### Prompt Injection: Sensitive File

```bash
curl -X POST http://127.0.0.1:3939/mcp/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 5,
    "method": "tools/call",
    "params": {
      "name": "read_project_file",
      "arguments": {
        "project_id": "default:test-app",
        "path": ".env"
      }
    }
  }'
```

**Expected:** `Read blocked: blocked: sensitive file blocked`

### Write Escape

Since write tools do not accept arbitrary paths, the only attack vector is content injection. The server still enforces:
- Output path is always under `.chatgpt/`
- Content size limits apply
- Secret redaction is applied

## Test Data

Create a test project for integration testing:

```bash
mkdir -p ~/Projects/test-app
cd ~/Projects/test-app
echo '{"name":"test-app","dependencies":{"react":"^18"}}' > package.json
echo '# Test' > README.md
mkdir -p src
echo 'console.log("hello")' > src/index.ts
mkdir -p .chatgpt
echo '{"indexed":true}' > .chatgpt/index.json
```

## Continuous Integration

A minimal CI pipeline should:

1. `go build ./cmd/server`
2. `go test ./...`
3. Start server, run integration curl tests
4. Verify security test cases return expected errors
