# Investigation: Parallel Execution Blocker

**Date:** February 3, 2026
**Subject:** Thread-safety of `api-linter`.

## 1. The Root Cause
The `locations` package uses a **global, thread-unsafe cache** to store source code location maps.

**File:** `locations/locations.go`

```go
var sourceInfoRegistry = sourceInfoRegistryType{}

type sourceInfoRegistryType map[*desc.FileDescriptor]sourceInfo

func (sir sourceInfoRegistryType) sourceInfo(fd *desc.FileDescriptor) sourceInfo {
    answer, ok := sir[fd] // concurrent read
    if !ok {
        // ... compilation logic ...
        sir[fd] = answer // concurrent write
    }
    return answer
}
```

Since the linter processes rules concurrently (or files concurrently), multiple goroutines can call `sourceInfo(fd)` for the same (or different) file descriptors. If one writes to the map while another reads/writes, the runtime will panic with "concurrent map read/write".

## 2. Solution: Mutex Protection
The fix is straightforward. We need to wrap the registry in a `sync.RWMutex`.

```go
var (
    sourceInfoRegistry = sourceInfoRegistryType{}
    sourceInfoRegistryMu sync.RWMutex
)

func (sir sourceInfoRegistryType) sourceInfo(fd *desc.FileDescriptor) sourceInfo {
    sourceInfoRegistryMu.RLock()
    answer, ok := sir[fd]
    sourceInfoRegistryMu.RUnlock()
    if ok {
        return answer
    }

    sourceInfoRegistryMu.Lock()
    defer sourceInfoRegistryMu.Unlock()
    
    // Double-check locking pattern
    if answer, ok := sir[fd]; ok {
        return answer
    }

    // ... compile ...
    sir[fd] = answer
    return answer
}
```

## 3. Scope of Impact
*   **V1 (`locations/locations.go`)**: Affected. Needs fixing.
*   **V2 (`locations/v2/locations.go`)**: Affected (since it was a direct copy). Needs fixing.

## 4. Effort Estimate
**Low.** The fix is localized to a single function in `locations.go` (and its V2 counterpart). It does not require API changes. Once applied, `cmd/buf-plugin-aep/main.go` can safely remove `check.MainWithParallelism(1)`.
