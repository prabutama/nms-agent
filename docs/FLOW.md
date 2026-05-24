flowchart TD
    A[Admin] --> B[nms-agentctl CLI]
    A --> C[nms-agent Service]

    B --> B1[validate config]
    B --> B2[queue status]
    B --> B3[queue retry]

    B1 --> D[Config Loader]
    B2 --> D
    B3 --> D

    C --> D[Config Loader]
    D --> E[agent.yml]
    D --> F[devices.d/*.yml]
    D --> G[adapters.yml]
    D --> H[thresholds.yml]

    D --> I[Config Validator]
    I -->|valid| J[Pipeline Runtime]
    I -->|invalid| X[Exit with Error]

    subgraph Pipeline[Agent Pipeline]
        J --> K[Collector Selection\n(--collector-mode)]
        K --> K1[Dummy Collector]
        K --> K2[ICMP Collector]
        K --> K3[SNMP Collector]
        K1 --> L[Raw Sample]
        K2 --> L
        K3 --> L
        L --> M[Passthrough Processor]
        M --> N[Canonical Telemetry]
        N --> O[Enqueue Telemetry]
        O --> Q[(SQLite Queue DB)]
        Q --> P[PendingBatch]
        P --> R[Terminal Adapter]
        R -->|send success| S[MarkDelivered]
        R -->|send failed| T[MarkFailed + retry_count]
        S --> Q
        T --> Q
    end

    R --> U[Terminal Output]

    I -->|valid| Q

    Q --> B2
    B2 --> V[Queue Summary]
    V --> A

    B3 --> P2[PendingBatch]
    P2 --> R
    R -->|send success| S2[MarkDelivered]
    R -->|send failed| T2[MarkFailed + retry_count]
    S2 --> Q
    T2 --> Q
