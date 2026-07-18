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

It also installs a sample environment file:

- `/etc/nms-agent/nms-agent.env`

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
- `/etc/nms-agent/nms-agent.env`

Agent state is stored under:

- `/var/lib/nms-agent/nms-agent.db`

## Reload Behavior

`systemctl reload nms-agent` sends `SIGHUP` to the running agent process.
The agent will reload config and rebuild its runtime pipeline without restarting the process.

## Environment Variables

The binary does not auto-load `.env` files. It expands `${ENV_VAR}` from the process environment.

For systemd deployment, use `/etc/nms-agent/nms-agent.env` together with the unit's `EnvironmentFile`.

Example:

```bash
TB_URL=https://nms.prabutama.my.id
TB_API_KEY=replace-with-site-api-key
```
