import { useEffect, useMemo, useState, type ReactNode } from "react";
import { App, Button } from "antd";
import { AlertTriangle, Box, CheckCircle2, CircleDot, FileQuestion, Save, ShieldCheck } from "lucide-react";

import { filmText, resolveFilmNode } from "@/film/domain/film-node-resolver";
import type { FilmNodeKind, FilmNodeState } from "@/film/domain/types";
import type { ProductionNavigationKey } from "@/film/panels/production-shell-model";
import { createProjectAssetVersion, getProjectFilmGenerationProviderRoutes, getProjectFilmGenerationTaskDraft, saveProjectFilmArtifact, saveProjectScene, saveProjectShot, updateProjectCharacter, type FilmGenerationProviderRouteCatalog, type FilmGenerationTaskDraft, type ProjectDetail } from "@/services/api/projects";
import type { CanvasNodeData } from "@/types/canvas";

type FilmInspectorProps = {
    node: CanvasNodeData | null;
    project?: ProjectDetail;
    activeSection?: ProductionNavigationKey;
    onProjectChanged?: () => Promise<unknown> | void;
    onProjectionChanged?: (nodeId: string, patch: Pick<CanvasNodeData, "title"> & { domainRef?: CanvasNodeData["domainRef"] }) => void;
};

const KIND_LABELS: Record<FilmNodeKind, string> = {
    project: "项目", story: "故事", script: "剧本", scene: "场景", character: "角色", character_state: "角色状态", location: "场地", location_view: "场地视图", prop: "道具", voice_profile: "声音档案", acting: "表演", acting_profile: "表演档案", scene_acting: "场景表演", storyboard: "分镜", shot: "镜头", shot_reference_pack: "镜头参考包", prompt_pack: "提示词包", generation: "生成任务", generation_attempt: "生成尝试", result: "生成结果", qc: "质量检查", continuity: "连续性", retry: "重试", needs_you: "需要人工决定", agent_task: "Agent 任务", delivery: "交付",
};

const TABS: Partial<Record<FilmNodeKind, readonly string[]>> = {
    scene: ["Overview", "Definition", "Shots", "Versions"],
    shot: ["Overview", "Script", "Blocking", "Acting", "References", "Camera", "Prompt", "Generation", "QC", "Versions"],
    character: ["Definition", "Face", "Body", "Costume", "Views", "States", "Descriptor", "Voice", "Acting Profile", "Versions", "Usage"],
    location: ["Definition", "Geometry", "Views", "Lighting", "Versions", "Usage"],
    prop: ["Definition", "States", "Continuity", "Versions", "Usage"],
    acting: ["Overview", "Beats", "Continuity", "Versions"],
    prompt_pack: ["Overview", "References", "Camera", "Timing", "Audio", "Locks", "Compiled", "Versions"],
    generation: ["Overview", "Request", "References", "Execution", "Versions"],
};
const DEFAULT_TABS = ["Overview", "References", "Production", "Versions"] as const;

export function FilmInspector({ node, project, activeSection = "overview", onProjectChanged, onProjectionChanged }: FilmInspectorProps) {
    const tabs = useMemo(() => (node?.filmKind && TABS[node.filmKind]) || DEFAULT_TABS, [node?.filmKind]);
    const [activeTab, setActiveTab] = useState<string>(tabs[0]);
    useEffect(() => setActiveTab(tabs[0]), [node?.id, tabs]);

    return (
        <aside className="relative z-[var(--z-panel)] hidden w-72 shrink-0 flex-col border-l border-border bg-background/94 backdrop-blur-xl lg:flex 2xl:w-80" aria-label="影视生产检查器">
            {node?.filmKind ? <NodeInspector node={node} project={project} tabs={tabs} activeTab={activeTab} onTabChange={setActiveTab} onProjectChanged={onProjectChanged} onProjectionChanged={onProjectionChanged} /> : <ProjectInspector project={project} activeSection={activeSection} />}
        </aside>
    );
}

function ProjectInspector({ project, activeSection }: { project?: ProjectDetail; activeSection: ProductionNavigationKey }) {
    return <><InspectorHeader eyebrow="PROJECT INSPECTOR" title={project?.project.name || "项目检查器"} subtitle={`当前区域 · ${navigationLabel(activeSection)}`} /><div className="thin-scrollbar min-h-0 flex-1 overflow-y-auto p-4"><InspectorSection title="项目事实源"><InspectorRow label="Project ID" value={project?.project.id || "未关联"} /><InspectorRow label="状态" value={project?.project.status || "unknown"} /><InspectorRow label="版本" value={project?.project.revision ? `r${project.project.revision}` : "未记录"} /></InspectorSection><div className="grid grid-cols-3 gap-2"><ProjectMetric label="场景" value={project?.scenes?.length || 0} /><ProjectMetric label="镜头" value={project?.shots?.length || 0} /><ProjectMetric label="资产" value={project?.assets?.length || 0} /></div><EmptyHint text="选择影视节点后，可直接编辑后端生产事实。" /></div></>;
}

function NodeInspector({ node, project, tabs, activeTab, onTabChange, onProjectChanged, onProjectionChanged }: { node: CanvasNodeData; project?: ProjectDetail; tabs: readonly string[]; activeTab: string; onTabChange: (tab: string) => void; onProjectChanged?: () => Promise<unknown> | void; onProjectionChanged?: FilmInspectorProps["onProjectionChanged"] }) {
    const resolved = resolveFilmNode(node, project);
    const artifactScoped = node.filmKind === "acting" || node.filmKind === "prompt_pack" || node.filmKind === "generation";
    const objectId = artifactScoped
        ? node.domainRef?.artifactId || resolved.artifact?.id || node.domainRef?.shotId || node.id
        : node.domainRef?.shotId || node.domainRef?.assetId || node.domainRef?.sceneId || node.id;
    const title = node.filmKind === "shot"
        ? resolved.shot?.title || node.title
        : artifactScoped
            ? node.title
            : resolved.asset?.title || resolved.scene?.title || node.title;
    return <><InspectorHeader eyebrow={KIND_LABELS[node.filmKind!] || node.filmKind!} title={title} subtitle={objectId} attention={node.filmState?.attention} /><TabBar tabs={tabs} activeTab={activeTab} onTabChange={onTabChange} /><div className="thin-scrollbar min-h-0 flex-1 overflow-y-auto p-4"><FilmFactEditor node={node} project={project} activeTab={activeTab} onProjectChanged={onProjectChanged} onProjectionChanged={onProjectionChanged} /></div></>;
}

function FilmFactEditor({ node, project, activeTab, onProjectChanged, onProjectionChanged }: { node: CanvasNodeData; project?: ProjectDetail; activeTab: string; onProjectChanged?: FilmInspectorProps["onProjectChanged"]; onProjectionChanged?: FilmInspectorProps["onProjectionChanged"] }) {
    const { message } = App.useApp();
    const resolved = resolveFilmNode(node, project);
    const [draft, setDraft] = useState<Record<string, string>>({});
    const [saving, setSaving] = useState(false);
    const [taskDraft, setTaskDraft] = useState<FilmGenerationTaskDraft | null>(null);
    const [loadingTaskDraft, setLoadingTaskDraft] = useState(false);
    const [providerRoutes, setProviderRoutes] = useState<FilmGenerationProviderRouteCatalog | null>(null);
    const [loadingProviderRoutes, setLoadingProviderRoutes] = useState(false);
    const editable = ["scene", "shot", "character", "location", "prop", "acting", "prompt_pack", "generation"].includes(node.filmKind || "");

    useEffect(() => {
        setDraft(buildDraft(node, project));
        setTaskDraft(null);
        setProviderRoutes(null);
    }, [node.id, node.filmKind, node.domainRef?.artifactId, project]);
    const set = (key: string, value: string) => setDraft((current) => ({ ...current, [key]: value }));
    const refreshGenerationPrompt = () => {
        if (node.filmKind !== "generation") return;
        const promptPack = resolveFilmNode({ ...node, filmKind: "prompt_pack" }, project);
        const compiledPrompt = filmText(promptPack.contract.compiledPrompt);
        if (!promptPack.artifact || !compiledPrompt) {
            message.info("请先在 Prompt Pack 中填写并保存最终编译提示词");
            return;
        }
        setDraft((current) => ({
            ...current,
            promptArtifactId: promptPack.artifact!.id,
            promptArtifactVersion: String(promptPack.artifact!.objectVersion),
            compiledPrompt,
        }));
        message.info("已载入最新 Prompt Pack；保存后会创建新的 Generation Request 版本");
    };

    const inspectGenerationTaskDraft = async () => {
        if (!project || node.filmKind !== "generation") return;
        const artifactId = node.domainRef?.artifactId || resolved.artifact?.id;
        if (!artifactId) {
            message.info("请先保存 Generation Request，再查看任务合同");
            return;
        }
        setLoadingTaskDraft(true);
        try {
            const response = await getProjectFilmGenerationTaskDraft(project.project.id, artifactId);
            setTaskDraft(response.taskDraft);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取任务合同失败");
        } finally {
            setLoadingTaskDraft(false);
        }
    };

    const inspectGenerationProviderRoutes = async () => {
        if (!project || node.filmKind !== "generation") return;
        const artifactId = node.domainRef?.artifactId || resolved.artifact?.id;
        if (!artifactId) {
            message.info("请先保存 Generation Request，再查看可用渠道");
            return;
        }
        setLoadingProviderRoutes(true);
        try {
            const response = await getProjectFilmGenerationProviderRoutes(project.project.id, artifactId);
            setProviderRoutes(response.providerRoutes);
        } catch (error) {
            message.error(error instanceof Error ? error.message : "读取可用渠道失败");
        } finally {
            setLoadingProviderRoutes(false);
        }
    };

    const save = async () => {
        if (!project || !node.filmKind) return;
        setSaving(true);
        try {
            let persistedDraft: Record<string, string> | undefined;
            if (node.filmKind === "scene" && resolved.scene) {
                const response = await saveProjectScene(project.project.id, { ...resolved.scene, title: draft.title, description: draft.description, interiorExterior: draft.interiorExterior, timeOfDay: draft.timeOfDay, locationAssetId: draft.locationAssetId });
                onProjectionChanged?.(node.id, { title: `${response.scene.code} · ${response.scene.title}` });
                persistedDraft = { ...draft, title: response.scene.title };
            } else if (node.filmKind === "shot" && resolved.shot) {
                const response = await saveProjectShot(project.project.id, { ...resolved.shot, title: draft.title, description: draft.description, durationMs: secondsToMs(draft.duration), contract: contractFromDraft(draft) });
                onProjectionChanged?.(node.id, { title: `${response.shot.title} · Shot Contract`, domainRef: { ...node.domainRef!, artifactId: response.shot.contractArtifactId, artifactVersion: String(response.shot.contractVersion) } });
                persistedDraft = { ...draft, title: response.shot.title, duration: String(response.shot.durationMs / 1000), description: response.shot.description };
            } else if (node.filmKind === "character" && resolved.asset) {
                const response = await updateProjectCharacter(project.project.id, resolved.asset.id, { name: draft.title, definition: definitionFromDraft(node.filmKind, draft, resolved.definition) });
                onProjectionChanged?.(node.id, { title: response.asset.title, domainRef: { ...node.domainRef!, assetVersionId: response.character.versionId } });
                persistedDraft = { ...draft, title: response.asset.title };
            } else if ((node.filmKind === "location" || node.filmKind === "prop") && resolved.asset) {
                const response = await createProjectAssetVersion(project.project.id, resolved.asset.id, { title: draft.title, definitionJson: JSON.stringify(definitionFromDraft(node.filmKind, draft, resolved.definition)), note: "由影视节点 Inspector 更新" });
                onProjectionChanged?.(node.id, { title: draft.title, domainRef: { ...node.domainRef!, assetVersionId: response.version.id } });
                persistedDraft = { ...draft };
            } else if ((node.filmKind === "acting" || node.filmKind === "prompt_pack" || node.filmKind === "generation") && resolved.shot) {
                const artifactType = node.filmKind === "acting" ? "acting" : node.filmKind === "prompt_pack" ? "video_prompt_pack" : "generation_request";
                const payload = node.filmKind === "generation" ? generationRequestFromDraft(draft) : artifactFromDraft(node.filmKind, draft, resolved.contract);
                const status = node.filmKind === "generation" ? draft.generationStatus || "draft" : "draft";
                const response = await saveProjectFilmArtifact(project.project.id, { shotId: resolved.shot.id, artifactType, status, payload });
                const artifactLabel = node.filmKind === "acting" ? "Acting" : node.filmKind === "prompt_pack" ? "Prompt Pack" : "Generation Request";
                const title = `${filmText(resolveFilmNode({ ...node, filmKind: "shot" }, project).contract.shotCode) || resolved.shot.title} · ${artifactLabel}`;
                onProjectionChanged?.(node.id, { title, domainRef: { ...node.domainRef!, artifactId: response.artifact.id, artifactVersion: String(response.artifact.objectVersion) } });
                persistedDraft = { ...draft };
            }
            await onProjectChanged?.();
            if (persistedDraft) setDraft(persistedDraft);
            message.success("生产事实已保存并生成新版本");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "保存失败");
        } finally {
            setSaving(false);
        }
    };

    return <>
        <ObjectHeader node={node} project={project} />
        {activeTab === "Overview" || activeTab === "Definition" ? <PrimaryEditor kind={node.filmKind!} draft={draft} set={set} project={project} onRefreshGenerationPrompt={refreshGenerationPrompt} /> : <SpecializedTab kind={node.filmKind!} tab={activeTab} draft={draft} set={set} resolved={resolved} project={project} taskId={node.domainRef?.taskId} generationTaskDraft={taskDraft} loadingGenerationTaskDraft={loadingTaskDraft} onInspectGenerationTaskDraft={inspectGenerationTaskDraft} providerRoutes={providerRoutes} loadingProviderRoutes={loadingProviderRoutes} onInspectGenerationProviderRoutes={inspectGenerationProviderRoutes} />}
        {editable ? <Button block type="primary" icon={<Save className="size-3.5" />} loading={saving} onClick={() => void save()}>保存到事实源</Button> : <EmptyHint text="该节点的事实编辑将在对应生产阶段接入。" />}
    </>;
}

function ObjectHeader({ node, project }: { node: CanvasNodeData; project?: ProjectDetail }) {
    const resolved = resolveFilmNode(node, project);
    const ref = node.domainRef;
    const artifactScoped = node.filmKind === "acting" || node.filmKind === "prompt_pack" || node.filmKind === "generation";
    const objectId = artifactScoped ? ref?.artifactId || resolved.artifact?.id || ref?.shotId || node.id : ref?.shotId || ref?.assetId || ref?.sceneId || node.id;
    return <InspectorSection title="统一对象头"><InspectorRow label="Object ID" value={objectId} /><InspectorRow label="Version" value={ref?.artifactVersion || resolved.artifact?.objectVersion?.toString() || resolved.asset?.currentVersion?.version?.toString() || resolved.asset?.character?.version?.toString() || "未固定"} /><InspectorRow label="Status" value={`${node.filmState?.lifecycle || "draft"} / ${node.filmState?.production || "not_started"}`} /><InspectorRow label="Evidence" value={node.filmState?.evidence || "unknown"} /></InspectorSection>;
}

function PrimaryEditor({ kind, draft, set, project, onRefreshGenerationPrompt }: { kind: FilmNodeKind; draft: Record<string, string>; set: (key: string, value: string) => void; project?: ProjectDetail; onRefreshGenerationPrompt?: () => void }) {
    if (kind === "scene") return <InspectorSection title="场次定义"><FormField label="场次编号" value={draft.code} onChange={(value) => set("code", value)} disabled /><FormField label="场景名称" value={draft.title} onChange={(value) => set("title", value)} /><FormSelect label="内 / 外景" value={draft.interiorExterior} onChange={(value) => set("interiorExterior", value)} options={[{"value":"","label":"待确定"},{"value":"interior","label":"内景"},{"value":"exterior","label":"外景"},{"value":"mixed","label":"混合"}]} /><FormSelect label="时间" value={draft.timeOfDay} onChange={(value) => set("timeOfDay", value)} options={[{"value":"","label":"待确定"},{"value":"day","label":"日"},{"value":"night","label":"夜"},{"value":"dawn","label":"黎明"},{"value":"dusk","label":"黄昏"},{"value":"continuous","label":"连续"}]} /><FormSelect label="场地资产" value={draft.locationAssetId} onChange={(value) => set("locationAssetId", value)} options={[{"value":"","label":"未绑定"}, ...(project?.assets.filter((asset) => asset.category === "environment").map((asset) => ({ value: asset.id, label: asset.title })) || [])]} /><FormField label="场次说明" value={draft.description} onChange={(value) => set("description", value)} multiline /></InspectorSection>;
    if (kind === "shot") return <InspectorSection title="Shot Contract"><FormField label="镜头编号" value={draft.title} onChange={(value) => set("title", value)} /><FormField label="时长（秒）" value={draft.duration} onChange={(value) => set("duration", value)} type="number" /><FormField label="景别" value={draft.shotSize} onChange={(value) => set("shotSize", value)} placeholder="Medium Close Up" /><FormField label="叙事意图" value={draft.narrativeIntent} onChange={(value) => set("narrativeIntent", value)} multiline /><FormField label="镜头说明" value={draft.description} onChange={(value) => set("description", value)} multiline /></InspectorSection>;
    if (kind === "character") return <InspectorSection title="角色定义"><FormField label="角色名称" value={draft.title} onChange={(value) => set("title", value)} /><FormField label="外貌" value={draft.appearance} onChange={(value) => set("appearance", value)} multiline /><FormField label="体型" value={draft.physique} onChange={(value) => set("physique", value)} /><FormField label="服装" value={draft.clothing} onChange={(value) => set("clothing", value)} multiline /><FormField label="稳定描述词" value={draft.descriptor} onChange={(value) => set("descriptor", value)} multiline /></InspectorSection>;
    if (kind === "location") return <InspectorSection title="Location Definition"><FormField label="场地名称" value={draft.title} onChange={(value) => set("title", value)} /><FormField label="规范几何" value={draft.canonicalGeometry} onChange={(value) => set("canonicalGeometry", value)} multiline /><FormField label="门" value={draft.doors} onChange={(value) => set("doors", value)} /><FormField label="窗" value={draft.windows} onChange={(value) => set("windows", value)} /><FormField label="锚点" value={draft.anchors} onChange={(value) => set("anchors", value)} multiline /><FormField label="固定道具" value={draft.fixedProps} onChange={(value) => set("fixedProps", value)} /><FormField label="可移动道具" value={draft.movableProps} onChange={(value) => set("movableProps", value)} /><FormField label="灯光逻辑" value={draft.lightingLogic} onChange={(value) => set("lightingLogic", value)} multiline /></InspectorSection>;
    if (kind === "prop") return <InspectorSection title="道具定义"><FormField label="道具名称" value={draft.title} onChange={(value) => set("title", value)} /><FormField label="外观" value={draft.appearance} onChange={(value) => set("appearance", value)} multiline /><FormField label="剧情功能" value={draft.function} onChange={(value) => set("function", value)} multiline /><FormField label="状态变化" value={draft.states} onChange={(value) => set("states", value)} multiline /><FormField label="连续性锁" value={draft.continuityLocks} onChange={(value) => set("continuityLocks", value)} multiline /></InspectorSection>;
    if (kind === "acting") return <InspectorSection title="Acting Direction"><FormField label="目标 Objective" value={draft.objective} onChange={(value) => set("objective", value)} multiline /><FormField label="阻碍 Obstacle" value={draft.obstacle} onChange={(value) => set("obstacle", value)} multiline /><FormField label="策略 Tactic" value={draft.tactic} onChange={(value) => set("tactic", value)} multiline /><FormField label="节拍 Beat" value={draft.beat} onChange={(value) => set("beat", value)} /></InspectorSection>;
    if (kind === "prompt_pack") return <InspectorSection title="CINEDANCE Prompt Pack"><FormField label="场景上下文" value={draft.sceneContext} onChange={(value) => set("sceneContext", value)} multiline /><FormField label="格式模式" value={draft.formatMode} onChange={(value) => set("formatMode", value)} placeholder="multi-shot / single-shot" /><FormField label="风格" value={draft.style} onChange={(value) => set("style", value)} multiline /><FormField label="编译提示词" value={draft.compiledPrompt} onChange={(value) => set("compiledPrompt", value)} multiline /></InspectorSection>;
    if (kind === "generation") return <InspectorSection title="Generation Request"><FormSelect label="请求状态" value={draft.generationStatus} onChange={(value) => set("generationStatus", value)} options={[{ value: "draft", label: "草稿" }, { value: "review", label: "待评审" }, { value: "ready", label: "就绪" }, { value: "locked", label: "锁定" }]} /><FormSelect label="媒体类型" value={draft.mediaType} onChange={(value) => set("mediaType", value)} options={[{ value: "video", label: "视频" }, { value: "image", label: "图片" }, { value: "audio", label: "音频" }]} /><FormField label="画幅比例（图片 / 视频）" value={draft.aspectRatio} onChange={(value) => set("aspectRatio", value)} placeholder="9:16" /><FormField label="时长（秒，仅视频）" value={draft.generationDuration} onChange={(value) => set("generationDuration", value)} type="number" /><FormField label="输出用途" value={draft.outputIntent} onChange={(value) => set("outputIntent", value)} placeholder="首轮镜头生成" /><FormField label="Prompt Pack ID" value={draft.promptArtifactId} onChange={(value) => set("promptArtifactId", value)} disabled /><FormField label="已编译提示词（由 Prompt Pack 冻结）" value={draft.compiledPrompt} onChange={(value) => set("compiledPrompt", value)} multiline disabled /><Button block onClick={onRefreshGenerationPrompt}>载入最新 Prompt Pack</Button></InspectorSection>;
    return <EmptyHint text="当前节点暂无定义编辑器。" />;
}

function SpecializedTab({ kind, tab, draft, set, resolved, project, taskId, generationTaskDraft, loadingGenerationTaskDraft, onInspectGenerationTaskDraft, providerRoutes, loadingProviderRoutes, onInspectGenerationProviderRoutes }: { kind: FilmNodeKind; tab: string; draft: Record<string, string>; set: (key: string, value: string) => void; resolved: ReturnType<typeof resolveFilmNode>; project?: ProjectDetail; taskId?: string; generationTaskDraft?: FilmGenerationTaskDraft | null; loadingGenerationTaskDraft?: boolean; onInspectGenerationTaskDraft?: () => void; providerRoutes?: FilmGenerationProviderRouteCatalog | null; loadingProviderRoutes?: boolean; onInspectGenerationProviderRoutes?: () => void }) {
    if (kind === "shot" && tab === "Script") return <InspectorSection title="镜头剧本"><FormField label="台词 / 行动" value={draft.script} onChange={(value) => set("script", value)} multiline /></InspectorSection>;
    if (kind === "shot" && tab === "Blocking") return <InspectorSection title="调度"><FormField label="走位与空间" value={draft.blocking} onChange={(value) => set("blocking", value)} multiline /><FormField label="连续性" value={draft.continuity} onChange={(value) => set("continuity", value)} multiline /></InspectorSection>;
    if (kind === "shot" && tab === "Acting") return <InspectorSection title="表演要求"><FormField label="表演" value={draft.acting} onChange={(value) => set("acting", value)} multiline /></InspectorSection>;
    if (kind === "shot" && tab === "Camera") return <InspectorSection title="摄影机"><FormField label="镜头" value={draft.camera} onChange={(value) => set("camera", value)} multiline /><FormField label="焦段" value={draft.lens} onChange={(value) => set("lens", value)} /><FormField label="运镜" value={draft.movement} onChange={(value) => set("movement", value)} /></InspectorSection>;
    if (kind === "acting" && tab === "Beats") return <InspectorSection title="表演节拍"><FormField label="行为说明" value={draft.performance} onChange={(value) => set("performance", value)} multiline /><FormField label="节拍" value={draft.beat} onChange={(value) => set("beat", value)} multiline /></InspectorSection>;
    if (kind === "acting" && tab === "Continuity") return <InspectorSection title="表演连续性"><FormField label="必须保持" value={draft.continuityLocks} onChange={(value) => set("continuityLocks", value)} multiline /></InspectorSection>;
    if (kind === "prompt_pack" && tab === "References") return <InspectorSection title="Active References"><FormField label="活动参考" value={draft.activeReferences} onChange={(value) => set("activeReferences", value)} multiline /><FormField label="场地地图" value={draft.locationMap} onChange={(value) => set("locationMap", value)} multiline /><FormField label="首帧" value={draft.firstFrame} onChange={(value) => set("firstFrame", value)} multiline /></InspectorSection>;
    if (kind === "prompt_pack" && tab === "Camera") return <InspectorSection title="空间与摄影"><FormField label="调度" value={draft.blocking} onChange={(value) => set("blocking", value)} multiline /><FormField label="身体朝向" value={draft.bodyOrientation} onChange={(value) => set("bodyOrientation", value)} multiline /><FormField label="视线" value={draft.gaze} onChange={(value) => set("gaze", value)} multiline /><FormField label="空间锚点" value={draft.anchors} onChange={(value) => set("anchors", value)} multiline /><FormField label="光学" value={draft.optics} onChange={(value) => set("optics", value)} /><FormField label="摄影机" value={draft.camera} onChange={(value) => set("camera", value)} multiline /></InspectorSection>;
    if (kind === "prompt_pack" && tab === "Timing") return <InspectorSection title="动作时间轴"><FormField label="分时动作节拍" value={draft.timedActionBeats} onChange={(value) => set("timedActionBeats", value)} multiline /><FormField label="物理约束" value={draft.physics} onChange={(value) => set("physics", value)} multiline /><FormField label="灯光" value={draft.lighting} onChange={(value) => set("lighting", value)} multiline /></InspectorSection>;
    if (kind === "prompt_pack" && tab === "Audio") return <InspectorSection title="声音与表演"><FormField label="声音" value={draft.audio} onChange={(value) => set("audio", value)} multiline /><FormField label="表演" value={draft.acting} onChange={(value) => set("acting", value)} multiline /></InspectorSection>;
    if (kind === "prompt_pack" && tab === "Locks") return <InspectorSection title="约束锁"><FormField label="正向锁（每行一项）" value={draft.positiveLocks} onChange={(value) => set("positiveLocks", value)} multiline /><FormField label="局部失败锁（每行一项）" value={draft.localFailureLocks} onChange={(value) => set("localFailureLocks", value)} multiline /><FormField label="负向约束（每行一项）" value={draft.emittedNegativeConstraints} onChange={(value) => set("emittedNegativeConstraints", value)} multiline /></InspectorSection>;
    if (kind === "prompt_pack" && tab === "Compiled") return <InspectorSection title="Compiled Prompt"><FormField label="最终提示词" value={draft.compiledPrompt} onChange={(value) => set("compiledPrompt", value)} multiline /></InspectorSection>;
    if (kind === "generation" && tab === "Request") return <InspectorSection title="冻结请求快照"><FormField label="已编译提示词" value={draft.compiledPrompt} onChange={(value) => set("compiledPrompt", value)} multiline disabled /><InspectorRow label="输出用途" value={draft.outputIntent || "未说明"} /></InspectorSection>;
    if (kind === "generation" && tab === "References") return <InspectorSection title="来源版本"><InspectorRow label="Prompt Pack" value={draft.promptArtifactId || "未记录"} /><InspectorRow label="Prompt Pack 版本" value={draft.promptArtifactVersion ? `v${draft.promptArtifactVersion}` : "未记录"} /><InspectorRow label="镜头资产" value="随请求快照写入 SourceRefs" /></InspectorSection>;
    if (kind === "generation" && tab === "Execution") return <InspectorSection title="执行状态"><InspectorRow label="Task ID" value={taskId || (resolved.artifact ? "尚未提交（不会在此创建）" : "请求未保存")} /><Button block loading={loadingGenerationTaskDraft} disabled={!resolved.artifact} onClick={onInspectGenerationTaskDraft}>查看只读任务合同</Button><Button block loading={loadingProviderRoutes} disabled={!resolved.artifact} onClick={onInspectGenerationProviderRoutes}>查看可用后端渠道</Button>{generationTaskDraft ? <><InspectorRow label="请求版本" value={`v${generationTaskDraft.generationRequestArtifactVersion} · ${generationTaskDraft.generationRequestStatus}`} /><InspectorRow label="未来 Task 类型" value={generationTaskDraft.taskType} /><InspectorRow label="请求已就绪" value={generationTaskDraft.requestReady ? "是" : "否"} /><InspectorRow label="Provider 路由" value={generationTaskDraft.providerRouteResolved ? "已解析" : "尚未解析"} /><InspectorRow label="提交状态" value={generationTaskDraft.submissionState} /><InspectorRow label="来源版本" value={`${generationTaskDraft.gatewayInput.sourceRefs.length} 个`} /><InspectorRow label="请求指纹" value={generationTaskDraft.requestFingerprint.slice(0, 16)} />{generationTaskDraft.blockers.map((blocker) => <EmptyHint key={blocker} text={blocker} />)}</> : <EmptyHint text="这里只读取未来 Provider Gateway 所需的任务合同；不会创建 Task、扣费或调用模型。" />}{providerRoutes ? <><InspectorRow label="路由状态" value={providerRoutes.state} /><InspectorRow label="候选渠道" value={`${providerRoutes.routes.length} 个`} />{providerRoutes.routes.map((route) => <div key={`${route.channelId}:${route.model}`} className="rounded-lg border border-border/70 bg-foreground/[.025] p-2"><InspectorRow label={route.channelName || "系统渠道"} value={`${route.modelDisplayName} · ${route.routeReady ? "可提交前置就绪" : "待配置"}`} />{route.blockers.map((blocker) => <div key={blocker} className="mt-1 text-[var(--fs-micro)] text-amber-500/80">{blocker}</div>)}</div>)}{providerRoutes.blockers.map((blocker) => <EmptyHint key={blocker} text={blocker} />)}<EmptyHint text="候选路由只读取后端系统渠道状态，不会提交任务、扣费或调用模型。" /></> : null}</InspectorSection>;
    if (kind === "shot" && tab === "References") { const references = project?.shotReferences || []; return <InspectorSection title="Active References">{references.filter((item) => item.shotId === resolved.shot?.id).map((item) => <InspectorRow key={item.id} label={item.role} value={item.assetVersionId} />)}{!references.some((item) => item.shotId === resolved.shot?.id) ? <EmptyHint text="当前镜头尚未绑定资产版本。" /> : null}</InspectorSection>; }
    if (kind === "scene" && tab === "Shots") return <InspectorSection title="场内镜头">{resolved.sceneShots.map((shot) => <InspectorRow key={shot.id} label={shot.title} value={`${Math.round(shot.durationMs / 100) / 10} 秒 · ${shot.status}`} />)}{!resolved.sceneShots.length ? <EmptyHint text="当前场次还没有镜头。" /> : null}</InspectorSection>;
    if (tab === "Usage") return <InspectorSection title="资产使用"><InspectorRow label="使用镜头" value={`${resolved.usedShotCount} 个`} /></InspectorSection>;
    if (tab === "Versions") { const artifactVersions = resolved.artifact ? project?.filmArtifacts?.filter((item) => item.shotId === resolved.shot?.id && item.artifactType === resolved.artifact?.artifactType).length || 1 : resolved.asset?.versionCount || resolved.shot?.contractVersion || 1; return <InspectorSection title="版本记录"><InspectorRow label="当前版本" value={resolved.artifact ? `v${resolved.artifact.objectVersion}` : `v${resolved.asset?.currentVersion?.version || resolved.asset?.character?.version || 1}`} /><InspectorRow label="版本总数" value={String(artifactVersions)} /></InspectorSection>; }
    return <InspectorSection title={tab}><EmptyHint text="本标签已绑定统一对象头；详细生产对象将在对应阶段接入。" /></InspectorSection>;
}

function buildDraft(node: CanvasNodeData, project?: ProjectDetail) {
    const { scene, shot, asset, contract, definition } = resolveFilmNode(node, project);
    const promptPack = node.filmKind === "generation" ? resolveFilmNode({ ...node, filmKind: "prompt_pack" }, project) : undefined;
    const hasGenerationSnapshot = node.filmKind === "generation" && Boolean(filmText(contract.promptArtifactId));
    return {
        title: shot && node.filmKind === "shot" ? filmText(contract.shotCode) || shot.title : scene?.title || asset?.title || node.title,
        code: scene?.code || "", description: scene?.description || shot?.description || "", interiorExterior: scene?.interiorExterior || "", timeOfDay: scene?.timeOfDay || "", locationAssetId: scene?.locationAssetId || "",
        duration: shot ? String(shot.durationMs / 1000) : "5",
        ...stringFields(contract, ["shotSize", "script", "narrativeIntent", "blocking", "acting", "camera", "lens", "movement", "continuity"]),
        ...stringFields(definition, ["appearance", "physique", "clothing", "descriptor", "canonicalGeometry", "doors", "windows", "anchors", "fixedProps", "movableProps", "lightingLogic", "masterView", "reverseViews", "function", "states", "continuityLocks"]),
        ...stringFields(contract, ["objective", "obstacle", "tactic", "beat", "performance", "continuityLocks", "sceneContext", "activeReferences", "locationMap", "firstFrame", "bodyOrientation", "gaze", "formatMode", "optics", "timedActionBeats", "physics", "lighting", "audio", "style", "positiveLocks", "localFailureLocks", "emittedNegativeConstraints", "compiledPrompt", "mediaType", "aspectRatio", "outputIntent", "promptArtifactId", "promptArtifactVersion"]),
        generationStatus: node.filmKind === "generation" ? resolved.artifact?.status || "draft" : "draft",
        mediaType: filmText(contract.mediaType) || "video",
        aspectRatio: filmText(contract.aspectRatio) || project?.project.aspectRatio || "9:16",
        outputIntent: filmText(contract.outputIntent) || "首轮镜头生成",
        generationDuration: durationSeconds(contract.durationMs, shot?.durationMs || 5000),
        promptArtifactId: hasGenerationSnapshot ? filmText(contract.promptArtifactId) : promptPack?.artifact?.id || "",
        promptArtifactVersion: hasGenerationSnapshot ? filmText(contract.promptArtifactVersion) : promptPack?.artifact?.objectVersion?.toString() || "",
        compiledPrompt: hasGenerationSnapshot ? filmText(contract.compiledPrompt) : filmText(promptPack?.contract.compiledPrompt),
    };
}

function stringFields(source: Record<string, unknown>, fields: string[]) { return Object.fromEntries(fields.map((field) => [field, filmText(source[field])])); }
function contractFromDraft(draft: Record<string, string>) { return { shotCode: draft.title, durationMs: secondsToMs(draft.duration), shotSize: draft.shotSize, script: draft.script, narrativeIntent: draft.narrativeIntent, blocking: draft.blocking, acting: draft.acting, camera: draft.camera, lens: draft.lens, movement: draft.movement, continuity: draft.continuity }; }
function secondsToMs(value: string) { const seconds = Number(value); return Number.isFinite(seconds) && seconds >= 0 ? Math.round(seconds * 1000) : 0; }
function durationSeconds(value: unknown, fallbackDurationMS: number) { const durationMS = Number(value); return String((Number.isFinite(durationMS) && durationMS > 0 ? durationMS : fallbackDurationMS) / 1000); }
function definitionFromDraft(kind: FilmNodeKind, draft: Record<string, string>, current: Record<string, unknown>) { const keys = kind === "location" ? ["canonicalGeometry", "doors", "windows", "anchors", "fixedProps", "movableProps", "lightingLogic", "masterView", "reverseViews"] : kind === "prop" ? ["appearance", "function", "states", "continuityLocks"] : ["appearance", "physique", "clothing", "descriptor"]; return { ...current, ...Object.fromEntries(keys.map((key) => [key, draft[key] || ""])) }; }
function artifactFromDraft(kind: FilmNodeKind, draft: Record<string, string>, current: Record<string, unknown>) { const keys = kind === "acting" ? ["objective", "obstacle", "tactic", "beat", "performance", "continuityLocks"] : ["sceneContext", "activeReferences", "locationMap", "firstFrame", "blocking", "bodyOrientation", "gaze", "anchors", "formatMode", "optics", "camera", "timedActionBeats", "physics", "lighting", "audio", "acting", "style", "compiledPrompt"]; const result = { ...current, ...Object.fromEntries(keys.map((key) => [key, draft[key] || ""])) }; if (kind === "prompt_pack") { for (const key of ["positiveLocks", "localFailureLocks", "emittedNegativeConstraints"]) result[key] = splitList(draft[key]); } return result; }
function generationRequestFromDraft(draft: Record<string, string>) { return { promptArtifactId: draft.promptArtifactId, mediaType: draft.mediaType || "video", aspectRatio: draft.aspectRatio, durationMs: secondsToMs(draft.generationDuration), outputIntent: draft.outputIntent || "首轮镜头生成" }; }
function splitList(value: string | undefined) { return (value || "").split(/\n|、/).map((item) => item.trim()).filter(Boolean); }

function TabBar({ tabs, activeTab, onTabChange }: { tabs: readonly string[]; activeTab: string; onTabChange: (tab: string) => void }) { return <div className="border-b border-border/70 px-3 py-2"><div className="thin-scrollbar flex gap-1 overflow-x-auto" role="tablist" aria-label="检查器标签">{tabs.map((tab) => <button key={tab} type="button" role="tab" aria-selected={activeTab === tab} onClick={() => onTabChange(tab)} className={`h-7 shrink-0 rounded px-2 text-[var(--fs-micro)] font-medium ${activeTab === tab ? "bg-[var(--workspace-accent-soft)] text-[var(--workspace-accent)]" : "text-foreground/42 hover:bg-foreground/[.05] hover:text-foreground"}`}>{tab}</button>)}</div></div>; }
function InspectorHeader({ eyebrow, title, subtitle, attention }: { eyebrow: string; title: string; subtitle: string; attention?: FilmNodeState["attention"] }) { const AttentionIcon = attention === "error" ? AlertTriangle : attention === "warning" || attention === "human_required" ? CircleDot : CheckCircle2; const tone = attention === "error" ? "text-red-400" : attention === "warning" || attention === "human_required" ? "text-amber-400" : "text-emerald-400"; return <header className="shrink-0 border-b border-border/70 px-4 py-3"><div className="flex items-start justify-between gap-2"><div className="min-w-0"><div className="text-[var(--fs-tiny)] font-semibold uppercase tracking-widest text-foreground/35">{eyebrow}</div><h2 className="mt-1 truncate text-sm font-semibold text-foreground/82" title={title}>{title}</h2><div className="mt-1 truncate font-mono text-[var(--fs-micro)] text-foreground/32" title={subtitle}>{subtitle}</div></div>{attention ? <AttentionIcon className={`mt-1 size-4 shrink-0 ${tone}`} /> : null}</div></header>; }
function InspectorSection({ title, children }: { title: string; children: ReactNode }) { return <section className="mb-5"><h3 className="mb-2 flex items-center gap-1.5 text-[var(--fs-tiny)] font-semibold uppercase tracking-widest text-foreground/38"><ShieldCheck className="size-3.5" />{title}</h3><div className="space-y-2">{children}</div></section>; }
function InspectorRow({ label, value }: { label: string; value: string }) { return <div className="grid grid-cols-2 gap-2 text-[var(--fs-caption)]"><span className="text-foreground/38">{label}</span><span className="break-all text-foreground/68">{value}</span></div>; }
function ProjectMetric({ label, value }: { label: string; value: number }) { return <div className="rounded-lg border border-border/70 bg-foreground/[.025] p-2 text-center"><strong className="block text-sm tabular-nums text-foreground/75">{value}</strong><span className="mt-0.5 block text-[var(--fs-micro)] text-foreground/36">{label}</span></div>; }
function EmptyHint({ text }: { text: string }) { return <div className="my-3 rounded-lg border border-dashed border-border/80 p-3 text-center"><FileQuestion className="mx-auto size-5 text-foreground/22" /><p className="mt-1 text-[var(--fs-caption)] leading-5 text-foreground/40">{text}</p></div>; }
function FormField({ label, value, onChange, multiline, disabled, placeholder, type = "text" }: { label: string; value: string; onChange: (value: string) => void; multiline?: boolean; disabled?: boolean; placeholder?: string; type?: string }) { const classes = "w-full rounded-md border border-border/80 bg-foreground/[.025] px-2.5 py-2 text-xs text-foreground/78 outline-none focus:border-[var(--workspace-accent)] disabled:opacity-45"; const normalizedValue = value ?? ""; return <label className="block"><span className="mb-1 block text-[var(--fs-micro)] text-foreground/42">{label}</span>{multiline ? <textarea rows={3} className={classes} value={normalizedValue} placeholder={placeholder} disabled={disabled} onChange={(event) => onChange(event.target.value)} /> : <input type={type} className={classes} value={normalizedValue} placeholder={placeholder} disabled={disabled} onChange={(event) => onChange(event.target.value)} />}</label>; }
function FormSelect({ label, value, onChange, options }: { label: string; value: string; onChange: (value: string) => void; options: Array<{ value: string; label: string }> }) { return <label className="block"><span className="mb-1 block text-[var(--fs-micro)] text-foreground/42">{label}</span><select className="w-full rounded-md border border-border/80 bg-background px-2.5 py-2 text-xs text-foreground/78 outline-none focus:border-[var(--workspace-accent)]" value={value ?? ""} onChange={(event) => onChange(event.target.value)}>{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>; }
function navigationLabel(section: ProductionNavigationKey) { return ({ overview: "总览", story: "故事", script: "剧本", scenes: "场景", characters: "角色", locations: "场地", props: "道具", storyboard: "分镜", shots: "镜头", assets: "资产", audio: "声音", qc: "QC", agents: "Agent", needs_you: "待处理", delivery: "交付" } satisfies Record<ProductionNavigationKey, string>)[section]; }
