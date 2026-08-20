#!/usr/bin/env sh
# Build and copy the binary into INSTALL_DIR (default ~/.local/bin), seeding the
# settings there from target/config.yaml on first install.
#
# INSTALL_DIR=<dir>  install somewhere else
# FORCE_CONFIG=1     overwrite the already installed config.yaml
set -o errexit -o nounset

. "$(dirname -- "$0")/common.sh"

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
FORCE_CONFIG="${FORCE_CONFIG:-0}"

"$ROOT_DIR/bin/build.sh"

mkdir -p "$INSTALL_DIR"

# Earlier versions of this script symlinked target/ as "<app>-distr".
if [ -L "$INSTALL_DIR/$APP-distr" ]; then
	rm -f "$INSTALL_DIR/$APP-distr"
	echo "Removed stale symlink $INSTALL_DIR/$APP-distr"
fi

# Copy to a temp file and rename, so upgrading works even while the installed
# binary is running (a plain cp over a busy executable fails with ETXTBSY).
tmp="$INSTALL_DIR/.$APP.new.$$"
trap 'rm -f "$tmp"' EXIT HUP INT TERM
cp "$BINARY" "$tmp"
chmod 755 "$tmp"
mv -f "$tmp" "$INSTALL_DIR/$APP"
trap - EXIT HUP INT TERM

echo "Installed $INSTALL_DIR/$APP"

# The app reads config.yaml from the directory holding the executable, so the
# installed copy needs its own next to the binary.
local_config="$TARGET_DIR/config.yaml"
installed_config="$INSTALL_DIR/config.yaml"

if [ ! -f "$local_config" ]; then
	echo "No $local_config to copy; $installed_config is written from the template on first run."
elif [ ! -f "$installed_config" ] || [ "$FORCE_CONFIG" = 1 ]; then
	cp "$local_config" "$installed_config"
	echo "Copied settings to $installed_config"
elif cmp -s "$local_config" "$installed_config"; then
	echo "Settings already match: $installed_config"
else
	echo "Kept $installed_config, it differs from $local_config"
	echo "Overwrite it with: FORCE_CONFIG=1 bin/install.sh"
fi

case ":$PATH:" in
*":$INSTALL_DIR:"*) ;;
*) echo "warning: $INSTALL_DIR is not in PATH" >&2 ;;
esac
