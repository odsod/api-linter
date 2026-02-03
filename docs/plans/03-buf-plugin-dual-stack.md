# Plan: Buf Plugin Dual Stack Support

**Goal:** Update `cmd/buf-plugin-aep` to run both V1 (`jhump`) and V2 (`protoreflect`) rules in parallel.

## 1. Imports Update
*   Import `lint_v2 "github.com/aep-dev/api-linter/lint/v2"`.
*   Import `rules_v2 "github.com/aep-dev/api-linter/rules/v2"`.

## 2. Registry Aggregation
The `newSpec` function needs to iterate over *both* registries.

```go
func newSpec() (*check.Spec, error) {
    // 1. Initialize V1 Registry
    v1Reg := lint.NewRuleRegistry()
    rules.Add(v1Reg)

    // 2. Initialize V2 Registry
    v2Reg := lint_v2.NewRuleRegistry()
    rules_v2.Add(v2Reg)

    // 3. Flatten into ruleSpecs
    var ruleSpecs []*check.RuleSpec
    
    // Add V1 Rules
    for _, r := range v1Reg {
        s, _ := newRuleSpecV1(r)
        ruleSpecs = append(ruleSpecs, s)
    }
    
    // Add V2 Rules
    for _, r := range v2Reg {
        s, _ := newRuleSpecV2(r)
        ruleSpecs = append(ruleSpecs, s)
    }
    
    // ... return Spec
}
```

## 3. Handler Implementation
We need two handler constructors because the Rule interfaces differ.

*   `newRuleHandlerV1(r lint.ProtoRule)`: Uses `ctx` to get `[]*desc.FileDescriptor`.
*   `newRuleHandlerV2(r lint_v2.ProtoRule)`: Uses `check.Request` to get `protoreflect.FileDescriptor` directly.
    *   *Note:* The `bufplugin` library natively provides `protoreflect`. We previously wrapped it for V1. For V2, we can just use it!

```go
func newRuleHandlerV2(r lint_v2.ProtoRule) check.RuleHandler {
    return check.RuleHandlerFunc(func(ctx, w, req) error {
        for _, fd := range req.FileDescriptors() {
            if fd.IsImport() { continue }
            
            // Native protoreflect!
            problems := r.Lint(fd.ProtoreflectFileDescriptor())
            
            for _, p := range problems {
                addProblemV2(w, p)
            }
        }
        return nil
    })
}
```

## 4. Problem Reporting
*   `addProblem` currently handles `lint.Problem`.
*   Create `addProblemV2` for `lint_v2.Problem`.
    *   V2 `Problem` uses `Descriptor` (protoreflect interface).
    *   Use `Descriptor.ParentFile().Path()` for filenames.
    *   Use `Descriptor.ParentFile().SourceLocations().ByDescriptor(...)` for location.

## 5. Execution Steps
1.  Refactor `main.go` to split V1/V2 spec generation.
2.  Implement `newRuleHandlerV2`.
3.  Implement `addProblemV2`.
4.  Run `go build` to verify.
