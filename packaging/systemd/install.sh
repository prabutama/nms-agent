#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

OPT_DIR="/opt/nms-agent"
ETC_DIR="/etc/nms-agent"
LIB_DIR="/var/lib/nms-agent"
LOG_DIR="/var/log/nms-agent"
UNIT_PATH="/etc/systemd/system/nms-agent.service"

if [[ "${EUID}" -ne 0 ]]; then
  echo "error: must run as root" >&2
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "error: systemctl not found (systemd is required)" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "error: go toolchain not found (install Go or use a prebuilt binary)" >&2
  exit 1
fi

if ! id -u nms-agent >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin nms-agent
fi

mkdir -p "${OPT_DIR}" "${ETC_DIR}" "${ETC_DIR}/devices.d" "${LIB_DIR}" "${LOG_DIR}"

# Build binaries into /opt.
(
  cd "${REPO_ROOT}"
  go build -o "${OPT_DIR}/nms-agent" ./cmd/nms-agent
  go build -o "${OPT_DIR}/nms-agentctl" ./cmd/nms-agentctl
)

install -m 0644 "${SCRIPT_DIR}/nms-agent.service" "${UNIT_PATH}"
install -m 0644 "${SCRIPT_DIR}/agent.yml" "${ETC_DIR}/agent.yml"
install -m 0644 "${SCRIPT_DIR}/adapters.yml" "${ETC_DIR}/adapters.yml"
install -m 0644 "${SCRIPT_DIR}/thresholds.yml" "${ETC_DIR}/thresholds.yml"

# Install profiles directory.
PROFILES_DIR="${ETC_DIR}/profiles"
mkdir -p "${PROFILES_DIR}"
for f in "${REPO_ROOT}/profiles/"*.yml; do
  [ -f "$f" ] && install -m 0644 "$f" "${PROFILES_DIR}/"
done

# Install example device if none exist.
if ! ls -1 "${ETC_DIR}/devices.d"/*.yml >/dev/null 2>&1; then
  install -m 0644 "${SCRIPT_DIR}/devices.d/example-linux-proxmox.yml" "${ETC_DIR}/devices.d/example-linux-proxmox.yml"
fi

chown -R nms-agent:nms-agent "${LIB_DIR}" "${LOG_DIR}"

systemctl daemon-reload
systemctl enable --now nms-agent

echo "installed: ${OPT_DIR}/nms-agent"
echo "installed: ${OPT_DIR}/nms-agentctl"
echo "config:    ${ETC_DIR}/agent.yml"
echo "service:   nms-agent (systemctl status nms-agent)"
