#!/bin/sh
set -eu

mkdir -p /data
chown -R ddash:ddash /data /app

exec su-exec ddash /usr/local/bin/ddash
