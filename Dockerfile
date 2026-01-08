FROM alpine:latest

COPY simple_api_gateway /simple_api_gateway
COPY example_config.toml /config.toml
COPY .qoder/repowiki/zh /.qoder/repowiki/zh

ENTRYPOINT ["/simple_api_gateway", "serve", "/config.toml"]
