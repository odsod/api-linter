# Migration Analysis: jhump/protoreflect to google.golang.org/protobuf

**Commit:** `42e6805`
**Date:** October 22, 2025
**Author:** Santiago Quiroga

## 1. Overview
This commit represents a massive architectural refactoring (841 files changed) to migrate the `api-linter` from the third-party `github.com/jhump/protoreflect` library to the official `google.golang.org/protobuf` (v2) library.

The migration was necessary to modernize the codebase, as `jhump/protoreflect` was effectively the standard for dynamic introspection before the official v2 API was released.

## 2. Key Architectural Changes

### A. Parsing Strategy (`cmd/api-linter/cli.go`)
The official protobuf library focuses on *generated* code and does not strictly provide a parser for raw `.proto` source files. To fill this gap, the project adopted **`github.com/bufbuild/protocompile`**.

*   **Before:** `jhump/protoreflect/desc/protoparse` handled parsing and import resolution.
*   **After:** `bufbuild/protocompile` handles parsing via a `protocompile.Compiler`.
*   **Structural Refactor (Custom Reporter):** `jhump` collected all syntax errors by default. `protocompile` fails on the first error. The team had to implement a custom `reporter.NewReporter` to capture all errors and signal the compiler to continue, preserving the user experience of seeing all syntax errors at once.
*   **Structural Refactor (Composite Resolver):** Unifying lookups for local `.proto` files and pre-compiled `.protoset` files required implementing a `CompositeResolver`. This manually prioritizes the `SourceResolver` and falls back to a `DescriptorSet` resolver.

### B. Reflection Interface (`lint/rule.go`)
The core definitions of what a "Rule" is had to change because the underlying types changed from struct pointers to interfaces.

*   **1-1 Type Replacement:**
    *   `*desc.FileDescriptor` -> `protoreflect.FileDescriptor`
    *   `*desc.MessageDescriptor` -> `protoreflect.MessageDescriptor`
*   **Structural Refactor (Traversal Logic):**
    *   **Old (Slice-based):** Simple range loops over slices.
    *   **New (Accessor-based):** The v2 API uses a List interface for child elements.
        ```go
        for i := 0; i < fd.Messages().Len(); i++ {
            msg := fd.Messages().Get(i)
            // ...
        }
        ```
    *   This required updating every single traversal loop across the entire ruleset (~800 files).

### C. Extension & Option Handling (`rules/internal/utils/extension.go`)
This was the most complex logic shift. Accessing custom options (e.g., `google.api.http`) no longer returns typed Go structs directly in dynamic contexts.

*   **Before:** `proto.GetExtension(opts, apb.E_Http)` often returned a castable struct.
*   **After (The "Marshal Roundtrip" Pattern):** To bridge the gap between dynamic `protoreflect.Message` objects and the typed structs rules expect, a generic helper was introduced:
    1.  Get the extension as a `protoreflect.Value`.
    2.  If it can't be cast to the target type `T`, **marshal it to wire format**.
    3.  **Unmarshal it back** into a new instance of `T`.
    *   This ensures rules can still use strongly-typed accessors (e.g., `res.GetType()`) even if the input was parsed dynamically.

### D. HTTP Rule Parsing (`rules/internal/utils/http.go`)
*   **Structural Refactor:** The logic to parse `google.api.http` annotations was completely rewritten. Instead of relying on the generated struct's helper methods, the utility now uses `rule.Range(...)` to dynamically iterate over fields like `body`, `get`, `post`, and `custom` to populate a local, linter-specific `HTTPRule` struct.

### E. Source Code Info (Comments)
*   **1-1 Replacement:** Accessing comments became more indirect but consistent.
    *   **Old:** `d.GetSourceInfo().GetLeadingComments()`
    *   **New:** `d.ParentFile().SourceLocations().ByDescriptor(d).LeadingComments`

## 3. Impact on Rules
Every single AIP rule (hundreds of files in `rules/`) was updated.
*   The `LintXXX` and `OnlyIf` function signatures were changed to use `protoreflect` interfaces.
*   Helper functions in `rules/internal/utils` were significantly expanded to hide the complexity of the new reflection API from rule authors.

## 4. Conclusion
This migration effectively "future-proofed" the linter. By moving to `protocompile` and `protoreflect`, the linter now shares the same foundation as the modern Protobuf ecosystem (including Buf), ensuring better long-term maintenance and compatibility.