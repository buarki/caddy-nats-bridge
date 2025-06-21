# Timeout Demo

Simple demo showing HTTP timeout configurations in Caddy-NATS-Bridge.

## Run

1. **Start NATS server:**
```bash
nats-server
```

2. **Start the NATS service:**
```bash
go run main.go
```

3. **Start Caddy:**
```bash
xcaddy build --with github.com/buarki/caddy-nats-bridge=./
./caddy run --config Caddyfile
```

4. **Test endpoints:**
```bash
curl http://127.0.0.1:8888/api/fast/test -i -k -L
# Expected: 200 in ~1s

curl http://127.0.0.1:8888/api/slow/test -i -k -L
# Expected: 504 in ~2s (timeout set to 2s, service takes 3s)

curl http://127.0.0.1:8888/api/very-slow/test -i -k -L
# Expected: 504 in ~7s (default timeout, service takes 8s)

curl http://127.0.0.1:8888/api/custom/test -i -k -L
# Expected: 200 in ~9s (timeout set to 9s, service takes 9s)

