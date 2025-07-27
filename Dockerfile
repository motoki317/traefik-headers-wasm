# Working version notes:
#
# >=0.35.0 -> http-wasm does not support newer versions
# https://github.com/traefik/traefik/issues/11916
FROM tinygo/tinygo:0.34.0 AS build

WORKDIR /work

COPY . .
RUN make build-debug

FROM traefik

COPY . /plugins-local/src/github.com/motoki317/traefik-headers-wasm/
COPY --from=build /work/plugin.wasm /plugins-local/src/github.com/motoki317/traefik-headers-wasm/
