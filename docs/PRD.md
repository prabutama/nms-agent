# PRD — Platform-Agnostic NMS Agent

## 1. Overview

The NMS Agent is a Go-based local monitoring agent that runs on gateways at each site. The agent collects network device data using SNMP and ICMP, pre-processes it, normalizes it into a neutral internal format, temporarily stores the data when the connection is interrupted, and then sends the data to the consumer platform via an adapter.

The agent is platform-agnostic, so the core agent is not dependent on ThingsBoard, Zabbix, Prometheus, or any specific platform.

---

## 2. Goals

1. Build a Go-based monitoring agent as a single binary.
2. Run the agent as a service using systemd.
3. Collect device data using SNMP and ICMP.
4. Support multi-vendor devices through device profiles.
5. Generate canonical telemetry format as an internal data format.
6. Store data in a local queue for store-and-forward.
7. Send data through an adapter to various consumer platforms.
8. Provides a CLI for device configuration, threshold, queue, status, and reloading.
9. Supports hot reload configuration without a full service restart.

---

## 3. Problem Statement

In previous testing, when the gateway connection to the monitoring server was lost for 10 minutes with a 1-minute delivery interval, the expected data was 10 records. However, only 4 records were successfully returned after the connection was restored.

This issue indicates that the buffer mechanism needs to be placed closer to the data source. Therefore, store-and-forward will be implemented at the agent level so that the collected and preprocessed data is stored before being sent to the consumer platform.

---

## 4. Core Architecture

```text
SNMP / ICMP Collector
↓
Device Profile / Vendor Profile
↓
Preprocessing Engine
↓
Canonical Telemetry Format
↓
SQLite Local Queue
↓
Output Adapter
↓
Consumer / Terminal Platform