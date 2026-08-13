# Film Generation Request Contract

## Status

Implemented as a versioned Film production fact with a server-owned, revision-safe Canvas Task projection contract. This document does **not** claim a real Provider submission, media result, QC, retry, or Agent Runtime.

## Boundary

`generation_request` is stored as an append-only `FilmArtifact`, scoped to one `Shot`.

```text
Prompt Pack (immutable version)
        ↓
Generation Request (immutable version)
        ↓
Task Draft (read-only, no Task row)
        ↓  future only
Provider Gateway → Task → ProviderJob → Result
```

- `FilmArtifact(generation_request)` is the reproducible request snapshot.
- `Task` remains the future mutable asynchronous execution fact.
- A Canvas `generation` node is only a projection of the request Artifact. Its persisted browser document never owns `DomainRef.taskId`; a future gateway may expose a Task ID only through the server-owned projection Patch described below.
- `Task Draft` is an auditable translation target for the future gateway. It is derived from one exact Generation Request Artifact version and is never placed in the Worker queue.

This separation prevents a queued/running Provider task from overwriting the prompt, references, or request settings that produced it.

## Request payload

The backend accepts the following fields when saving a request:

```json
{
  "promptArtifactId": "prompt-pack-artifact-id",
  "mediaType": "video",
  "aspectRatio": "9:16",
  "durationMs": 5000,
  "outputIntent": "首轮镜头生成"
}
```

The server—not the browser—then writes the authoritative snapshot fields:

```json
{
  "promptArtifactId": "prompt-pack-artifact-id",
  "promptArtifactVersion": 3,
  "compiledPrompt": "the saved Prompt Pack compiled prompt"
}
```

Rules:

- `mediaType` is `image`, `video`, or `audio`.
- Image and video requests require `aspectRatio`.
- Video requests require an integral duration between 1 ms and 120000 ms.
- `outputIntent` is required so a result can later be judged against its production purpose.
- The saved `promptArtifactId` must equal the current Prompt Pack for the same Shot. If it changed, the user explicitly refreshes the request from the latest Prompt Pack and saves a new version.

## Provenance and security

Every Generation Request records `SourceRefs` containing:

1. the current Shot Contract, when present;
2. the authoritative Prompt Pack Artifact; and
3. every linked Shot AssetVersion.

The service rejects credential-shaped fields such as API keys, secrets, credentials, authorization headers, tokens, passwords, and provider keys. Provider configuration remains in backend secret/channel configuration only.

## Canvas projection

The Production Canvas exposes a **生成请求** menu action only after the user selects a saved Prompt Pack. It creates a typed `Prompt Pack → Generation Request` edge and places the node after the Prompt Pack in the auto layout.

The Inspector allows the user to edit output settings and intentionally refresh the frozen prompt snapshot. Saving creates a new Artifact version and advances the Project revision; it does not submit a Provider task.

The Generation Request Inspector also records an explicit request status: `draft`, `review`, `ready`, or `locked`. Only `ready` and `locked` are request-ready for a future gateway; selecting either status still does not submit anything in the current phase.

### Server-owned Task projection Patch

`CanvasProject.PayloadJSON` is saved as a complete browser document, so a Task ID cannot safely be made part of the browser-owned truth. The backend therefore stores a separate `CanvasProjectionPatch` for `film_generation_task`:

```text
Canvas ID + Node ID + Patch kind
        ↓
Target Project ID + Generation Request Artifact ID + exact version
        ↓
already-created Task ID + Patch revision
```

On Canvas read, the service overlays `domainRef.taskId` only when the Canvas is still associated with the Patch target project, the current node is still a `generation` node, and its project ID, Artifact ID and Artifact version all match the Patch. Before every browser Canvas save, client-supplied generation-node `taskId` values are removed. Whole-document replacement keeps Patches for Canvas IDs that still exist, so an old browser snapshot cannot erase an approved binding; deleting a Canvas or unlinking/deleting its target project removes the relevant Patch.

The current binding method is server-only and accepts an already-created Task only after it validates that the request is `ready` or `locked`, the Canvas/project relationship, exact request version, node target, Task type/operation and Task input provenance. There is no public client route for assigning a Task ID, and this Patch step still creates no Task, billing record, ProviderJob or Provider call by itself.

## Task Draft contract

The read-only endpoint is:

```text
GET /projects/:projectId/film-generation-requests/:artifactId/task-draft
```

It returns a provider-independent contract built from the **exact** Artifact ID requested, not from a newer request version for the same Shot. Before it returns anything, the backend also verifies that the recorded Prompt Pack still exists in the same Project and Shot, has the referenced immutable version, and has the same frozen compiled prompt. The returned data includes:

- `taskType`: `canvas_image`, `canvas_video`, or `canvas_audio`;
- `operation`: `film_generation`;
- the frozen compiled prompt, output intent, aspect ratio and video duration;
- Generation Request / Prompt Pack version IDs and recorded `SourceRefs`;
- a deterministic request fingerprint for future idempotency checks; and
- `requestReady`, `submissionState`, `blockers`, and `submissionAllowed`.

The current implementation always reports `providerRouteResolved: false` and `submissionAllowed: false`. It intentionally does **not** create a `Task`, a billing order, a ProviderJob, a Result, or a Task ID. This matters because the existing generic `CreateTask` API immediately makes a queued Task eligible for Worker pickup and billing.

The only allowed `submissionState` values in this phase are:

| State | Meaning |
| --- | --- |
| `awaiting_request_review` | The immutable request is still `draft` or `review`. |
| `awaiting_provider_route` | The request is `ready` or `locked`, but no backend Provider Gateway route has been resolved. |
| `not_submittable` | The request is superseded/archived, malformed, or its project is archived. |

`gatewayInput` intentionally excludes Base URL, channel, model, API key, secret key, authorization headers, and all provider configuration.

## Provider route discovery

The next read-only gateway boundary is:

```text
GET /projects/:projectId/film-generation-requests/:artifactId/provider-routes
```

It starts from the same exact immutable request version, then enumerates only matching **backend system-channel** models. Each candidate reports its channel/model identity, protocol, capability version, pricing readiness, and a redacted readiness state. It never returns a Base URL, API key, secret key, authorization header, provider job ID, or raw channel configuration.

```text
Generation Request (exact version)
        ↓
Read-only route candidates
        ↓  future only
Selected route re-resolved in transaction → Task + billing + Canvas projection
```

Route discovery is not task submission. It performs no Provider request, creates no Task or billing order, and never writes `DomainRef.taskId`. It also requires the requested video duration to be whole seconds before a future task-runtime submission, because the current queued-video protocol accepts integral seconds.

The route catalog is intentionally separated from the eventual submit action. The revision-safe Canvas Patch contract is now present, but no submit route exists yet: the eventual write must still create Task, billing facts and the Patch in one backend transaction. The application must not expose a submit button that pretends a client-side full-document save is atomic.

## Next boundary

The next Film Phase 4/5 slice may bind a saved request to the existing Task runtime through a Provider Gateway adapter. It must preserve this contract, re-resolve the selected backend-only route inside the write transaction, validate capability and billing, create the normal queued Task, and create/update the matching Canvas Projection Patch in that same transaction. It must retain failed/partial task evidence rather than faking a Result.
