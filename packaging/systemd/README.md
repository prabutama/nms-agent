# systemd Packaging (Phase 10)

This directory contains a minimal systemd unit and install script for running `nms-agent` as a Linux service.

## Install (builds from repo)

Run as root on a systemd-based Linux host (Ubuntu/Debian):

```bash
sudo ./packaging/systemd/install.sh
```

This will:

- create user `nms-agent` (system user)
- build `nms-agent` and `nms-agentctl` into `/opt/nms-agent/`
- install config into `/etc/nms-agent/`
- install unit into `/etc/systemd/system/nms-agent.service`
- enable and start the service

## Service Commands

```bash
sudo systemctl status nms-agent
sudo systemctl restart nms-agent
sudo systemctl reload nms-agent
sudo systemctl stop nms-agent
```

## Logs (journald)

```bash
journalctl -u nms-agent -n 200 --no-pager
journalctl -u nms-agent -f
```

## Config Layout

- `/etc/nms-agent/agent.yml`
- `/etc/nms-agent/adapters.yml`
- `/etc/nms-agent/thresholds.yml`
- `/etc/nms-agent/devices.d/*.yml`

Queue data is stored under:

- `/var/lib/nms-agent/queue.db`

## Reload Behavior

`systemctl reload nms-agent` sends `SIGHUP` to the running agent process.
The agent will reload config and rebuild its runtime pipeline without restarting the process.
