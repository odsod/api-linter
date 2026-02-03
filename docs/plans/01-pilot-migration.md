# Plan: Pilot Dual Stack Migration

**Goal:** Establish the V2 pipeline and migrate the first rule (`aep0122`) using the "Dual Stack via Vendoring" strategy.

## 1. Preparation
*   Ensure `original-api-linter` is checked out at commit `42e6805`.
*   Ensure `go.mod` is updated to include V2 dependencies (`bufbuild/protocompile`, `google.golang.org/protobuf`).

## 2. Infrastructure Setup (The "V2 Core")
We need to create the V2 packages by vendoring code from upstream.

1.  **`internal/v2`**:
    *   Copy `original-api-linter/internal` -> `internal/v2`.
    *   *Why:* Contains version info and potentially shared internal logic updated for V2.
2.  **`lint/v2`**:
    *   Copy `original-api-linter/lint` -> `lint/v2`.
    *   *Modifications:* Update imports inside these files to point to `github.com/aep-dev/api-linter/internal/v2` instead of the original path.
3.  **`rules/v2` (Scaffolding)**:
    *   Create `rules/v2`.
    *   Copy `original-api-linter/rules/internal` -> `rules/v2/internal`.
    *   *Why:* This contains the V2 versions of `utils` (e.g., `IsResource` using `protoreflect`).

## 3. Pilot Rule Migration: `aep0122`

1.  **Vendor Rule:**
    *   Copy `original-api-linter/rules/aep0122` -> `rules/v2/aep0122`.
2.  **Register Rule:**
    *   Create `rules/v2/rules.go` (based on upstream `rules/rules.go`).
    *   Register only `aep0122`.
3.  **Deactivate V1 Rule:**
    *   Modify `rules/rules.go` (V1) to **remove** `aep0122` registration.
    *   *Note:* We don't delete the code yet, just unregister it to avoid noise.

## 4. CLI Integration (`cmd/api-linter/cli.go`)

This is the most complex step. We need to implement the "Second Loop".

1.  **Import V2:** Add imports for `lint/v2` and `rules/v2`.
2.  **Implement `runV2`:**
    *   Copy the parsing logic from upstream `cli.go` (using `protocompile`).
    *   Instantiate `lint_v2.New` with `rules_v2.GlobalRegistry`.
    *   Run linting.
3.  **Merge Results:**
    *   Call `runV1`.
    *   Call `runV2`.
    *   Append `problemsV2` to `problemsV1`.

## 5. Verification

1.  **Build:** Ensure all imports resolve.
2.  **Test:** Run `api-linter` against a sample proto that violates AIP-122 (Self Links).
    *   *Expectation:* The error is reported (by the V2 stack).
3.  **Regression:** Run against other violations.
    *   *Expectation:* Other errors are reported (by the V1 stack).

## 6. Cleanup
Once verified:
*   Delete `rules/aep0122` (V1 code).