# Design: Dual Stack Migration (Vendoring Strategy)

**Status:** Approved
**Goal:** Migrate `api-linter` to `protoreflect` (V2) by running parallel V1 (Legacy) and V2 (Modern) pipelines, gradually moving rules from one to the other.

## 1. Strategic Rationale
Instead of refactoring the existing codebase with complex adapters (Hybrid Approach), we will leverage the fact that the upstream `google/api-linter` repository has already completed this migration in commit `42e6805`.

**Key Benefit:** We can copy-paste production-proven V2 code (`lint/`, `internal/`, `rules/`) directly into our repository under `v2/` namespaces. This eliminates the risk of introducing subtle bugs during manual refactoring and drastically reduces cognitive load.

**Trade-off:** This approach requires parsing the input proto files twice (once for V1, once for V2) until the migration is complete. We accept this runtime performance cost in exchange for engineering velocity and correctness.

## 2. Architecture

The CLI acts as the coordinator, managing two completely isolated linter stacks.

```mermaid
graph TD
    UserConfig --> CLI
    
    subgraph "V1 Pipeline (Legacy - Frozen)"
        CLI -->|Parse (jhump)| AST_V1[desc.FileDescriptor]
        AST_V1 --> LinterV1
        LinterV1 --> RulesV1[Legacy Rules]
    end
    
    subgraph "V2 Pipeline (Modern - Active)"
        CLI -->|Parse (protocompile)| AST_V2[protoreflect.FileDescriptor]
        AST_V2 --> LinterV2
        LinterV2 --> RulesV2[Modern Rules]
    end
    
    LinterV1 -->|[]Problem| Merger
    LinterV2 -->|[]Problem| Merger
    Merger --> Output
```

## 3. Directory Structure

We will introduce `v2` packages to house the ported code.

```text
api-linter/
├── cmd/
│   └── api-linter/
│       └── cli.go        <-- UPDATED: Orchestrates both pipelines
├── lint/                 <-- V1 Core (Legacy)
├── lint/v2/              <-- V2 Core (Copied from upstream)
├── internal/             <-- V1 Utils (Legacy)
├── internal/v2/          <-- V2 Utils (Copied from upstream)
├── rules/                <-- V1 Rules (Legacy)
└── rules/v2/             <-- V2 Rules (Copied from upstream)
```

## 4. Execution Flow (CLI)

1.  **Load Configuration:** Determine enabled/disabled rules.
2.  **Phase 1 (V1):**
    *   Parse files using `jhump/protoparse`.
    *   Run `lint.New(...)`.
    *   Collect `problemsV1`.
    *   *Optimization:* Explicitly release V1 AST memory if possible (set to nil).
3.  **Phase 2 (V2):**
    *   Parse files using `bufbuild/protocompile` (logic copied from upstream `cli.go`).
    *   Run `lint_v2.New(...)`.
    *   Collect `problemsV2`.
4.  **Merge:**
    *   Combine `problemsV1` and `problemsV2`.
    *   Sort by file and location.
    *   Output results.

## 5. Migration Workflow (Per Rule)

To migrate a specific rule (e.g., `aep0122`):

1.  **Copy:** Copy `rules/aep0122` from `original-api-linter` to `rules/v2/aep0122`.
2.  **Disable V1:** Delete `rules/aep0122` (or disable it in the V1 registry) to prevent duplicate error reporting.
3.  **Register V2:** Add the new V2 rule package to the V2 registry in `rules/v2/rules.go`.

## 6. Comparison with Hybrid Approach

| Feature | Hybrid (Adapter) | Dual Stack (Vendoring) |
| :--- | :--- | :--- |
| **Code Changes** | High (Rewrite rules & utils) | **Low** (Copy-paste upstream) |
| **Risk** | High (Logic bugs) | **Low** (Proven code) |
| **Runtime Perf** | Fast (Single Parse) | Slower (Double Parse) |
| **Complexity** | High (Adapters, Generics) | Low (Two simple loops) |

This design prioritizes **safety and speed of implementation** over runtime performance.
