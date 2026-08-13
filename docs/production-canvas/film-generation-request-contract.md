# Film Generation Request Contract

## Status

Implemented as a versioned Film production fact plus a server-owned, revision-safe **single-route atomic Task submission and execution-fact persistence** boundary. The worktree now defines `GenerationAttempt`, `ProviderJob` and Film-aware `Result` registration, but this document does **not** claim a successful real Provider call, real media, QC, end-user Retry UI, browser submit UI, Agent Runtime, or executed acceptance tests for the current changes.

## Boundary

`generation_request` is stored as an append-only `FilmArtifact`, scoped to one `Shot`.

```text
Prompt Pack (immutable version)
        ↓
Generation Request (immutable version)
        ↓
Task Draft (read-only, no Task row)
        ↓
Provider Route Catalog (read-only)
        ↓
Atomic Provider Gateway submit → queued Task + queued GenerationAttempt + billing + Canvas Patch
        ↓  existing asynchronous Worker; not accepted here against a real provider
Worker claim → running GenerationAttempt
        ↓
Provider observation → ProviderJob
        ↓
Task success transaction → succeeded Attempt + Film Result
```

- `FilmArtifact(generation_request)` is the reproducible request snapshot.
- `Task` is the mutable asynchronous queue item and current status pointer. The atomic submit endpoint creates it only after all request, route, capability, billing and Canvas guards pass.
- `GenerationAttempt` preserves each claim/retry execution of that Task. A real upstream identity belongs to an attempt-scoped `ProviderJob`; a successful normalized output belongs to an attempt-scoped `Result`.
- A Canvas `generation` node is only a projection of backend facts. Its persisted browser document never owns Task, Attempt, ProviderJob or Result IDs; those IDs are exposed only through the server-owned read projection described below.
- `Task Draft` is the auditable, side-effect-free translation target consumed by the gateway. It is derived from one exact Generation Request Artifact version and is never placed in the Worker queue.

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

The Generation Request Inspector also records an explicit request status: `draft`, `review`, `ready`, or `locked`. Only `ready` and `locked` may be submitted. Changing status or loading the route catalog still performs no submission; the current UI intentionally has no submit action yet.

### Server-owned Task projection Patch

`CanvasProject.PayloadJSON` is saved as a complete browser document, so a Task ID cannot safely be made part of the browser-owned truth. The backend therefore stores a separate `CanvasProjectionPatch` for `film_generation_task`:

```text
Canvas ID + Node ID + Patch kind
        ↓
Target Project ID + Generation Request Artifact ID + exact version
        ↓
already-created Task ID + Patch revision
```

On Canvas read, the service overlays `domainRef.taskId` only when the Canvas is still associated with the Patch target project, the current node is still a `generation` node, and its project ID, Artifact ID and Artifact version all match the Patch. It then derives the newest `generationAttemptId`, `providerJobId` and `resultId` from execution tables; those IDs are transient and are not stored on the Patch row. Before every browser Canvas save, client-supplied generation-node Task/Attempt/ProviderJob/Result IDs are removed. Whole-document replacement keeps Patches for Canvas IDs that still exist, so an old browser snapshot cannot erase an approved binding; deleting a Canvas or unlinking/deleting its target project removes the relevant Patch.

The standalone binding method remains server-only and accepts an already-created Task only after it validates that the request is `ready` or `locked`, the Canvas/project relationship, exact request version, node target, Task type/operation and Task input provenance. There is no public client route for assigning an arbitrary Task ID. The public gateway submit route does not call this standalone method: it creates Task, optional billing reservation and Patch inside one repository transaction.

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

The draft still reports `providerRouteResolved: false` and `submissionAllowed: false`, because it is only a read operation and no route has been selected in that request. It intentionally does **not** create a `Task`, a billing order, a ProviderJob, a Result, or a Task ID. Submission is a separate authenticated operation so reading the Inspector can never incur cost.

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
        ↓
Explicit authenticated submit
        ↓
Selected route locked/revalidated in transaction → Task + billing + Canvas projection
```

Route discovery is not task submission. It performs no Provider request, creates no Task or billing order, and never writes `DomainRef.taskId`. It also requires the requested video duration to be whole seconds before a future task-runtime submission, because the current queued-video protocol accepts integral seconds.

The route catalog is intentionally separated from submission. Discovery never spends credits. It also runs the same request-specific capability preparation used by submission, so a model that cannot satisfy the current画幅、时长或生成模式 is not marked ready.

## Atomic Task submission

The authenticated write endpoint is:

```text
POST /projects/:projectId/film-generation-requests/:artifactId/tasks
```

The browser may send only:

```json
{
  "canvasId": "canvas-id",
  "canvasNodeId": "generation-node-id",
  "requestFingerprint": "sha256 fingerprint from the Task Draft",
  "channelId": "server catalog channel id",
  "model": "server catalog model key"
}
```

It cannot send a Base URL, API key, secret, authorization header, raw provider options, ProviderJob ID or Task ID. The backend compiles image/video runtime options from the immutable request and the selected model capability declaration. Audio submission remains closed in this slice.

Before commit, the service and repository enforce all of the following:

1. the request is the exact `ready`/`locked` Artifact and its fingerprint still matches;
2. the Canvas is still linked to the Project and the unique node still targets that exact Artifact version;
3. the system channel/model is enabled, authorized, server-configured, price-configured and supports the request;
4. current Project, Canvas payload, Artifact, channel execution fields, model protocol, capability version and price snapshot still match the server-resolved rows under transaction locks;
5. active-task and storage quotas pass; and
6. credit reservation, queued `Task`, queued initial `GenerationAttempt` and `CanvasProjectionPatch` creation/update all commit in one transaction.

If any step fails, no Task, billing order, credit ledger reservation or Patch is committed. Retrying the same Canvas node and exact request version returns the already-bound Task and does not reserve credits again. A different request version cannot replace a still queued/running Task on that node.

The persisted Task contains only `channelId`, model key, normalized non-secret runtime options and an auditable route/version snapshot. Worker execution resolves the current backend channel endpoint and credentials again from system-channel storage. It rejects a queued task if the bound channel-model protocol or capability version changed. Once the transaction commits, an active Worker may claim the Task and call the selected provider, so clients must treat this POST as a potentially billable action. No POST or real Provider call was performed as part of this implementation pass.

## Execution fact chain

The mutable Task row is not treated as complete execution history:

```text
Generation Request vN
        ↓ exact request and route snapshot
Task
        ├── GenerationAttempt 1 → ProviderJob A → Result A
        ├── GenerationAttempt 2 → ProviderJob B → Result B
        └── ...
```

Rules:

- Initial atomic submission creates Attempt 1 as `queued`; it is not a claim and has no `startedAt`.
- Worker claim increments `Task.attempts` and changes the matching Attempt to `running` inside the claim transaction.
- Lease recovery never silently reuses an uncertain upstream execution. It closes the expired running Attempt as `uncertain` and starts a new numbered Attempt. Completion and terminal writes require the original lease owner, so a late Worker cannot close the replacement Attempt.
- A real `providerRequestId` creates or updates a `ProviderJob` under the Attempt number carried by that Worker request. A late observation remains attached to its original Attempt and cannot overwrite the Task pointer owned by a newer Attempt. Normalized states are `accepted`, `running`, `succeeded`, `failed`, `cancellation_requested`, `cancelled`, and `uncertain`.
- A Task failure with a known upstream ID but no confirmed terminal ProviderJob remains `uncertain`; it is not rewritten as a definite provider failure.
- Retry preserves all earlier Attempts and creates a new queued Attempt. If the bound Provider capability changed, Retry is rejected and the user must create a new Generation Request submission.
- A Film Task can become `succeeded` only in the same transaction that marks its Attempt succeeded and inserts `Result(kind = film_generation_result)`. The Result records the exact request ID/version, Task and Attempt.
- Existing `api_call_logs` remain the redacted raw call audit. Attempt and ProviderJob rows never contain credentials, endpoints, headers, request bodies, or response bodies.

The authenticated read-only history endpoint is:

```text
GET /projects/:projectId/film-generation-requests/:artifactId/executions
```

It verifies current-user ownership of the short-drama Project and requested Generation Request before returning Attempts, their exact Generation Request versions, their ProviderJobs, and their Results. It does not read IDs from the browser Canvas document.

## Next boundary

The next Film Phase 5 slice should expose the already-defined submission and execution history through an explicit Inspector cost-confirmation flow, then perform a controlled single-provider acceptance against non-production test inputs. It must show queued/running/uncertain/failed states honestly, must not auto-submit on Inspector read, and must not treat a URL-shaped fixture as real media proof. QC/Retry UI and additional providers remain later boundaries.
