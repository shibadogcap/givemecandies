#!/bin/sh
# snapshot.sh save [tag]     -> /var/lib/candy-snapshots/<tag>.tgz
# snapshot.sh restore <tag>  -> put the files back and restart
# restore never deletes the snapshot, so a bad restore can be undone.
set -e
case "$1" in
  save)
    TAG=${2:-$(date +%Y%m%d-%H%M%S)}
    ssh oracle "sudo tar -C /var/lib/candy -czf /var/lib/candy-snapshots/$TAG.tgz . && sudo chown ubuntu:ubuntu /var/lib/candy-snapshots/$TAG.tgz && echo saved:$TAG"
    ;;
  restore)
    [ -n "$2" ] || { echo "usage: $0 restore <tag>"; exit 1; }
    ssh oracle "set -e
      sudo systemctl stop candy
      sudo tar -C /var/lib/candy -xzf /var/lib/candy-snapshots/$2.tgz
      sudo systemctl start candy
      sleep 1.5
      curl -sS localhost:8080/api/stats"
    ;;
  *)
    echo "usage: $0 save [tag] | restore <tag>"; exit 1;;
esac
