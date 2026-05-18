#!/usr/bin/env sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
group=${1:-staff}

if ! id -Gn "$USER" | tr ' ' '\n' | grep -qx "$group"; then
  echo "warning: current user '$USER' is not in group '$group'" >&2
fi

chgrp -R "$group" "$repo_root"
chmod -R g+rwX "$repo_root"
find "$repo_root" -type d -exec chmod g+s {} +

echo "shared permissions fixed for $repo_root using group '$group'"
