#!/usr/bin/env bash
# Adds centralauth.local subdomain entries to /etc/hosts.
# Run once: sudo ./setup-hosts.sh

HOSTS_FILE="/etc/hosts"
MARKER="# centralauth.local (central-auth dev)"

if grep -q "$MARKER" "$HOSTS_FILE"; then
  echo "Entries already present in $HOSTS_FILE — nothing to do."
  exit 0
fi

cat >> "$HOSTS_FILE" <<EOF

$MARKER
127.0.0.1 centralauth.local
127.0.0.1 auth.centralauth.local
127.0.0.1 api.auth.centralauth.local
127.0.0.1 analytics.centralauth.local
127.0.0.1 api.analytics.centralauth.local
127.0.0.1 report.centralauth.local
127.0.0.1 api.report.centralauth.local
127.0.0.1 transaction.centralauth.local
127.0.0.1 api.transaction.centralauth.local
EOF

echo "Done. Entries added to $HOSTS_FILE."
