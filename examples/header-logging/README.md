# HTTP Header Logging with Redaction

This example demonstrates how HTTP headers are logged when publishing NATS messages, and how to redact sensitive headers to protect sensitive information in logs.

## Overview

When using `nats_request` or `nats_publish` handlers, all HTTP request headers are automatically logged in the debug logs. This is useful for debugging and monitoring, but can expose sensitive information like API keys, authorization tokens, etc.

The `LOGGER_REDACT_HEADERS` environment variable allows you to specify which headers should be redacted (logged as `***` instead of their actual values).

## Configuration

### Environment Variable

Set `LOGGER_REDACT_HEADERS` to a comma-separated list of header names that should be redacted:

```bash
export LOGGER_REDACT_HEADERS="Authorization,X-API-Key,Custom-Secret-Header"
```

Or set it inline when running Caddy:

```bash
LOGGER_REDACT_HEADERS="Authorization,X-API-Key" ./caddy run --config Caddyfile
```

### Header Name Matching

- **Case-insensitive**: `Authorization`, `authorization`, and `AUTHORIZATION` are all treated the same
- **Exact match**: Header names must match exactly (after case normalization)
- **Multiple headers**: Separate multiple header names with commas
- **Whitespace**: Spaces around commas are automatically trimmed

## How It Works

1. When a request is received, all HTTP headers are captured
2. Headers listed in `LOGGER_REDACT_HEADERS` have their values replaced with `***`
3. The headers (with redacted values) are logged in the debug log when publishing to NATS
4. Headers not in the list are logged with their original values

## Example Log Output

### Without Redaction

When `LOGGER_REDACT_HEADERS` is not set, all headers are logged with their actual values:

```json
{
  "level": "debug",
  "msg": "publishing NATS message",
  "subject": "test.subject.hello",
  "headers": {
    "Authorization": ["Bearer secret-token-12345"],
    "X-Api-Key": ["my-secret-api-key"],
    "Content-Type": ["application/json"],
    "Custom-Header": ["visible-value"]
  }
}
```

### With Redaction

When `LOGGER_REDACT_HEADERS="Authorization,X-API-Key"` is set:

```json
{
  "level": "debug",
  "msg": "publishing NATS message",
  "subject": "test.subject.hello",
  "headers": {
    "Authorization": ["***"],
    "X-Api-Key": ["***"],
    "Content-Type": ["application/json"],
    "Custom-Header": ["visible-value"]
  }
}
```

Notice that:
- `Authorization` and `X-Api-Key` are redacted (shown as `***`)
- `Content-Type` and `Custom-Header` keep their original values (not in the redaction list)

## Testing

### Step 1: Start NATS Server

In one terminal:

```bash
nats-server
```

### Step 2: Create a NATS Responder (for nats_request testing)

In another terminal:

```bash
nats reply 'test.subject.>' --command "echo 'Response for {{1}}'"
```

### Step 3: Start Caddy

#### Without Redaction (Baseline)

```bash
cd examples/header-logging
./caddy run --config Caddyfile
```

#### With Redaction

```bash
cd examples/header-logging
LOGGER_REDACT_HEADERS="Authorization,X-API-Key" ./caddy run --config Caddyfile
```

### Step 4: Make Test Requests

In a third terminal:

```bash
# Test nats_request
curl -H "Authorization: Bearer secret-token-12345" \
     -H "X-API-Key: my-secret-api-key" \
     -H "Content-Type: application/json" \
     -H "Custom-Header: visible-value" \
     http://127.0.0.1:8888/test/hello

# Test nats_publish
curl -H "Authorization: Bearer secret-token-12345" \
     -H "X-API-Key: my-secret-api-key" \
     http://127.0.0.1:8888/publish/test
```

### Step 5: Check the Logs

Look for log entries containing `"publishing NATS message"` in the Caddy output. You should see:

- **Without redaction**: All headers with their actual values
- **With redaction**: Sensitive headers (in the list) shown as `***`, others with original values

## Quick Test Script

You can use the provided `build-run.sh` script:

```bash
cd examples/header-logging
./build-run.sh
```

This will:
1. Build Caddy if needed
2. Start NATS server
3. Start Caddy with instructions

## Common Use Cases

### Redact Authentication Headers

```bash
LOGGER_REDACT_HEADERS="Authorization,X-API-Key,Api-Key" ./caddy run --config Caddyfile
```

### Redact Multiple Custom Headers

```bash
LOGGER_REDACT_HEADERS="Authorization,X-API-Key,X-Secret-Token,X-Auth-Token" ./caddy run --config Caddyfile
```

### Case-Insensitive Matching

These are all equivalent:

```bash
LOGGER_REDACT_HEADERS="Authorization" ./caddy run --config Caddyfile
LOGGER_REDACT_HEADERS="authorization" ./caddy run --config Caddyfile
LOGGER_REDACT_HEADERS="AUTHORIZATION" ./caddy run --config Caddyfile
```

## Important Notes

1. **Debug Logs Only**: Header logging only appears in DEBUG level logs. Make sure your Caddyfile has `log { level DEBUG }` configured.

2. **Log Security**: Even with redaction, be careful about where you store logs. Redacted headers still indicate which headers were present.

3. **Performance**: Header redaction has minimal performance impact as it only processes headers when logging.

4. **Multiple Values**: If a header has multiple values, all values are redacted to `***`.

## Troubleshooting

### Headers Not Appearing in Logs

- Check that log level is set to `DEBUG` in your Caddyfile
- Look for `"publishing NATS message"` in the logs
- Ensure you're making requests to routes that use `nats_request` or `nats_publish`

### Redaction Not Working

- Verify `LOGGER_REDACT_HEADERS` is set correctly (check for typos)
- Remember that matching is case-insensitive
- Check that the header name matches exactly (after case normalization)

### Headers Still Visible

- Make sure the header name is in the `LOGGER_REDACT_HEADERS` list
- Check for extra spaces or typos in the header name
- Verify the environment variable is set before starting Caddy
