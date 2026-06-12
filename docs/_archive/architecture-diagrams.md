> Superseded by `architecture-flows.html/.json` + `flows-explainer.html` (repo root, freshness-registered) on 2026-06-12. Archived per DOC_STANDARDS §4 — hand-drawn master-flow predates per-agent notebooks, ACP backends, and the act_cli whitelist.

# ACT Architecture Diagrams

## Master Flow — All Concepts Flattened

Every node here represents one diagram in the full diagram set.
Block nesting shows conceptual containment. Edges show the live workflow.
The source node is the User. The end product is code in a local filesystem.
The filesystem loops back to Tier 1 — ACT observes and iterates on its own output.

```mermaid
flowchart TD
    classDef sys    fill:#0d1b2e,stroke:#4a9eff,stroke-width:2px,color:#e0e0e0,font-weight:bold
    classDef tier1  fill:#0d2137,stroke:#4a9eff,color:#e0e0e0
    classDef tier2  fill:#2e1a0d,stroke:#ff884a,color:#e0e0e0
    classDef srv    fill:#0d2e1a,stroke:#4aff88,color:#e0e0e0
    classDef back   fill:#1a0d37,stroke:#9b4aff,color:#e0e0e0
    classDef ext    fill:#1a2e1a,stroke:#4aff88,stroke-width:2px,color:#e0e0e0

    User(["👤 User"])

    ACT["ACT System"]

    NESTTY["NesTTY · Orchestrator"]
    CONV["Planner Conversation Loop"]
    Intake["INTAKE Mode"]
    Build["BUILD Mode"]
    SPIL["SPIL Format"]
    BG["Background Loops"]
    ObsLoop["Observer Loop"]
    ValPoll["Validation Poll"]
    QAPoll["QA Poll"]
    AutoRoute["autoRoutePlanner"]

    SERVER["ACT Server & State"]
    RESTAPI["REST API"]
    MemState["In-Memory State"]
    ChronLog["ChronLog"]
    PVM["PVM Vector Store"]
    FileLock["File Locking"]

    TIER2["Tier 2 Swarm"]
    RUNNER_B["Runner"]
    RunnerSpawn["Runner Spawner"]
    ExecLoop["Agent Execution Loop"]
    VALID_B["Validation Pipeline"]
    ScoreCrit["@success_criteria Scoring"]
    GapRetry["Gap Analysis + Retry"]


    BACKENDS["Agent Backends"]
    APIBack["API Backend"]
    ACPB["ACP CLI Backend"]
    ACPProto["ACP Protocol Flow"]
    PromptInj["Prompt Injection Rules"]

    LocalFS[("📁 Local Filesystem")]

    User --> ACT
    ACT --> NESTTY
    ACT --> SERVER
    ACT --> TIER2
    ACT --> BACKENDS

    NESTTY --> CONV
    NESTTY --> BG
    CONV --> Intake
    CONV --> Build
    CONV --> SPIL
    BG --> ObsLoop
    BG --> ValPoll
    BG --> QAPoll
    BG --> AutoRoute
    AutoRoute -.-> CONV

    CONV --> SERVER
    BG --> SERVER
    SERVER --> RESTAPI
    SERVER --> MemState
    SERVER --> ChronLog
    SERVER --> PVM
    SERVER --> FileLock
    SERVER --> TIER2

    TIER2 --> RUNNER_B
    RUNNER_B --> RunnerSpawn
    RUNNER_B --> ExecLoop
    ExecLoop --> ValPoll
    ValPoll --> VALID_B
    VALID_B --> ScoreCrit
    VALID_B --> GapRetry
    GapRetry --> QAPoll
    QAPoll --> LocalFS

    NESTTY --> BACKENDS
    BACKENDS --> APIBack
    BACKENDS --> ACPB
    ACPB --> ACPProto
    ACPB --> PromptInj

    class ACT sys
    class NESTTY,CONV,Intake,Build,SPIL,BG,ObsLoop,ValPoll,QAPoll,AutoRoute,VALID_B,ScoreCrit,GapRetry tier1
    class TIER2,RUNNER_B,RunnerSpawn,ExecLoop tier2
    class SERVER,RESTAPI,MemState,ChronLog,PVM,FileLock srv
    class BACKENDS,APIBack,ACPB,ACPProto,PromptInj back
    class LocalFS ext
```

---

*Detailed diagrams for each node above — to be built.*
