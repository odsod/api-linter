# Retrospective & Migration Guide: The Dual Stack Architecture

**Date:** February 3, 2026
**Status:** Architecture Implemented. Migration In Progress.

## 1. Retrospective: The Strategic Pivot

We initially attempted a "Hybrid Adapter" approach (converting legacy descriptors to modern ones on-the-fly). While technically feasible, it introduced significant complexity and risk into the core linter engine.

We pivoted to a **Dual Stack (Vendoring)** strategy. This approach treats the migration not as a refactoring task, but as an **integration task**. By vendoring the already-modernized core from the upstream `google/api-linter` into `v2/` directories, we achieved:

1.  **Risk Reduction:** Legacy rules run on the untouched Legacy engine. Modern rules run on the production-proven Modern engine.
2.  **Isolation:** V1 and V2 logic is physically separated in `cli_v1.go` and `cli_v2.go`.
3.  **Velocity:** We skipped weeks of manual refactoring of the `lint` package.

## 2. Repository Overview (Current State)

The repository is now partitioned into two parallel stacks.

```text
github.com/aep-dev/api-linter/
├── cmd/
│   ├── api-linter/
│   │   ├── main.go         # Bootstraps both V1 and V2 registries
│   │   ├── cli.go          # Orchestrator: merges V1 and V2 results
│   │   ├── cli_v1.go       # Legacy Pipeline (jhump/protoparse)
│   │   └── cli_v2.go       # Modern Pipeline (bufbuild/protocompile)
│   └── buf-plugin-aep/
│       ├── main.go         # Orchestrator
│       ├── v1.go           # Legacy Handler (wraps protoreflect -> desc)
│       └── v2.go           # Modern Handler (uses native protoreflect)
│
├── lint/                   # [LEGACY] V1 Engine
├── lint/v2/                # [MODERN] V2 Engine (Vendored)
│
├── rules/                  # [LEGACY] V1 Rules
│   ├── rules.go            # V1 Registry
│   └── ...                 # 800+ legacy rules
│
└── rules/v2/               # [MODERN] V2 Rules
    ├── rules.go            # V2 Registry
    ├── aep0122/            # Migrated Pilot Rule (no-self-links)
    └── internal/           # V2 Shared Utilities
```

## 3. The Way Forward: Migration Playbook

To migrate the remaining rules, contributors should follow this loop:

### Step 1: Port a Rule Package
1.  Copy the rule package from `original-api-linter/rules/aepXXXX` to `rules/v2/aepXXXX`.
2.  **Rewire Imports:** In the new files, replace all V1 imports with V2 imports:
    *   `.../lint` -> `.../lint/v2`
    *   `.../rules/internal/utils` -> `.../rules/internal/utils/v2`
3.  **Check Utils:** If the rule relies on `utils` functions that haven't been ported to `rules/internal/utils/v2` yet, copy those specific functions from upstream into the V2 utils package.

### Step 2: Register & Disable
1.  **Register V2:** Add `aepXXXX.AddRules` to `rules/v2/rules.go`.
2.  **Disable V1:** Comment out `aepXXXX.AddRules` in `rules/rules.go`.

### Step 3: Verify
1.  Run tests: `go test ./rules/v2/aepXXXX/...`
2.  (Optional) Run the linter against a known violation to confirm the rule still fires.

## 4. Files Requiring Attention (The Backlog)

### High Priority (Migration Targets)
*   **`rules/`**: ~25 subdirectories (AIPs) still need to be moved to `rules/v2/`.
*   **`rules/internal/utils/v2`**: Currently contains a subset of helpers. Needs to be populated with the rest of the utils from upstream as rules require them.

### Low Priority (Cleanup - Post Migration)
*   **`lint/` (Root)**: Delete once all rules are migrated.
*   **`internal/` (Root)**: Delete once V1 is gone.
*   **`cmd/**/cli_v1.go`**: Delete the legacy pipeline code.

## 5. Known Limitations & Optimizations
*   **Double Parsing:** The CLI currently parses input files twice (once for V1, once for V2). This is acceptable for now but creates memory pressure on large repos.
    *   *Optimization:* Once V1 usage drops below 50%, we could consider enabling V2 parsing *first*, and only running V1 parsing if V1 rules are active.
*   **Deprecation Checks:** `rule_enabled.go` in V2 needs to implement proper deprecation checking using `protoreflect`.
