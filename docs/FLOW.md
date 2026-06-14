flowchart TD
    A[Admin] --> B[nms-agentctl CLI]
    A --> C[nms-agent Service]

    B --> B1[validate config]
    B --> B2[queue status]
    B --> B3[queue retry]
    B --> B4[view / discover]

    B1 --> D[Config Loader]
    B2 --> D
    B3 --> D
    B4 --> D

    C --> D[Config Loader]
    D --> E[agent.yml]
    D --> F[devices.d/*.yml]
    D --> G[adapters.yml]
    D --> H[thresholds.yml]
    D --> I[profiles/*.yml]

    D --> J[Config Validator]
    J -->|valid| K[Pipeline Runtime]
    J -->|invalid| X[Exit with Error]

    subgraph Pipeline[Agent Pipeline]
        K --> L[Collector Selection\n(--collector-mode)]
        L --> L1[Dummy Collector]
        L --> L2[ICMP Collector]
        L --> L3[SNMP Collector]
        L1 --> M[Raw Samples]
        L2 --> M
        L3 --> M
        M --> N[Preprocess + Normalize]
        N --> N1[Derived metrics\nif utilization\nmemory used_pct\nstorage used_pct/bytes]
        N1 --> O[Canonical Telemetry]
        O --> P[Enqueue Telemetry]
        P --> Q[(SQLite Queue DB)]
        Q --> R[PendingBatch]
        R --> S[Active Adapter\nTUI / Generic MQTT / ThingsBoard MQTT]
        S -->|send success| T[MarkDelivered]
        S -->|send failed| U[MarkFailed + retry_count]
        T --> Q
        U --> Q
    end

    subgraph ThingsBoardHybrid[ThingsBoard Hybrid Side Effects]
        S --> V[Relation reconcile]
        S --> W[Topology publish]
        S --> Y[Alarm create / clear]
    end

    S --> Z[Adapter Output]

    J -->|valid| Q

    Q --> B2
    B2 --> AA[Queue Summary]
    AA --> A

    B3 --> AB[PendingBatch]
    AB --> S
