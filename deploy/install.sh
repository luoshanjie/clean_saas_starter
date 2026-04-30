#!/bin/sh
set -eu

APP_NAME="${APP_NAME:-service}"
APP_USER="${APP_USER:-service}"
APP_GROUP="${APP_GROUP:-service}"
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
if [ "${SRC_DIR:-}" ]; then
  SRC_DIR="$(CDPATH= cd -- "$SRC_DIR" && pwd)"
elif [ -f "$SCRIPT_DIR/$APP_NAME" ]; then
  SRC_DIR="$SCRIPT_DIR"
else
  SRC_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
fi
INSTALL_DIR="${INSTALL_DIR:-/opt/$APP_NAME}"
CONFIG_DIR="${CONFIG_DIR:-/etc/$APP_NAME}"
LOG_DIR="${LOG_DIR:-/var/log/$APP_NAME}"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root" >&2
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemctl not found; this installer requires systemd" >&2
  exit 1
fi

if ! getent group "$APP_GROUP" >/dev/null 2>&1; then
  groupadd --system "$APP_GROUP"
fi

if ! id "$APP_USER" >/dev/null 2>&1; then
  useradd --system --gid "$APP_GROUP" --home-dir "$INSTALL_DIR" --shell /sbin/nologin "$APP_USER"
fi

install -d -o "$APP_USER" -g "$APP_GROUP" "$INSTALL_DIR" "$LOG_DIR"
install -d "$CONFIG_DIR"
if [ -f "$SRC_DIR/build/$APP_NAME" ]; then
  BIN_SRC="$SRC_DIR/build/$APP_NAME"
else
  BIN_SRC="$SRC_DIR/$APP_NAME"
fi
install -m 0755 "$BIN_SRC" "$INSTALL_DIR/$APP_NAME"
install -d "$INSTALL_DIR/internal/repo/casbin"
install -m 0644 "$SRC_DIR/internal/repo/casbin/model.conf" "$INSTALL_DIR/internal/repo/casbin/model.conf"
install -m 0644 "$SRC_DIR/internal/repo/casbin/policy.csv" "$INSTALL_DIR/internal/repo/casbin/policy.csv"

if [ ! -f "$CONFIG_DIR/app.yaml" ]; then
  install -m 0640 "$SRC_DIR/app.yaml.example" "$CONFIG_DIR/app.yaml"
fi

if [ ! -f "$CONFIG_DIR/$APP_NAME.env" ]; then
  sed \
    -e "s#/etc/service#/etc/$APP_NAME#g" \
    -e "s#/var/log/service#/var/log/$APP_NAME#g" \
    "$SRC_DIR/deploy/systemd/service.env.example" > "$CONFIG_DIR/$APP_NAME.env"
  chmod 0640 "$CONFIG_DIR/$APP_NAME.env"
fi

sed \
  -e "s#User=service#User=$APP_USER#g" \
  -e "s#Group=service#Group=$APP_GROUP#g" \
  -e "s#/opt/service#/opt/$APP_NAME#g" \
  -e "s#/etc/service/service.env#/etc/$APP_NAME/$APP_NAME.env#g" \
  -e "s#ExecStart=/opt/$APP_NAME/service#ExecStart=/opt/$APP_NAME/$APP_NAME#g" \
  "$SRC_DIR/deploy/systemd/service.service" > "$SYSTEMD_DIR/$APP_NAME.service"
chmod 0644 "$SYSTEMD_DIR/$APP_NAME.service"
systemctl daemon-reload
systemctl enable "$APP_NAME.service"

echo "installed $APP_NAME"
echo "edit $CONFIG_DIR/app.yaml and $CONFIG_DIR/$APP_NAME.env, then run: systemctl start $APP_NAME"
