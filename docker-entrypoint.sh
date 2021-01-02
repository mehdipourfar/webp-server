#!/usr/bin/env bash

set -e

if [ -z "$TOKEN" ]; then
    echo "TOKEN env variable must be defined."
    exit 1
else
    export WEBP_SERVER_TOKEN="$TOKEN";
fi

set -- webp-server -config /var/lib/webp-server/config.yml
exec "$@"
