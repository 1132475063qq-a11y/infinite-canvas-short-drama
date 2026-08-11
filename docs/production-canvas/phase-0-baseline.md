# Production Canvas v2 — Phase 0 Baseline

## Source baseline

- Git commit: `30f18f3b18f652c44385e6c7ac2521d81b6f7df8`
- Branch: `main`
- Purpose: imported `open-ai-canvas-main` before Production Canvas v2 work.

## Runtime database baseline

- Default development driver: SQLite.
- Schema authority: `backend/internal/database/schema.go` via `database.Models()` and `database.MigrateSchema()`.
- Verified local development database: 53 application tables.
- Recovery copy: `.local/project-workbench-debug/backups/open_ai_canvas-before-storage-v2-20260811-140557.db`.

The SQLite database and its backup are deliberately excluded from Git: they can contain user projects, sessions, and provider configuration. Schema changes must be represented by Go models and migration tests, not committed user data.

## Canvas document baseline

`CanvasProjectDocument` is a backwards-compatible document stored in `CanvasProject.PayloadJSON` and local canvas storage. Its v2 minimum contract is:

```ts
{
  schemaVersion: 2,
  layout: { gridSize: 8 },
  nodes: CanvasNodeData[],
}
```

Film semantic fields remain optional on `CanvasNodeData` so old documents open unchanged. `migrateCanvasProjectDocument()` upgrades film-compatible legacy nodes without replacing the existing canvas engine.

## Golden fixture

`web/test/fixtures/parasite-ad-sc01.ts` is the fixed, provider-free starting fixture for 《寄生广告》第一场：`SC01 黑市诊所`.

It intentionally verifies only the current baseline production facts—project, scene, shot, and planned reference assets. It does not claim that prompt compilation, generation, QC, retry, or Agent Runtime are complete.

## Phase 0 exit criteria

- [x] Main source baseline identified and tagged.
- [x] Database schema authority and recovery copy recorded without committing user data.
- [x] Canvas document v2 contract recorded.
- [x] Provider-free Golden Fixture added and covered by tests.
- [x] Frontend and backend baseline tests run before the next phase.
