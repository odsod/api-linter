# Status: Ready for Dual Stack Pilot

**Date:** February 3, 2026
**State:** Documentation updated. Strategy pivoted to Dual Stack (Vendoring).

## 1. Decision Record
We have abandoned the Hybrid Adapter approach (converting V1 descriptors to V2 on the fly) in favor of a **Dual Stack** approach.
*   **Why:** `original-api-linter` has already completed the V2 migration in a separate branch. We can vendor this code directly, avoiding the need to manually rewrite hundreds of rules and utilities.
*   **Performance:** We accept the cost of double-parsing (running both `jhump` and `protocompile` sequentially) during the transition period.

## 2. Next Steps
Execute `docs/plans/01-pilot-migration.md`:
1.  Vendor `lint/`, `internal/` -> `v2/`.
2.  Vendor `rules/aep0122` -> `rules/v2/aep0122`.
3.  Update `cli.go` to run both stacks.
