#!/usr/bin/env sh
set -o errexit -o nounset

. "$(dirname -- "$0")/common.sh"

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

if [ -e "$INSTALL_DIR/$APP" ]; then
	rm -f "$INSTALL_DIR/$APP"
	echo "Removed $INSTALL_DIR/$APP"
else
	echo "Nothing to remove at $INSTALL_DIR/$APP"
fi

echo "Left $INSTALL_DIR/config.yaml, datacenters.yaml and the keychain entry alone."
echo "Run '$APP --logout' before uninstalling to drop the stored credentials."
