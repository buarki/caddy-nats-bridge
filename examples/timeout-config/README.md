# NATS Request Timeout Configuration

This example demonstrates how to configure NATS request timeouts using environment variables and Caddyfile directives.

## Configuration Options

### 1. Environment Variable (Global Default)

Set a global default timeout for all `nats_request` handlers:

```bash
export NATS_REQUEST_DEFAULT_TIMEOUT=30s
./caddy run --config Caddyfile
```

Or set it inline:

```bash
NATS_REQUEST_DEFAULT_TIMEOUT=30s ./caddy run --config Caddyfile
```

### 2. Caddyfile Directive (Per-Handler Override)

Override the environment variable timeout for specific handlers:

```nginx
route /api/data/* {
    nats_request cli.data.{http.request.uri.path.1} {
        timeout 10s  # Override environment variable
    }
}
```

### 3. Default Fallback

If no environment variable is set, the timeout defaults to **60 seconds**.

## Supported Duration Formats

The timeout accepts standard Go duration formats:

- `30s` - 30 seconds
- `2m` - 2 minutes  
- `1h` - 1 hour
- `100ms` - 100 milliseconds
- `1.5m` - 1 minute 30 seconds

## Priority Order

1. **Caddyfile timeout directive** (highest priority)
2. **Environment variable** (`NATS_REQUEST_DEFAULT_TIMEOUT`)
3. **Hardcoded default** (60 seconds)

## Testing the Configuration

1. Start NATS server:
   ```bash
   nats-server
   ```

2. Start a test NATS responder:
   ```bash
   nats reply 'cli.weather.>' --command "sleep 5 && echo 'Weather data for {{2}}'"
   ```

3. Run Caddy with custom timeout:
   ```bash
   NATS_REQUEST_DEFAULT_TIMEOUT=10s ./caddy run --config Caddyfile
   ```

4. Test the endpoints:
   ```bash
   # This should work (10s timeout > 5s response)
   curl http://127.0.0.1:8888/api/weather/Dresden
   
   # This should timeout (2s timeout < 5s response)
   NATS_REQUEST_DEFAULT_TIMEOUT=2s ./caddy run --config Caddyfile &
   curl http://127.0.0.1:8888/api/weather/Dresden
   ```

## Logging

When the environment variable is used, you'll see a log message like:

```
INFO    using NATS request timeout from environment variable    {"timeout": "30s", "env_var": "NATS_REQUEST_DEFAULT_TIMEOUT"}
``` 