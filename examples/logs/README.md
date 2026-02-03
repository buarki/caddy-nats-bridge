
```sh
~/go/bin/xcaddy build --with github.com/sandstorm/caddy-nats-bridge
```

```sh
cp ../../caddy .
```

```sh
cd /Users/buarki/projects/caddy-nats-bridge/examples/logs && ./caddy run --config Caddyfile
```

```sh
nats subscribe 'my.log.>' --count 1
```

```sh
curl http://127.0.0.1:8888
```
