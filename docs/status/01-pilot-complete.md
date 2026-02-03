# Status: Pilot Migration Complete

**Date:** February 3, 2026
**Achievement:** Successfully migrated `aep0122/no-self-links` to `protoreflect` V2 while maintaining the legacy `jhump` linter core.

## 1. Key Changes Implemented

### Core Infrastructure
*   **`lint/rule.go`**: Added `ProtoRuleV2` and `MessageRuleV2` interfaces.
*   **`lint/lint.go`**: Implemented the **Hybrid Adapter**. The linter loop now detects V2 rules and lazy-converts `jhump` descriptors to `protoreflect` descriptors using `protodesc.NewFile`.
*   **`lint/problem.go`**: Updated `Problem` struct to hold `DescriptorV2` and calculate source locations correctly from either V1 or V2 descriptors.

### Utilities
*   **`rules/internal/utils/resource.go`**: Added `IsResourceV2` helper using `protoreflect` API to check `aep.api.resource` extensions.

### Pilot Rule
*   **`rules/aep0122/no_self_links.go`**: Converted to `MessageRuleV2`. Logic rewritten to use `m.Fields().Get(i)` iteration.
*   **`rules/aep0122/no_self_links_test.go`**: Updated tests to manually convert test descriptors to V2 before invoking the rule, verifying the rule logic works in isolation.

## 2. Verification
Ran `go test ./rules/aep0122/...` -> **PASS**.

## 3. Next Steps
1.  **Refine `ruleIsEnabled`**: Currently `ruleIsEnabled` in `lint/rule_enabled.go` was patched to accept V2 descriptors, but deprecation checks for V2 are TODO.
2.  **Scale Up**: Begin migrating other rules in batches (e.g., AIP-192, AIP-131).
3.  **Port Utils**: Systematically migrate `rules/internal/utils` functions to V2 as needed by rules.
