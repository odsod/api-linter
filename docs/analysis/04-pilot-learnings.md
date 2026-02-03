# Retrospective: Pilot V2 Migration

**Date:** February 3, 2026
**Scope:** Review of the "Hybrid Adapter" implementation for migrating `api-linter` to `protoreflect`.

## 1. Success of the Hybrid Model
The "Core-First / Parser-First" hybrid strategy (keeping the `jhump` parser but adapting descriptors for V2 rules) proved to be the correct choice.
*   **Low Impact:** We migrated a core rule (`no-self-links`) without breaking the 800+ existing rules.
*   **Lazy Conversion:** implementing the conversion logic inside `lint.go`'s loop ensured that we only pay the cost of converting descriptors (`protodesc.NewFile`) when a V2 rule is actually enabled and running.

## 2. Key Technical Learnings

### A. The Interface Gap (`CommonRule`)
We discovered that simply adding `ProtoRuleV2` wasn't enough because shared infrastructure (like `RuleRegistry` and `buf-plugin-aep`) iterates over *all* rules.
*   **Problem:** `RuleRegistry` was `map[string]ProtoRule`. V2 rules didn't satisfy `ProtoRule` because the `Lint()` signature differs.
*   **Solution:** We introduced a marker interface `CommonRule` (containing only `GetName` and `GetRuleType`).
    *   Registry became `map[string]CommonRule`.
    *   Consumers (like `lint.go` and `main.go`) now type-switch:
        ```go
        switch r := rule.(type) {
        case ProtoRule:   // ...
        case ProtoRuleV2: // ...
        }
        ```

### B. Dependency Resolution is Critical
Converting `jhump` descriptors to `protoreflect` using `protodesc.NewFile` is not stateless. It requires a `protoregistry.Files` resolver to handle imports.
*   **Challenge:** In unit tests, dependencies (like `google/protobuf/descriptor.proto`) are often implicit. `protodesc` fails if it can't resolve them.
*   **Workaround:** We utilized `protoregistry.GlobalFiles` for standard imports in tests. For a robust production migration, we may need a more sophisticated caching resolver that persists across rule executions to avoid re-parsing dependencies for every rule.

### C. The `buf-plugin` Canary
The `cmd/buf-plugin-aep` integration failed initially, serving as an excellent canary.
*   **Lesson:** Internal tools often couple tightly to interfaces. We assumed `lint/` changes were internal, but the plugin (in `cmd/`) acts as an external consumer. Updating `addProblem` to handle both V1 and V2 descriptor paths was necessary to keep the plugin functional.

### D. Testing Infrastructure
Our test helpers (`SetDescriptor`) were tightly coupled to V1.
*   **Lesson:** We cannot migrate rules without migrating test infrastructure. We added `SetDescriptorV2` to `Problems`, but ideally, future work should create `testutils.ParseProtoV2` to reduce boilerplate in test files.

## 3. Recommendations for Next Steps

1.  **Standardize V2 Test Helpers:** Create a dedicated test parsing pipeline that produces `protoreflect` descriptors natively, reducing the need for manual conversion in every test case.
2.  **Optimize Conversion:** Currently, `getV2Descriptor` runs for *every* V2 rule. If we have 50 V2 rules, we might convert the file 50 times (depending on caching). We should cache the `protoreflect.FileDescriptor` at the `Linter` level (per file) to ensure we only converting once per file.
3.  **Deprecation Checks:** We temporarily bypassed `disableDeprecated` for V2. We need to implement the `protoreflect` equivalent of checking `GetOptions().GetDeprecated()`.
