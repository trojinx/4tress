# 4tress

4tress is a host-based HTTP reverse proxy written in Go. It routes an incoming
request to an allow-listed upstream based on its `Host` header and, optionally,
an exact host/path route.

## Run

```powershell
go run .
```

The proxy reads its two JSON files from the current working directory:

- `4tress.config.json` configures the listener and optional rate limiter.
- `4tress.records.json` contains route keys and their upstream HTTP(S) URLs.

`GET /health` is handled by the proxy itself and returns `200 OK`.

## Routing

The most specific exact `host/path` key wins; otherwise the host-only key is
used. The incoming path and query string are then forwarded to the selected
upstream. For example, with the supplied configuration:

```text
abc.com       -> http://127.0.0.1:7000
abc.com/tt    -> http://127.0.0.1:7079
```

`http://abc.com/products?page=2` goes to the first upstream, while
`http://abc.com/tt?page=2` goes to the second. Both preserve the request path
and query string. Upstream URLs must use `http://` or `https://` and include a
host.

## Rate limiting

Set `EnableRateLimiting` to `true`. `RateLimitAfter` is the number of requests
allowed from one client IP in each `RateLimitWindowSeconds` window. Exceeded
requests receive `429 Too Many Requests` and a `Retry-After` header.

## Operations and security

4tress uses Go's standard reverse proxy, which correctly handles normal HTTP
streaming and upgrade requests such as WebSockets. It strips hop-by-hop headers,
preserves the requested virtual host, and sets trusted `X-Forwarded-*` headers.

It is not a TLS terminator or a web application firewall. Put it behind an
HTTPS/TLS endpoint and use a dedicated WAF when those controls are required.
The former unused `EnableWebFireWall` setting has been removed rather than
pretending that it enabled protection.

The non-JSON `4tress.config` and `4tress.settings` files are historical notes;
they are not read by the program.
