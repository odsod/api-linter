# Plan: Dual Stack Pilot (PoC)

**Goal:** Implement the "Dual Stack" architecture and migrate `aep0122/no-self-links` to V2 by vendoring code from `original-api-linter`.

## 1. Dependencies
*   Update `go.mod` to include `github.com/bufbuild/protocompile`.

## 2. Vendor V2 Core Infrastructure
We will copy the V2-ready code from `original-api-linter` into `v2` leaf directories.

*   **`lint/v2/`**:
    *   Source: `original-api-linter/lint/`
    *   Destination: `lint/v2/`
*   **`internal/v2/`**:
    *   Source: `original-api-linter/internal/`
    *   Destination: `internal/v2/`
*   **`rules/internal/utils/v2/`**:
    *   Source: `original-api-linter/rules/internal/utils/`
    *   Destination: `rules/internal/utils/v2/`
*   **`rules/internal/testutils/v2/`**:
    *   Source: `original-api-linter/rules/internal/testutils/`
    *   Destination: `rules/internal/testutils/v2/`
*   **`rules/internal/data/v2/`**:
    *   Source: `original-api-linter/rules/internal/data/`
    *   Destination: `rules/internal/data/v2/`

## 3. Vendor Pilot Rule
*   **`rules/v2/aep0122/`**:
    *   Source: `original-api-linter/rules/aep0122/`
    *   Destination: `rules/v2/aep0122/`

## 4. Refactor Imports (Mass Search-and-Replace)
The copied code still refers to `github.com/googleapis/api-linter/...`. We need to rewrite this to `github.com/aep-dev/api-linter/...` and point to the `v2` packages.

*   **Pattern 1 (Base):** `github.com/googleapis/api-linter` -> `github.com/aep-dev/api-linter`
*   **Pattern 2 (Lint):** `.../lint` -> `.../lint/v2`
*   **Pattern 3 (Internal):** `.../internal` -> `.../internal/v2` (careful not to match `rules/internal`)
*   **Pattern 4 (Rules Utils):** `.../rules/internal` -> `.../rules/v2/internal`

## 5. Bootstrap V2 Registry
*   **`rules/v2/rules.go`**:
    *   Create a minimal registry file.
    *   Import `rules/v2/aep0122`.
    *   Expose `Add(r lint_v2.RuleRegistry)`.

## 6. Update CLI (Dual Pipeline)
*   **`cmd/api-linter/cli.go`**:
    *   Import `lint_v2 "github.com/aep-dev/api-linter/lint/v2"`.
    *   Implement `runV2(files []string) ([]lint_v2.Response, error)` using `protocompile` (referencing logic from `original-api-linter/cmd/api-linter/cli.go`).
    *   Update `lint()` to call `runV1` then `runV2`.
    *   Implement `mergeResponses(r1, r2)`.

## 7. Disable V1 Rule
*   **`rules/aep0122/aep0122.go`**: Comment out `noSelfLinks` registration.

## 8. Verification
*   Run `go mod tidy`.
*   Run `go test ./rules/v2/aep0122/...`.
*   Run `go build ./cmd/api-linter`.
*   Run the linter against a test file violating AIP-122.
