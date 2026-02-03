# Plan: Pilot V2 Rule Migration

**Goal:** Implement the infrastructure for V2 (protoreflect) rules and migrate **one** pilot rule (`aep0122/no-self-links`) to prove the concept in a single, shippable commit.

## 1. Candidate Rule: `aep0122/no-self-links`
**Why?**
*   **Simple Logic:** It checks for a single field (`name`) inside a message.
*   **Minimal Deps:** It only relies on `utils.IsResource`.
*   **High Value:** It's a standard check, ensuring we don't break core functionality.

## 2. File Updates Required

### A. Core Infrastructure (`lint/`)

1.  **`lint/rule.go`**:
    *   Add `ProtoRuleV2` interface (accepts `protoreflect.FileDescriptor`).
    *   Add `MessageRuleV2` struct (implementation for messages).
    *   *Note:* We don't need to touch `ProtoRule` (V1).

2.  **`lint/problem.go`**:
    *   Add `DescriptorV2 protoreflect.Descriptor` field to `Problem` struct.
    *   Update methods that use `Descriptor` to fall back to `DescriptorV2` if V1 is nil (e.g., location calculation).

3.  **`lint/lint.go`**:
    *   Update `lintFileDescriptor`:
        *   Add logic to `UnwrapFile()` (jhump -> protoreflect).
        *   Add a type switch to the rule loop.
        *   If `case ProtoRuleV2`: convert descriptor and run.

### B. Utilities (`rules/internal/utils/`)

4.  **`rules/internal/utils/resource.go`** (or `utils/v2/`):
    *   Add `IsResource(m protoreflect.MessageDescriptor) bool`.
    *   *Implementation:* Use `proto.GetExtension` with `protoreflect` options to check for `google.api.resource`.

### C. The Rule (`rules/aep0122/`)

5.  **`rules/aep0122/no_self_links.go`**:
    *   Change variable type: `&lint.MessageRule` -> `&lint.MessageRuleV2`.
    *   Update `OnlyIf`: Change arg to `protoreflect.MessageDescriptor`.
    *   Update `LintMessage`: Change logic to use V2 accessors (e.g., `m.Fields().ByName("name")` instead of `m.FindFieldByName("name")`).

6.  **`rules/aep0122/no_self_links_test.go`**:
    *   *Crucial:* Since the test runner likely uses `lint.Run(rule, descriptor)`, the test code itself might not need drastic changes *if* the Linter adapter works correctly. The adapter should handle the V2 rule even if the test passes in a V1 descriptor (because the linter converts it).
    *   However, if the test manually calls `rule.LintMessage(...)`, that will break. We need to check if we need to update the test setup to use `lint.LintProtos`.

## 3. Execution Steps

1.  **Prep:** `go get google.golang.org/protobuf` (ensure dependency is current).
2.  **Core:** Implement `ProtoRuleV2`, `MessageRuleV2`, and `Problem` updates.
3.  **Engine:** Implement the adapter loop in `lint/lint.go`.
4.  **Utils:** Port `IsResource` to V2.
5.  **Migrate:** Rewrite `no_self_links.go`.
6.  **Verify:** Run `go test ./rules/aep0122/...` to ensure the adapter works.
