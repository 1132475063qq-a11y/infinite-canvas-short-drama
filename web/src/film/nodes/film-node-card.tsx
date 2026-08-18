import { AlertTriangle, Box, CircleDot, Clapperboard, Drama, MapPin, Package, PersonStanding, Sparkles } from "lucide-react";

import { filmText, resolveFilmNode } from "@/film/domain/film-node-resolver";
import type { ProjectDetail } from "@/services/api/projects";
import type { CanvasNodeData } from "@/types/canvas";

const labels: Record<string, string> = { scene: "SCENE", shot: "SHOT CONTRACT", character: "CHARACTER", location: "LOCATION", prop: "PROP", acting: "ACTING", prompt_pack: "PROMPT PACK", generation: "GENERATION REQUEST", needs_you: "NEEDS YOU" };

export function FilmNodeCard({ node, project }: { node: CanvasNodeData; project?: ProjectDetail }) {
    const state = node.filmState;
    const resolved = resolveFilmNode(node, project);
    const { shot, asset, contract } = resolved;
    const isShot = node.filmKind === "shot";
    const isActing = node.filmKind === "acting";
    const isPromptPack = node.filmKind === "prompt_pack";
    const isGeneration = node.filmKind === "generation";
    const title = isShot && shot ? filmText(contract.shotCode) || shot.title : asset?.title || node.title;
    const subtitle = isShot && shot
        ? `${formatSeconds(shot.durationMs)} · ${filmText(contract.shotSize) || "景别待定"}`
        : isActing || isPromptPack || isGeneration
          ? `v${resolved.artifact?.objectVersion || 1} · ${resolved.artifact?.status || "draft"}`
          : asset
            ? `v${asset.currentVersion?.version || asset.character?.version || asset.versionCount || 1} · ${statusLabel(asset.status)}`
            : "等待领域事实";
    const details = isShot && shot
        ? [filmText(contract.camera) || filmText(contract.movement), filmText(contract.narrativeIntent) || shot.description]
        : isActing
          ? [filmText(contract.objective), [filmText(contract.obstacle), filmText(contract.tactic)].filter(Boolean).join(" · ")]
          : isPromptPack
            ? [filmText(contract.compiledPrompt) || filmText(contract.sceneContext), [filmText(contract.formatMode), filmText(contract.camera)].filter(Boolean).join(" · ")]
            : isGeneration
              ? [filmText(contract.compiledPrompt), [filmText(contract.mediaType).toUpperCase(), filmText(contract.aspectRatio), formatDuration(contract.durationMs)].filter(Boolean).join(" · ")]
        : asset
          ? assetDetails(node.filmKind, resolved.definition)
          : [];
    const Icon = node.filmKind === "character" ? PersonStanding : node.filmKind === "location" ? MapPin : node.filmKind === "prop" ? Package : node.filmKind === "shot" ? CircleDot : node.filmKind === "acting" ? Drama : node.filmKind === "prompt_pack" ? Sparkles : node.filmKind === "generation" ? Clapperboard : Box;
    const AttentionIcon = state?.attention === "error" || state?.attention === "warning" || state?.attention === "human_required" ? AlertTriangle : CircleDot;

    return (
        <div className="flex h-full w-full flex-col overflow-hidden p-4">
            <div className="flex items-center justify-between gap-2 text-[var(--fs-tiny)] font-semibold text-foreground/42">
                <span className="inline-flex items-center gap-1.5"><Icon className="size-3.5" /> {labels[node.filmKind || ""] || node.filmKind}</span>
                <span className="inline-flex items-center gap-1"><AttentionIcon className={`size-3 ${state?.attention === "error" ? "text-red-400" : state?.attention === "warning" || state?.attention === "human_required" ? "text-amber-400" : "text-emerald-400"}`} /> {state?.evidence || "unknown"}</span>
            </div>
            <div className="mt-2 truncate text-sm font-semibold text-foreground/88" title={title}>{title}</div>
            <div className="mt-1 text-[var(--fs-caption)] text-foreground/48">{subtitle}</div>
            <div className="mt-3 min-h-0 flex-1 space-y-1 overflow-hidden text-[var(--fs-caption)] leading-5 text-foreground/58">
                {details.filter(Boolean).slice(0, 2).map((detail) => <p key={detail} className="line-clamp-1">{detail}</p>)}
            </div>
            <div className="mt-2 flex items-center justify-between gap-2 border-t border-border/60 pt-2 text-[var(--fs-micro)] text-foreground/38">
                <span>{state?.lifecycle || "draft"}</span>
                <span>{state?.production || "not_started"}</span>
            </div>
        </div>
    );
}

function assetDetails(kind: string | undefined, definition: Record<string, unknown>) {
    if (kind === "location") return [filmText(definition.canonicalGeometry), filmText(definition.lightingLogic)];
    if (kind === "prop") return [filmText(definition.appearance), filmText(definition.states)];
    return [filmText(definition.appearance) || filmText(definition.descriptor), filmText(definition.clothing) || filmText(definition.physique)];
}

function formatSeconds(durationMs: number) {
    return `${Math.round(durationMs / 100) / 10} sec`;
}

function formatDuration(value: unknown) {
    const durationMs = Number(value);
    return Number.isFinite(durationMs) && durationMs > 0 ? formatSeconds(durationMs) : "";
}

function statusLabel(status: string) {
    return ({ confirmed: "已确认", draft: "草稿", review: "审核中", archived: "已归档" } as Record<string, string>)[status] || status;
}
