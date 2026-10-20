# Migration Notes

**Model self-assessed confidence:** 82% (not a measured result)  
**Build at time of writing:** passed  
**Target unit tests:** not run  
**Behavior compared with the original:** passed

The final recommendation is in the pull request description.

---

## What was migrated

- `notes.js` → `internal/notes/store.go` (90% confidence)
- `index.js` → `internal/api/handler.go` (20% confidence) ⚠️ needs review
## Files requiring manual review

These files were migrated but scored below the confidence threshold.
Review them carefully before merging.

### `index.js`
Confidence: 20%
