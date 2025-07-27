# Traefik Headers Manipulation WASM Plugin

A Traefik middleware plugin that manipulates HTTP request and response headers using WebAssembly.

## Features

- Path-based or header-based manipulation using regex matching
- Dynamic header value generation with capture group substitution
- Support for both request and response headers
- WASM-based implementation for security and portability

## Installation

### From Plugin Catalog

```yaml
# Static configuration
experimental:
  plugins:
    headers:
      moduleName: github.com/motoki317/traefik-headers-wasm
      version: v0.1.0
```

### Local Development

```shell
make dev
```

## Configuration

```yaml
# Dynamic configuration
http:
  middlewares:
    my-headers:
      plugin:
        headers:
          manipulations:
            - matchPath: "^/api/v([0-9]+)/(.*)$"
              customRequestHeaders:
                - name: "X-API-Version"
                  value: "$1"
                - name: "X-Resource"
                  value: "$2"
              customResponseHeaders:
                - name: "X-Processed-By"
                  value: "api-v$1"
```

## Options

- `manipulations`: Array of manipulation rules
    - `matchPath`: Regex pattern to match against the request path (mutually exclusive with `matchRequestHeader`)
    - `matchRequestHeader`: Match against a request header value (mutually exclusive with `matchPath`)
        - `name`: Header name to match against
        - `value`: Regex pattern to match the header value
    - `customRequestHeaders`: Array of headers to add/modify on requests
        - `name`: Header name
        - `value`: Header value (supports capture group substitution with $1, $2, etc.)
        - `replace`: Whether to replace existing header or add to it (default: false)
    - `customResponseHeaders`: Array of headers to add/modify on responses
        - Same fields as `customRequestHeaders`

## Example

```yaml
http:
  routers:
    my-router:
      rule: Host(`example.com`)
      middlewares:
        - my-headers
      service: my-service

  middlewares:
    my-headers:
      plugin:
        headers:
          manipulations:
            # Path-based matching
            - matchPath: "^/test/([^/]+)/([^/]+)$"
              customRequestHeaders:
                - name: "X-Request-Test"
                  value: "first=$1,second=$2"
                  replace: true
              customResponseHeaders:
                - name: "X-Response-Test"
                  value: "second=$2"
                  replace: true
            
            # Header-based matching
            - matchRequestHeader:
                name: "X-My-Header"
                value: "test-([^-]+)-([^-]+)"
              customRequestHeaders:
                - name: "X-Header-Match"
                  value: "matched-$1-$2"
                  replace: true
              customResponseHeaders:
                - name: "X-Header-Response"
                  value: "from-header-$2"
                  replace: true

  services:
    my-service:
      loadBalancer:
        servers:
          - url: http://localhost:8080
```

## Development

```bash
make dev
```

## License

MIT
