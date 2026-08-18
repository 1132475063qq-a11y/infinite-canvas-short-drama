# Production Canvas v2 — Phase 0 Baseline

## Source baseline

- Git commit: `30f18f3b18f652c44385e6c7ac2521d81b6f7df8`
- Branch: `main`
- Purpose: imported `open-ai-canvas-main` before Production Canvas v2 work.

## Runtime database baseline

- Default development driver: SQLite.
- Schema authority: `backend/internal/database/schema.go` via `database.Models()` and `database.MigrateSchema()`.
- Verified local development database: 53 application tables.
- Recovery copy: a Git-ignored local database backup was created before the storage migration.

The SQLite database and its backup are deliberately excluded from Git: they can contain user projects, sessions, and provider configuration. Schema changes must be represented by Go models and migration tests, not committed user data.

## Canvas document baseline

`CanvasProjectDocument` is a backwards-compatible document stored in `CanvasProject.PayloadJSON` and local canvas storage. Its canonical v2 contract is:

```ts
{
  schemaVersion: 2,
  projectId?: string,
  layout: { gridSize: 8 },
  nodes: CanvasNodeData[],
  connections: CanvasConnection[],
  viewport: ViewportTransform,
  groups: CanvasDocumentGroup[],
}
```

Film semantic fields remain optional on `CanvasNodeData` so old documents open unchanged. `migrateCanvasProjectDocument()` upgrades film-compatible legacy nodes, accepts legacy `edges` and `layout.grid` aliases, and normalizes them to `connections` and `layout.gridSize` without replacing the existing canvas engine.

## Merged AI Creative Studio boundary

The repository now has two explicitly isolated production domains:

```text
Shared Canvas Core
├── Film       → FilmNodeKind / FilmArtifact / Scene / Shot
└── Ecommerce  → Ecommerce contracts and Provider-free Prototype A artifacts
```

Ecommerce reuses the Canvas Core, Project/Asset/Task/Result and Provider boundaries, but it must not add Ecommerce fields to `FilmArtifact` or change Film node semantics. The Ecommerce contract and current `READY FOR PROTOTYPE IMPLEMENTATION` gate are recorded in [`ecommerce-prototype-addendum.md`](./ecommerce-prototype-addendum.md), with the canonical split documents [`ecommerce-agent-team-v1.2.1-codex-ready.md`](./ecommerce-agent-team-v1.2.1-codex-ready.md) and [`ecommerce-prototype-implementation-spec-v1.0.md`](./ecommerce-prototype-implementation-spec-v1.0.md). The repository now contains a Provider-free Prototype A contract skeleton, an independent `EcommerceArtifact` persistence API, a project-type entry and a developer panel; no Ecommerce Agent Runtime, real Provider call, browser proof, media-quality result, or real QA proof is claimed.

## Film Generation Request baseline

Film `generation_request` is a versioned `FilmArtifact`, not a mutable task row. It snapshots the current Prompt Pack ID, Prompt Pack version and compiled prompt on the backend, records Shot Contract / Prompt Pack / Shot AssetVersion provenance, and rejects credential-shaped or unowned provider-execution fields. A read-only Task Draft is derived from one exact request version, and a separate read-only Provider Route catalog reveals only redacted matching backend system-channel candidates. An authenticated image/video submit boundary can atomically reserve billing, create one normal queued Task plus its initial queued `GenerationAttempt`, and install the revision-safe server `canvas_projection_patch`; duplicate submission returns the existing binding. Worker claim, attempt-scoped `ProviderJob`, Film-aware `Result`, retry history and read-time Canvas execution IDs now have persistence contracts. The browser document never persists or assigns those runtime IDs, and no provider credential is copied into Task or execution facts. Controlled image acceptance has been recorded; video, current worktree regression, QC, Retry UI and browser submission acceptance still require the scopes listed in `pending-test.mdx`. The complete contract is in [`film-generation-request-contract.md`](./film-generation-request-contract.md).

## Golden fixture

`web/test/fixtures/parasite-ad-sc01.ts` is the fixed, provider-free starting fixture for 《寄生广告》第一场：`SC01 黑市诊所`.

It intentionally verifies only the current baseline production facts—project, scene, shot, and planned reference assets. It does not prove a real Provider submission, media result, QC, retry, or Agent Runtime.

## Phase 0 exit criteria

- [x] Main source baseline identified and tagged.
- [x] Database schema authority and recovery copy recorded without committing user data.
- [x] Canvas document v2 contract recorded.
- [x] Provider-free Golden Fixture added and covered by tests.
- [x] Frontend and backend baseline tests run before the next phase.
