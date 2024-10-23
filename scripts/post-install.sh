#! /bin/sh

mkdir -p /etc/simple_api_gateway

mkdir -p /var/log/simple_api_gateway

/usr/bin/simple_api_gateway gen /etc/simple_api_gateway/config.toml
