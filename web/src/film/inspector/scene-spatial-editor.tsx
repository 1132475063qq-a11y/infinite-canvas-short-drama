import { useEffect, useMemo, useState } from "react";
import { App, Button } from "antd";
import { AlertTriangle, Camera, CheckCircle2, ClipboardList, Image, LockKeyhole, Map, RefreshCw, ShieldAlert, ShieldCheck } from "lucide-react";

import type { FilmArtifact, ProjectDetail, ProjectScene, SceneAssetPackSaveResult, SceneSpatialGate } from "@/services/api/projects";
import { saveProjectSceneAssetPack } from "@/services/api/projects";
import { SceneSpatialAssetsPanel } from "./scene-spatial-editor-assets";
import { CheckboxField, EditorGroup, EditorItem, SelectField, TextField, TextListField, removeAt, replaceAt } from "./scene-spatial-editor-controls";
import { buildSceneAssetPackPayload, hasLegacyLocationFields, nextSpatialId, parseSceneAssetPackDraft, removeLegacyLocationFields, spatialPanelForPath, type AuthorityConflictDraft, type SceneAssetPackDraft } from "./scene-spatial-editor-model";
import { SceneSpatialCoveragePanel, SceneSpatialStructurePanel } from "./scene-spatial-editor-spatial";

type SpatialPanel = "overview" | "structure" | "coverage" | "assets" | "gate";
type SaveStatus = "draft" | "ready" | "locked";

type SceneSpatialEditorProps = {
    project: ProjectDetail;
    scene: ProjectScene;
    packArtifact?: FilmArtifact;
    gateArtifact?: FilmArtifact;
    gate?: SceneSpatialGate;
    onProjectChanged?: () => Promise<unknown> | void;
};

export function SceneSpatialEditor({ project, scene, packArtifact, gateArtifact, gate, onProjectChanged }: SceneSpatialEditorProps) {
    const { message } = App.useApp();
    const [activePanel, setActivePanel] = useState<SpatialPanel>("overview");
    const [saved, setSaved] = useState<SceneAssetPackSaveResult>();
    const localSave = saved?.packArtifact.sceneId === scene.id && (!packArtifact || saved.packArtifact.objectVersion >= packArtifact.objectVersion) ? saved : undefined;
    const sourceArtifact = localSave?.packArtifact || packArtifact;
    const sourceGate = localSave?.gate || gate;
    const sourceGateArtifact = localSave?.gateArtifact || gateArtifact;
    const sourceKey = `${scene.id}:${sourceArtifact?.id || "new"}:${sourceArtifact?.objectVersion || 0}`;
    const [draft, setDraft] = useState<SceneAssetPackDraft>(() => parseSceneAssetPackDraft(sourceArtifact?.payloadJson, scene));
    const [loadedKey, setLoadedKey] = useState(sourceKey);
    const [dirty, setDirty] = useState(false);
    const [stale, setStale] = useState(false);
    const [saving, setSaving] = useState<SaveStatus>();
    const [refreshing, setRefreshing] = useState(false);

    useEffect(() => {
        if (loadedKey === sourceKey) return;
        setDraft(parseSceneAssetPackDraft(sourceArtifact?.payloadJson, scene));
        setLoadedKey(sourceKey);
        setDirty(false);
    }, [loadedKey, scene, sourceArtifact?.payloadJson, sourceKey]);

    const update = (updater: (current: SceneAssetPackDraft) => SceneAssetPackDraft) => {
        setDraft((current) => updater(current));
        setDirty(true);
    };
    const refresh = async () => {
        if (!onProjectChanged || (dirty && !stale)) return;
        setRefreshing(true);
        try {
            await onProjectChanged();
            if (stale) {
                setSaved(undefined);
                setDirty(false);
                setStale(false);
            }
            message.success(stale ? "已放弃本地修改并载入最新版本" : "已刷新场景空间版本");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "刷新场景空间版本失败");
        } finally {
            setRefreshing(false);
        }
    };
    const save = async (status: SaveStatus) => {
        setSaving(status);
        try {
            const response = await saveProjectSceneAssetPack(project.project.id, scene.id, {
                status,
                expectedVersion: sourceArtifact?.objectVersion || 0,
                payload: buildSceneAssetPackPayload(draft, scene),
            });
            const result = response.sceneAssetPack;
            setSaved(result);
            setDraft(parseSceneAssetPackDraft(result.packArtifact.payloadJson, scene));
            setLoadedKey(`${scene.id}:${result.packArtifact.id}:${result.packArtifact.objectVersion}`);
            setDirty(false);
            setStale(false);
            try {
                await onProjectChanged?.();
            } catch (error) {
                message.warning(error instanceof Error ? `已保存，但刷新画布失败：${error.message}` : "已保存，但刷新画布失败");
            }
            message.success(status === "draft" ? "空间资产包草稿已保存并完成检查" : status === "ready" ? "空间资产包已通过并可用于镜头生成" : "当前通过的空间资产包已锁定");
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : "空间资产包保存失败";
            if (errorMessage.includes("已被其他操作更新")) setStale(true);
            message.error(errorMessage);
        } finally {
            setSaving(undefined);
        }
    };
    const panels = useMemo(
        () => [
            { id: "overview" as const, label: "清单", icon: ClipboardList },
            { id: "structure" as const, label: "空间", icon: Map },
            { id: "coverage" as const, label: "机位", icon: Camera },
            { id: "assets" as const, label: "资产", icon: Image },
            { id: "gate" as const, label: "门禁", icon: ShieldCheck },
        ],
        [],
    );

    return (
        <div className="space-y-4">
            <section className="border-b border-border/70 pb-4">
                <div className="flex items-start justify-between gap-3">
                    <div>
                        <h3 className="text-sm font-semibold text-foreground/85">场景空间资产包</h3>
                        <p className="mt-1 text-[var(--fs-caption)] leading-5 text-foreground/45">场景空间事实与连续性门禁。</p>
                    </div>
                    <div className="flex items-center gap-1">
                        <Button
                            size="small"
                            type="text"
                            icon={<RefreshCw className="size-3.5" />}
                            loading={refreshing}
                            disabled={!onProjectChanged || (dirty && !stale)}
                            title={stale ? "放弃本地修改并载入最新版本" : dirty ? "请先保存当前修改" : "刷新最新空间版本"}
                            aria-label={stale ? "放弃本地修改并载入最新版本" : "刷新最新空间版本"}
                            onClick={() => void refresh()}
                        />
                        <GateBadge status={sourceGate?.status} />
                    </div>
                </div>
                <div className="mt-3 grid grid-cols-2 gap-2 text-[var(--fs-caption)]">
                    <Fact label="空间包" value={sourceArtifact ? `v${sourceArtifact.objectVersion} · ${sourceArtifact.status}` : "尚未建立"} />
                    <Fact label="门禁" value={sourceGateArtifact ? `v${sourceGateArtifact.objectVersion}` : "尚未检查"} />
                </div>
                {stale ? (
                    <p className="mt-3 text-[var(--fs-caption)] text-red-600 dark:text-red-300">服务器已有新版本，请使用刷新按钮放弃本地修改并重新载入。</p>
                ) : dirty ? (
                    <p className="mt-3 text-[var(--fs-caption)] text-amber-600 dark:text-amber-300">当前修改尚未检查或保存。</p>
                ) : null}
            </section>

            <nav className="grid grid-cols-5 gap-1" role="tablist" aria-label="场景空间编辑器">
                {panels.map((panel) => {
                    const Icon = panel.icon;
                    const active = activePanel === panel.id;
                    return (
                        <button
                            key={panel.id}
                            type="button"
                            role="tab"
                            aria-selected={active}
                            className={`flex min-h-9 flex-col items-center justify-center gap-0.5 rounded-md text-[var(--fs-micro)] ${active ? "bg-[var(--workspace-accent-soft)] text-[var(--workspace-accent)]" : "text-foreground/48 hover:bg-foreground/5 hover:text-foreground/75"}`}
                            onClick={() => setActivePanel(panel.id)}
                        >
                            <Icon className="size-3.5" />
                            <span>{panel.label}</span>
                        </button>
                    );
                })}
            </nav>

            {activePanel === "overview" ? <SceneManifestPanel draft={draft} update={update} /> : null}
            {activePanel === "structure" ? <SceneSpatialStructurePanel draft={draft} sceneId={scene.id} update={update} /> : null}
            {activePanel === "coverage" ? <SceneSpatialCoveragePanel draft={draft} sceneId={scene.id} update={update} /> : null}
            {activePanel === "assets" ? <SceneSpatialAssetsPanel draft={draft} assets={project.assets} update={update} /> : null}
            {activePanel === "gate" ? <SceneSpatialGatePanel draft={draft} gate={sourceGate} onJump={(path) => setActivePanel(spatialPanelForPath(path))} onUpdate={update} /> : null}

            <section className="border-t border-border/70 pt-4">
                <div className="grid gap-2">
                    <Button block icon={<CheckCircle2 className="size-3.5" />} loading={saving === "draft"} disabled={Boolean(saving)} onClick={() => void save("draft")}>
                        保存草稿并检查
                    </Button>
                    <Button block type="primary" icon={<ShieldCheck className="size-3.5" />} loading={saving === "ready"} disabled={Boolean(saving)} onClick={() => void save("ready")}>
                        保存为可生成版本
                    </Button>
                    <Button
                        block
                        icon={<LockKeyhole className="size-3.5" />}
                        loading={saving === "locked"}
                        disabled={Boolean(saving) || dirty || sourceGate?.status !== "PASS"}
                        title={dirty ? "请先保存当前修改" : sourceGate?.status !== "PASS" ? "仅可锁定已通过的空间包" : "锁定当前版本"}
                        onClick={() => void save("locked")}
                    >
                        锁定当前通过版本
                    </Button>
                </div>
            </section>
        </div>
    );
}

function SceneManifestPanel({ draft, update }: { draft: SceneAssetPackDraft; update: (updater: (current: SceneAssetPackDraft) => SceneAssetPackDraft) => void }) {
    const manifest = draft.sceneManifest;
    const setManifest = (patch: Partial<SceneAssetPackDraft["sceneManifest"]>) => update((current) => ({ ...current, sceneManifest: { ...current.sceneManifest, ...patch } }));
    return (
        <>
            <EditorGroup title="SceneManifest" detail="从剧情拆出真正会经过的空间、事件和出入口；这是所有后续空间事实的来源。">
                <div className="grid grid-cols-2 gap-2">
                    <TextField label="场景名称" value={manifest.sceneName} onChange={(sceneName) => setManifest({ sceneName })} />
                    <TextField label="父级地点" value={manifest.parentLocation} onChange={(parentLocation) => setManifest({ parentLocation })} />
                    <SelectField
                        label="内 / 外景"
                        value={manifest.interiorExterior}
                        options={[
                            { value: "", label: "待确认" },
                            { value: "interior", label: "内景" },
                            { value: "exterior", label: "外景" },
                            { value: "mixed", label: "混合" },
                        ]}
                        onChange={(interiorExterior) => setManifest({ interiorExterior })}
                    />
                    <TextField label="连续性优先级" value={manifest.continuityPriority} onChange={(continuityPriority) => setManifest({ continuityPriority })} />
                    <TextField label="前一场" value={manifest.previousScene} onChange={(previousScene) => setManifest({ previousScene })} />
                    <TextField label="后一场" value={manifest.nextScene} onChange={(nextScene) => setManifest({ nextScene })} />
                </div>
                <TextListField label="剧情事件" values={manifest.storyEvents} onChange={(storyEvents) => setManifest({ storyEvents })} placeholder="每行一个本场发生的剧情事件" />
                <TextListField label="角色" values={manifest.characters} onChange={(characters) => setManifest({ characters })} />
                <TextListField label="必需道具" values={manifest.requiredProps} onChange={(requiredProps) => setManifest({ requiredProps })} />
                <div className="grid grid-cols-2 gap-2">
                    <TextListField label="入口" values={manifest.entrances} onChange={(entrances) => setManifest({ entrances })} />
                    <TextListField label="出口" values={manifest.exits} onChange={(exits) => setManifest({ exits })} />
                </div>
                <TextField label="Scene Manifest 来源" value={draft.sceneManifestRef} onChange={(sceneManifestRef) => update((current) => ({ ...current, sceneManifestRef }))} placeholder="例如 production-breakdown artifact ID" />
            </EditorGroup>
            <EditorGroup title="禁止变化" detail="填入在后续图像/视频生成中不得重新设计的空间事实。">
                <TextListField label="连续性约束" values={draft.forbiddenChanges} onChange={(forbiddenChanges) => update((current) => ({ ...current, forbiddenChanges }))} placeholder="例如：诊所门始终位于右后方" />
            </EditorGroup>
        </>
    );
}

function SceneSpatialGatePanel({ draft, gate, onJump, onUpdate }: { draft: SceneAssetPackDraft; gate?: SceneSpatialGate; onJump: (path?: string) => void; onUpdate: (updater: (current: SceneAssetPackDraft) => SceneAssetPackDraft) => void }) {
    const legacy = hasLegacyLocationFields(draft);
    return (
        <>
            <EditorGroup title="Spatial Continuity Gate" detail="保存草稿后由服务端重新校验；只有 PASS 才能进入正式镜头生成。">
                <div className="flex items-center justify-between gap-3">
                    <GateBadge status={gate?.status} />
                    <span className="text-[var(--fs-caption)] text-foreground/45">{gate?.validatedAt ? `最近校验：${new Date(gate.validatedAt).toLocaleString()}` : "尚未保存检查"}</span>
                </div>
                {gate?.issues?.length ? (
                    <div className="mt-3">
                        {gate.issues.map((issue, index) => (
                            <div key={`${issue.code}-${issue.path}-${index}`} className="border-t border-border/60 py-3 first:border-t-0 first:pt-0">
                                <div className="flex items-start justify-between gap-2">
                                    <div className="min-w-0">
                                        <div className="flex items-center gap-1.5">
                                            <IssueIcon severity={issue.severity} />
                                            <span className="text-[var(--fs-caption)] font-semibold text-foreground/75">{issue.code}</span>
                                        </div>
                                        <p className="mt-1 text-[var(--fs-caption)] leading-5 text-foreground/60">{issue.message}</p>
                                        {issue.path ? <p className="mt-1 font-mono text-[var(--fs-micro)] text-foreground/35">{issue.path}</p> : null}
                                    </div>
                                    <Button size="small" type="text" onClick={() => onJump(issue.path)}>
                                        定位修复
                                    </Button>
                                </div>
                            </div>
                        ))}
                    </div>
                ) : (
                    <p className="mt-3 text-[var(--fs-caption)] leading-5 text-foreground/45">保存一次草稿后，问题会按合同路径显示在这里。</p>
                )}
                {legacy ? (
                    <Button className="mt-3" size="small" danger onClick={() => onUpdate((current) => removeLegacyLocationFields(current))}>
                        删除旧版自由文本字段
                    </Button>
                ) : null}
            </EditorGroup>
            <EditorGroup
                title="权威冲突"
                detail="这里列出的冲突会把 Gate 置为 NEEDS_YOU。人工裁定后，请移除对应条目并重新保存。"
                onAdd={() => {
                    const field = nextSpatialId(
                        "authority",
                        draft.authorityConflicts.map((conflict) => conflict.field),
                    );
                    onUpdate((current) => ({ ...current, authorityConflicts: [...current.authorityConflicts, { field, sources: [], description: "" }] }));
                }}
                addLabel="记录冲突"
            >
                {draft.authorityConflicts.map((conflict, index) => (
                    <AuthorityConflictEditor
                        key={`${conflict.field}-${index}`}
                        conflict={conflict}
                        index={index}
                        onChange={(next) => onUpdate((current) => ({ ...current, authorityConflicts: replaceAt(current.authorityConflicts, index, next) }))}
                        onRemove={() => onUpdate((current) => ({ ...current, authorityConflicts: removeAt(current.authorityConflicts, index) }))}
                    />
                ))}
                {!draft.authorityConflicts.length ? <p className="py-1 text-[var(--fs-caption)] text-foreground/42">没有待人工裁定的空间来源冲突。</p> : null}
            </EditorGroup>
        </>
    );
}

function AuthorityConflictEditor({ conflict, index, onChange, onRemove }: { conflict: AuthorityConflictDraft; index: number; onChange: (conflict: AuthorityConflictDraft) => void; onRemove: () => void }) {
    return (
        <EditorItem title={conflict.field || `冲突 ${index + 1}`} onRemove={onRemove}>
            <TextField label="冲突字段" value={conflict.field} onChange={(field) => onChange({ ...conflict, field })} />
            <TextListField label="相互冲突的来源" values={conflict.sources} onChange={(sources) => onChange({ ...conflict, sources })} />
            <TextField label="待裁定说明" value={conflict.description} multiline onChange={(description) => onChange({ ...conflict, description })} />
        </EditorItem>
    );
}

function GateBadge({ status }: { status: SceneSpatialGate["status"] | undefined }) {
    const value = status || "UNCERTAIN";
    const tone =
        value === "PASS"
            ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
            : value === "FAIL"
              ? "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300"
              : value === "NEEDS_YOU"
                ? "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300"
                : "border-foreground/15 bg-foreground/5 text-foreground/60";
    return (
        <span className={`inline-flex items-center gap-1 rounded-md border px-2 py-1 text-[var(--fs-caption)] font-semibold ${tone}`}>
            {value === "PASS" ? <CheckCircle2 className="size-3.5" /> : value === "FAIL" || value === "NEEDS_YOU" ? <ShieldAlert className="size-3.5" /> : <AlertTriangle className="size-3.5" />}
            {value}
        </span>
    );
}

function IssueIcon({ severity }: { severity: string }) {
    return severity === "ERROR" || severity === "BLOCKER" ? <ShieldAlert className="size-3.5 text-red-500" /> : <AlertTriangle className="size-3.5 text-amber-500" />;
}
function Fact({ label, value }: { label: string; value: string }) {
    return (
        <div>
            <span className="block text-[var(--fs-micro)] text-foreground/40">{label}</span>
            <span className="mt-0.5 block break-all text-foreground/70">{value}</span>
        </div>
    );
}
