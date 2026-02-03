# Code Examples: Hybrid Linter Implementation

**Goal:** Concrete code examples showing how to implement the Hybrid Adapter strategy where `jhump` and `protoreflect` rules coexist.

## 1. Defining V2 Interfaces (`lint/rule.go`)

We introduce new interfaces (or extended interfaces) that support `protoreflect` types.

```go
package lint

import (
    "github.com/jhump/protoreflect/desc"
    "google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoRuleV2 is the new interface for modern rules.
// Note: It accepts protoreflect.FileDescriptor instead of *desc.FileDescriptor.
type ProtoRuleV2 interface {
    GetName() RuleName
    Lint(protoreflect.FileDescriptor) []Problem
    GetRuleType() RuleType
}

// MessageRuleV2 is the modern replacement for MessageRule.
type MessageRuleV2 struct {
    Name        RuleName
    LintMessage func(protoreflect.MessageDescriptor) []Problem
    OnlyIf      func(protoreflect.MessageDescriptor) bool
    RuleType    *RuleType
}

func (r *MessageRuleV2) GetName() RuleName { return r.Name }
func (r *MessageRuleV2) GetRuleType() RuleType {
    if r.RuleType == nil { return NotCategorizedRule }
    return *r.RuleType
}

// Lint implements the adapter logic: it traverses the file using V2 accessors.
func (r *MessageRuleV2) Lint(fd protoreflect.FileDescriptor) []Problem {
    var problems []Problem
    
    // Modern traversal using .Messages() list instead of GetMessageTypes() slice
    messages := []protoreflect.MessageDescriptor{}
    for i := 0; i < fd.Messages().Len(); i++ {
        messages = append(messages, fd.Messages().Get(i))
    }
    // (Recursive nested message logic omitted for brevity, but similar)

    for _, msg := range messages {
        if r.OnlyIf == nil || r.OnlyIf(msg) {
            problems = append(problems, r.LintMessage(msg)...)
        }
    }
    return problems
}
```

## 2. The Hybrid Linter Engine (`lint/lint.go`)

We modify the `lintFileDescriptor` function to detect which interface the rule satisfies and convert the descriptor if necessary.

```go
package lint

import (
    "github.com/jhump/protoreflect/desc"
    "google.golang.org/protobuf/reflect/protodesc"
    "google.golang.org/protobuf/types/descriptorpb"
)

func (l *Linter) lintFileDescriptor(fd *desc.FileDescriptor) (Response, error) {
    resp := Response{
        FilePath: fd.GetName(),
        Problems: []Problem{},
    }

    // Lazy conversion: only convert to V2 descriptor if we encounter a V2 rule.
    var v2FD protoreflect.FileDescriptor
    var v2Err error

    getV2Descriptor := func() (protoreflect.FileDescriptor, error) {
        if v2FD != nil {
            return v2FD, nil
        }
        // Conversion logic: jhump -> proto -> v2 descriptor
        // Note: In a real implementation, we might cache this to avoid re-parsing.
        protoV2 := fd.AsFileDescriptorProto() 
        // We assume we have a resolver available or create a basic one
        v2FD, v2Err = protodesc.NewFile(protoV2, nil) 
        return v2FD, v2Err
    }

    for _, rule := range l.rules {
        if !l.configs.IsRuleEnabled(string(rule.GetName()), fd.GetName()) {
            continue
        }

        // --- The Switch ---
        switch r := rule.(type) {
        
        // CASE 1: Legacy Rule (Business as usual)
        case ProtoRule:
            if probs, err := l.runAndRecoverFromPanics(r, fd); err == nil {
                resp.Problems = append(resp.Problems, probs...)
            }

        // CASE 2: Modern V2 Rule (The Adapter)
        case ProtoRuleV2:
            v2Desc, err := getV2Descriptor()
            if err != nil {
                // Log error: failed to convert descriptor for V2 rule
                continue 
            }
            // Execute using the converted descriptor
            if probs, err := l.runAndRecoverFromPanicsV2(r, v2Desc); err == nil {
                resp.Problems = append(resp.Problems, probs...)
            }
        }
    }
    
    return resp, nil
}
```

## 3. Handling Dual Problems (`lint/problem.go`)

The `Problem` struct must be able to hold either descriptor type so that the final reporter can look up line numbers.

```go
package lint

import (
    "github.com/jhump/protoreflect/desc"
    "google.golang.org/protobuf/reflect/protoreflect"
)

type Problem struct {
    Message    string
    Suggestion string
    
    // Legacy Descriptor
    Descriptor desc.Descriptor
    
    // New V2 Descriptor
    DescriptorV2 protoreflect.Descriptor
    
    // Location is calculated from either of the above
    Location   *Location 
}

// In lint/response.go or similar, where we format the output:
func (p *Problem) CalculateLocation() {
    if p.Descriptor != nil {
        // Use jhump logic to find location
        loc := p.Descriptor.GetSourceInfo()
        // ...
    } else if p.DescriptorV2 != nil {
        // Use protoreflect logic
        loc := p.DescriptorV2.ParentFile().SourceLocations().ByDescriptor(p.DescriptorV2)
        // ...
    }
}
```

## 4. Summary

This code demonstrates that the "Adapter" is essentially a `type switch` in the main loop and a lazy converter function. 

*   **Complexity:** Low. It is localized to `lint.go` and `rule.go`.
*   **Performance:** We only pay the conversion cost (`protodesc.NewFile`) once per file, and only if V2 rules are enabled.
*   **Safety:** The parser (jhump) remains the single source of truth for the AST.
