#!/bin/sh
set -e
if ! id gobookmarks >/dev/null 2>&1; then
    useradd --system --home /nonexistent --shell /usr/sbin/nologin gobookmarks
fi
chown gobookmarks:gobookmarks /etc/gobookmarks/config.json || true
chmod 600 /etc/gobookmarks/config.json || true
if [ -f /etc/gobookmarks/gobookmarks.env ]; then
    chown gobookmarks:gobookmarks /etc/gobookmarks/gobookmarks.env || true
    chmod 600 /etc/gobookmarks/gobookmarks.env || true
fi
