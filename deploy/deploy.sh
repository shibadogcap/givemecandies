#!/bin/sh
# zero-downtime deploy: the new build binds the same port via SO_REUSEPORT,
# then systemd restarts onto the same path and the standby leaves.
set -e
BIN=${1:-./candy}
scp -q "$BIN" oracle:/tmp/candy.new
ssh oracle 'set -e
  sudo install -m755 /tmp/candy.new /opt/candy/candy
  sudo systemd-run --unit=candy-standby --collect --uid=ubuntu --working-directory=/opt/candy /opt/candy/candy >/dev/null 2>&1
  sleep 2
  [ "$(sudo ss -lnt | grep -c :8080)" = 2 ] || { echo "standby did not bind"; sudo systemctl stop candy-standby; exit 1; }
  sudo systemctl restart candy
  sleep 1
  sudo systemctl stop candy-standby
  sleep 1
  echo "live: $(systemctl is-active candy)  listeners: $(sudo ss -lnt | grep -c :8080)"'
