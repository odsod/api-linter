# Codebase Analysis: api-linter

**Date:** February 3, 2026
**Scope:** Full architectural and structural overview.

## 1. High-Level Architecture

The `api-linter` is a Go-based static analysis tool for [Protocol Buffers](https://protobuf.dev/), designed to enforce [AIP (API Improvement Proposals)](https://aep.dev/) standards for Google-style APIs.

The system operates as a pipeline:
1.  **Input:** Accepts `.proto` files via CLI.
2.  **Parsing:** Uses `github.com/jhump/protoreflect` to parse files into rich descriptors.
3.  **Linting Engine:** The `lint` package traverses these descriptors.
4.  **Rule Execution:** Checks descriptors against a registry of enabled rules (mostly in `rules/`).
5.  **Reporting:** Outputs violations as `Problem` objects in various formats (JSON, YAML, GitHub Actions).

## 2. Project Structure

*   **`cmd/`**: Entry points.
    *   `api-linter/`: Main CLI (flag parsing via `spf13/pflag`).
    *   `buf-plugin-aep/`: Plugin for the [Buf](https://buf.build/) ecosystem.
*   **`lint/`**: The core framework (Stable Engine).
    *   `Linter`: Main runner loop.
    *   `Rule`: Interfaces (`ProtoRule`, `MessageRule`, etc.) defining *what* logic to run.
    *   `Problem`: Defines a lint violation.
    *   `RuleRegistry`: Central storage for active rules.
*   **`rules/`**: The business logic (Volatile/Active).
    *   Organized strictly by AIP number (e.g., `rules/aep0131/`).
    *   `internal/utils/`: Shared logic to keep specific rules declarative.

## 3. The Rule System

The rule system favors **declarative composition**.

*   **Granularity:** Rules are small and specific (e.g., "AIP-131: request-message-name").
*   **Structure:**
    ```go
    var requestMessageName = &lint.MethodRule{
        Name:       lint.NewRuleName(131, "request-message-name"),
        OnlyIf:     utils.IsGetMethod,
        LintMethod: utils.LintMethodHasMatchingRequestName,
    }
    ```
*   **Registration:** Each AIP package bulk-registers its rules; `rules/rules.go` aggregates them all.

## 4. Key Dependencies

*   **`github.com/jhump/protoreflect`**: Core engine for parsing proto files into dynamic descriptors.
*   **`github.com/stoewer/go-strcase`**: Validation of naming conventions (Snake, Kebab, Camel).
*   **`github.com/gertd/go-pluralize`**: Grammar enforcement for resource names.
*   **`gopkg.in/yaml.v2`**: Configuration parsing.

## 5. Code Volume & Distribution

The codebase is approximately **30,000 lines of Go code**.

| Component | LOC (Approx) | Share | Description |
| :--- | :--- | :--- | :--- |
| **Rules (`rules/`)** | **~24,600** | **81%** | Core business logic. |
| **Core (`lint/`)** | ~3,100 | 10% | Traversal engine and reporting. |
| **CLI (`cmd/`)** | ~1,300 | 4% | Entry points. |
| **Other** | ~300 | 1% | Utilities. |

**Test Coverage:**
*   **Tests:** ~17,000 LOC (56%)
*   **Implementation:** ~13,300 LOC (44%)
*   **Ratio:** 1.27:1 (Test:Code), indicating strict regression/testing standards.

**Complexity Hotspots:**
*   **AIP-133 (Create):** ~2,300 LOC
*   **AIP-134 (Update):** ~1,800 LOC
*   **AIP-132 (List):** ~1,750 LOC
