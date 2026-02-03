# Design: Dual Stack Package Structure & Implementation

**Status:** Draft
**Goal:** Define the concrete package layout and code changes required to support simultaneous V1 (Legacy) and V2 (Modern) linter stacks.

## 1. Package Structure

The core principle is **leaf-style versioning**. V2 code lives in `/v2` subdirectories at the package level.

```text
github.com/aep-dev/api-linter/
├── cmd/
│   └── api-linter/
│       ├── main.go         (Bootstraps both registries)
│       └── cli.go          (Orchestrates the dual pipeline)
│
├── internal/               (Legacy V1 Utils)
├── internal/v2/            (NEW: Ported V2 Utils)
│
├── lint/                   (Legacy V1 Engine: jhump)
├── lint/v2/                (NEW: Ported V2 Engine: protoreflect)
│
├── rules/                  (Legacy V1 Rules)
│   ├── rules.go            (Registers V1 rules)
│   ├── aep0122/            (Contains V1 implementation)
│   └── internal/
│       ├── utils/          (V1 helpers)
│       └── utils/v2/       (NEW: V2 helpers using protoreflect)
│
└── rules/v2/               (NEW: Ported V2 Rules)
    ├── rules.go            (Registers V2 rules)
    └── aep0122/            (NEW: Contains V2 implementation)
```

## 2. Key File Updates

### A. `cmd/api-linter/cli.go` (The Coordinator)

This file requires the most significant change. It currently just runs the V1 linter. We need to inject the V2 pipeline.

**Current Flow:**
`runCLI` -> `lint` -> `protoparse` -> `lint.New` -> `Output`

**New Flow:**
```go
func (c *cli) lint(...) error {
    // 1. Run V1 Pipeline (Existing)
    // ... protoparse ...
    l1 := lint.New(v1Rules, v1Configs, ...)
    resp1, _ := l1.LintProtos(fds1...)

    // 2. Run V2 Pipeline (New)
    // ... protocompile (copied from upstream cli.go) ...
    // Note: Re-resolves imports, creates new descriptors.
    l2 := lint_v2.New(v2Rules, v2Configs, ...)
    resp2, _ := l2.LintProtos(fds2...)

    // 3. Merge & Deduplicate
    merged := mergeResponses(resp1, resp2)
    
    // 4. Output
    return output(merged)
}
```

### B. `cmd/api-linter/main.go` (Registry Bootstrap)

We need to initialize the V2 registry alongside the V1 registry.

```go
import (
    "github.com/aep-dev/api-linter/lint"
    lint_v2 "github.com/aep-dev/api-linter/lint/v2"
    "github.com/aep-dev/api-linter/rules"
    rules_v2 "github.com/aep-dev/api-linter/rules/v2"
)

var (
    globalRules   = lint.NewRuleRegistry()
    globalRulesV2 = lint_v2.NewRuleRegistry() // NEW
)

func init() {
    rules.Add(globalRules)
    rules_v2.Add(globalRulesV2) // NEW
}

func runCLI(...) {
    c.lint(globalRules, globalRulesV2, ...)
}
```

### C. `rules/v2/rules.go` (V2 Registry)

This file will initially be empty or contain only the pilot rule.

```go
package rules_v2

import (
    "github.com/aep-dev/api-linter/lint/v2"
    "github.com/aep-dev/api-linter/rules/v2/aep0122"
)

func Add(r lint.RuleRegistry) error {
    return aep0122.AddRules(r)
}
```

### D. `rules/rules.go` (V1 Registry)

We must ensure we don't run the same rule twice. When we migrate `aep0122` to V2, we comment it out here.

```go
func Add(r lint.RuleRegistry) error {
    return r.Register(
        // ...
        // aep0122.AddRules, // DISABLED: Migrated to V2
        // ...
    )
}
```

## 3. Detailed Code Changes: The Merge Logic

The `mergeResponses` function in `cli.go` is critical. It must combine `[]lint.Response` (V1) and `[]lint_v2.Response` (V2).

Since `lint.Response` and `lint_v2.Response` are different types (even if structurally identical), we need a common output format or a converter.

**Strategy:**
The `outputFormatFunc` (e.g., for JSON/YAML) usually takes `interface{}`. We can create a unified struct for outputting.

```go
// In cli.go

func mergeResponses(r1 []lint.Response, r2 []lint_v2.Response) []UnifiedResponse {
    // Map by file path to combine problems for the same file
    m := make(map[string]*UnifiedResponse)
    
    // Add V1
    for _, r := range r1 {
        // convert r.Problems (V1) to UnifiedProblem
        // add to map
    }
    
    // Add V2
    for _, r := range r2 {
        // convert r.Problems (V2) to UnifiedProblem
        // add to map
    }
    
    // Flatten map to slice
    return flatten(m)
}
```

*Note:* `lint.Problem` and `lint_v2.Problem` are likely identical in JSON serialization structure, so "UnifiedProblem" might just be `map[string]interface{}` or a copy of the struct definitions.

## 4. Import Path Rewrites

When copying code from `original-api-linter` to `v2/` directories, we must systematically rewrite imports:

*   `github.com/googleapis/api-linter/lint` -> `github.com/aep-dev/api-linter/lint/v2`
*   `github.com/googleapis/api-linter/internal` -> `github.com/aep-dev/api-linter/internal/v2`
*   `github.com/googleapis/api-linter/rules/internal/utils` -> `github.com/aep-dev/api-linter/rules/v2/internal/utils`

This can be automated with `sed` or `gofmt -r`.
