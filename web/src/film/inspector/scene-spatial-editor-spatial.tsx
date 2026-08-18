import { Button } from "antd";

import type { CameraAnchorDraft, RequiredPathDraft, SceneAnchorDraft, SceneAssetPackDraft, TopologyEdgeDraft, TopologyNodeDraft, ViewpointDraft } from "./scene-spatial-editor-model";
import { nextSpatialId } from "./scene-spatial-editor-model";
import { CheckboxField, EditorGroup, EditorItem, NumberField, NumberListField, PointFields, SelectField, TextField, TextListField, removeAt, replaceAt } from "./scene-spatial-editor-controls";

type EditorProps = { draft: SceneAssetPackDraft; sceneId: string; update: (updater: (current: SceneAssetPackDraft) => SceneAssetPackDraft) => void };

export function SceneSpatialStructurePanel({ draft, sceneId, update }: EditorProps) {
    const nodes = draft.sceneTopologyGraph.nodes;
    const nodeOptions = [{ value: "", label: "未绑定" }, ...nodes.map((node) => ({ value: node.id, label: node.label ? `${node.id} · ${node.label}` : node.id }))];
    const setGraph = (patch: Partial<SceneAssetPackDraft["sceneTopologyGraph"]>) => update((current) => ({ ...current, sceneTopologyGraph: { ...current.sceneTopologyGraph, ...patch } }));
    const setPlan = (patch: Partial<SceneAssetPackDraft["spatialFloorPlan"]>) => update((current) => ({ ...current, spatialFloorPlan: { ...current.spatialFloorPlan, ...patch } }));

    return (
        <>
            <EditorGroup
                title="SceneTopologyGraph"
                detail="先确认房间、门口、街角等空间如何相连，再制作视角资产。"
                onAdd={() => {
                    const id = nextSpatialId(
                        "space",
                        nodes.map((node) => node.id),
                    );
                    setGraph({ nodes: [...nodes, { id, kind: "room", label: "", position: { x: 0, y: 0, z: 0 } }] });
                }}
                addLabel="添加空间"
            >
                {nodes.map((node, index) => (
                    <TopologyNodeEditor key={`${node.id}-${index}`} node={node} index={index} onChange={(next) => setGraph({ nodes: replaceAt(nodes, index, next) })} onRemove={() => setGraph({ nodes: removeAt(nodes, index) })} />
                ))}
                {!nodes.length ? <EmptyState text="至少需要一个可辨识的空间节点。" /> : null}
                <div className="mt-3 border-t border-border/60 pt-3">
                    <div className="mb-2 flex items-center justify-between">
                        <span className="text-xs font-semibold text-foreground/80">空间关系</span>
                        <Button
                            size="small"
                            onClick={() =>
                                setGraph({
                                    edges: [...draft.sceneTopologyGraph.edges, { from: nodes[0]?.id || "", to: "", direction: "", distance: 0, visibility: "", elevation: "", transition: "", connectionType: "", doorRelation: "", windowRelation: "" }],
                                })
                            }
                        >
                            添加关系
                        </Button>
                    </div>
                    {draft.sceneTopologyGraph.edges.map((edge, index) => (
                        <TopologyEdgeEditor
                            key={`${edge.from}-${edge.to}-${index}`}
                            edge={edge}
                            nodeOptions={nodeOptions}
                            onChange={(next) => setGraph({ edges: replaceAt(draft.sceneTopologyGraph.edges, index, next) })}
                            onRemove={() => setGraph({ edges: removeAt(draft.sceneTopologyGraph.edges, index) })}
                        />
                    ))}
                </div>
            </EditorGroup>

            <EditorGroup title="SpatialFloorPlan" detail="坐标使用本场景的逻辑单位。门窗、固定物、人物和机位都不能超出此范围。">
                <div className="grid grid-cols-2 gap-2">
                    <TextField label="坐标系" value={draft.spatialFloorPlan.coordinateSystem} onChange={(coordinateSystem) => setPlan({ coordinateSystem })} />
                    <NumberField label="宽度" value={draft.spatialFloorPlan.width} min={0} onChange={(width) => setPlan({ width })} />
                    <NumberField label="深度" value={draft.spatialFloorPlan.depth} min={0} onChange={(depth) => setPlan({ depth })} />
                </div>
                <div className="mt-2">
                    <PointFields label="平面原点" value={draft.spatialFloorPlan.origin} onChange={(origin) => setPlan({ origin })} />
                </div>
                <div className="mt-3 border-t border-border/60 pt-3">
                    <div className="mb-2 flex items-center justify-between">
                        <span className="text-xs font-semibold text-foreground/80">功能区</span>
                        <Button
                            size="small"
                            onClick={() => {
                                const id = nextSpatialId(
                                    "zone",
                                    draft.spatialFloorPlan.zones.map((zone) => zone.id),
                                );
                                setPlan({ zones: [...draft.spatialFloorPlan.zones, { id, topologyNodeId: "", label: "", minX: 0, minY: 0, maxX: draft.spatialFloorPlan.width, maxY: draft.spatialFloorPlan.depth, purpose: "" }] });
                            }}
                        >
                            添加功能区
                        </Button>
                    </div>
                    {draft.spatialFloorPlan.zones.map((zone, index) => (
                        <EditorItem key={`${zone.id}-${index}`} title={zone.id || `功能区 ${index + 1}`} onRemove={() => setPlan({ zones: removeAt(draft.spatialFloorPlan.zones, index) })}>
                            <div className="grid grid-cols-2 gap-2">
                                <TextField label="Zone ID" value={zone.id} onChange={(id) => setPlan({ zones: replaceAt(draft.spatialFloorPlan.zones, index, { ...zone, id }) })} />
                                <SelectField label="拓扑节点" value={zone.topologyNodeId} options={nodeOptions} onChange={(topologyNodeId) => setPlan({ zones: replaceAt(draft.spatialFloorPlan.zones, index, { ...zone, topologyNodeId }) })} />
                                <TextField label="标签" value={zone.label} onChange={(label) => setPlan({ zones: replaceAt(draft.spatialFloorPlan.zones, index, { ...zone, label }) })} />
                                <TextField label="用途" value={zone.purpose} onChange={(purpose) => setPlan({ zones: replaceAt(draft.spatialFloorPlan.zones, index, { ...zone, purpose }) })} />
                                <NumberField label="Min X" value={zone.minX} onChange={(minX) => setPlan({ zones: replaceAt(draft.spatialFloorPlan.zones, index, { ...zone, minX }) })} />
                                <NumberField label="Min Y" value={zone.minY} onChange={(minY) => setPlan({ zones: replaceAt(draft.spatialFloorPlan.zones, index, { ...zone, minY }) })} />
                                <NumberField label="Max X" value={zone.maxX} onChange={(maxX) => setPlan({ zones: replaceAt(draft.spatialFloorPlan.zones, index, { ...zone, maxX }) })} />
                                <NumberField label="Max Y" value={zone.maxY} onChange={(maxY) => setPlan({ zones: replaceAt(draft.spatialFloorPlan.zones, index, { ...zone, maxY }) })} />
                            </div>
                        </EditorItem>
                    ))}
                </div>
                <div className="mt-3 border-t border-border/60 pt-3">
                    <div className="mb-2 flex items-center justify-between">
                        <span className="text-xs font-semibold text-foreground/80">门窗开口</span>
                        <Button
                            size="small"
                            onClick={() => {
                                const id = nextSpatialId(
                                    "opening",
                                    draft.spatialFloorPlan.openings.map((opening) => opening.id),
                                );
                                setPlan({ openings: [...draft.spatialFloorPlan.openings, { id, kind: "door", position: { x: 0, y: 0, z: 0 }, orientation: "", state: "", connectsTo: "", continuityLock: true }] });
                            }}
                        >
                            添加开口
                        </Button>
                    </div>
                    {draft.spatialFloorPlan.openings.map((opening, index) => (
                        <EditorItem key={`${opening.id}-${index}`} title={opening.id || `开口 ${index + 1}`} onRemove={() => setPlan({ openings: removeAt(draft.spatialFloorPlan.openings, index) })}>
                            <div className="grid grid-cols-2 gap-2">
                                <TextField label="Opening ID" value={opening.id} onChange={(id) => setPlan({ openings: replaceAt(draft.spatialFloorPlan.openings, index, { ...opening, id }) })} />
                                <SelectField
                                    label="类型"
                                    value={opening.kind}
                                    options={[
                                        { value: "door", label: "门" },
                                        { value: "window", label: "窗" },
                                    ]}
                                    onChange={(kind) => setPlan({ openings: replaceAt(draft.spatialFloorPlan.openings, index, { ...opening, kind }) })}
                                />
                                <TextField label="朝向" value={opening.orientation} onChange={(orientation) => setPlan({ openings: replaceAt(draft.spatialFloorPlan.openings, index, { ...opening, orientation }) })} />
                                <TextField label="状态" value={opening.state} onChange={(state) => setPlan({ openings: replaceAt(draft.spatialFloorPlan.openings, index, { ...opening, state }) })} />
                                <SelectField label="连接空间" value={opening.connectsTo} options={nodeOptions} onChange={(connectsTo) => setPlan({ openings: replaceAt(draft.spatialFloorPlan.openings, index, { ...opening, connectsTo }) })} />
                            </div>
                            <PointFields value={opening.position} onChange={(position) => setPlan({ openings: replaceAt(draft.spatialFloorPlan.openings, index, { ...opening, position }) })} />
                            <CheckboxField label="锁定门窗连续性" checked={opening.continuityLock} onChange={(continuityLock) => setPlan({ openings: replaceAt(draft.spatialFloorPlan.openings, index, { ...opening, continuityLock }) })} />
                        </EditorItem>
                    ))}
                </div>
            </EditorGroup>

            <AnchorList title="固定空间锚点" detail="墙、门、窗、固定家具等必须开启连续性锁。" anchors={draft.fixedAnchors} sceneId={sceneId} fixed onChange={(fixedAnchors) => update((current) => ({ ...current, fixedAnchors }))} />
            <AnchorList title="可移动锚点" detail="人物和可移动道具的起始位置，也会供视角可见性和路径校验使用。" anchors={draft.movableAnchors} sceneId={sceneId} onChange={(movableAnchors) => update((current) => ({ ...current, movableAnchors }))} />
        </>
    );
}

export function SceneSpatialCoveragePanel({ draft, sceneId, update }: EditorProps) {
    const nodes = draft.sceneTopologyGraph.nodes;
    const anchors = [...draft.fixedAnchors, ...draft.movableAnchors];
    const cameraOptions = [{ value: "", label: "未绑定" }, ...draft.cameraAnchorPlan.anchors.map((camera) => ({ value: camera.cameraId, label: camera.cameraId }))];
    const nodeOptions = [{ value: "", label: "未绑定" }, ...nodes.map((node) => ({ value: node.id, label: node.label ? `${node.id} · ${node.label}` : node.id }))];
    const anchorOptions = [{ value: "", label: "未绑定" }, ...anchors.map((anchor) => ({ value: anchor.anchorId, label: anchor.objectId ? `${anchor.anchorId} · ${anchor.objectId}` : anchor.anchorId }))];
    const sceneAssetOptions = [draft.masterSceneAsset, ...draft.viewAssets].flatMap((asset, index) =>
        [asset.assetId, asset.uri].filter(Boolean).map((value) => ({
            value,
            label: index === 0 ? `主场景 · ${value}` : asset.viewId ? `${asset.viewId} · ${value}` : value,
        })),
    );
    const viewAssetOptions = [{ value: "", label: "未绑定" }, ...Array.from(new Map(sceneAssetOptions.map((option) => [option.value, option])).values())];
    const setCameraPlan = (patch: Partial<SceneAssetPackDraft["cameraAnchorPlan"]>) => update((current) => ({ ...current, cameraAnchorPlan: { ...current.cameraAnchorPlan, ...patch } }));
    const setCoverage = (patch: Partial<SceneAssetPackDraft["viewpointCoverageMatrix"]>) => update((current) => ({ ...current, viewpointCoverageMatrix: { ...current.viewpointCoverageMatrix, ...patch } }));

    return (
        <>
            <EditorGroup
                title="CameraAnchorPlan"
                detail="机位必须落在平面图范围内；正反打需互相声明并位于同一轴线侧。"
                onAdd={() => {
                    const id = nextSpatialId(
                        "camera",
                        draft.cameraAnchorPlan.anchors.map((camera) => camera.cameraId),
                    );
                    setCameraPlan({
                        anchors: [
                            ...draft.cameraAnchorPlan.anchors,
                            {
                                cameraId: id,
                                sceneId,
                                topologyNodeId: "",
                                anchorId: "",
                                position: { x: 0, y: 0, z: 0 },
                                height: 0,
                                direction: "",
                                facing: "",
                                target: "",
                                lensClass: "",
                                shotSizeRange: [],
                                allowedFovRange: [],
                                movementConstraints: [],
                                axisSide: "",
                                reverseOf: "",
                            },
                        ],
                    });
                }}
                addLabel="添加机位"
            >
                <TextField label="动作轴" value={draft.cameraAnchorPlan.actionAxis} onChange={(actionAxis) => setCameraPlan({ actionAxis })} placeholder="例如：操作台 → 桥接椅" />
                <div className="mt-3">
                    {draft.cameraAnchorPlan.anchors.map((camera, index) => (
                        <CameraEditor
                            key={`${camera.cameraId}-${index}`}
                            camera={camera}
                            index={index}
                            nodeOptions={nodeOptions}
                            anchorOptions={anchorOptions}
                            cameraOptions={cameraOptions}
                            onChange={(next) => setCameraPlan({ anchors: replaceAt(draft.cameraAnchorPlan.anchors, index, next) })}
                            onRemove={() => setCameraPlan({ anchors: removeAt(draft.cameraAnchorPlan.anchors, index) })}
                        />
                    ))}
                    {!draft.cameraAnchorPlan.anchors.length ? <EmptyState text="至少需要一个可拍机位。" /> : null}
                </div>
            </EditorGroup>

            <EditorGroup
                title="ViewpointCoverageMatrix"
                detail="按剧情镜头需求覆盖视角。必需视角要绑定机位、用途和对应视角资产。"
                onAdd={() => {
                    const id = nextSpatialId(
                        "view",
                        draft.viewpointCoverageMatrix.views.map((view) => view.viewId),
                    );
                    setCoverage({ views: [...draft.viewpointCoverageMatrix.views, { viewId: id, cameraAnchorId: "", facing: "", purpose: "", required: true, assetReference: "", requiredVisibleObjectIds: [], forbiddenObjectIds: [] }] });
                }}
                addLabel="添加视角"
            >
                {draft.viewpointCoverageMatrix.views.map((view, index) => (
                    <ViewpointEditor
                        key={`${view.viewId}-${index}`}
                        view={view}
                        index={index}
                        cameraOptions={cameraOptions}
                        viewAssetOptions={viewAssetOptions}
                        onChange={(next) => setCoverage({ views: replaceAt(draft.viewpointCoverageMatrix.views, index, next) })}
                        onRemove={() => setCoverage({ views: removeAt(draft.viewpointCoverageMatrix.views, index) })}
                    />
                ))}
                {!draft.viewpointCoverageMatrix.views.length ? <EmptyState text="不要按正面、侧面机械补图；只添加剧情实际需要的视角。" /> : null}
            </EditorGroup>

            <EditorGroup
                title="必经路径"
                detail="用于校验人物或道具是否能沿拓扑图到达下一个动作位置。"
                onAdd={() => {
                    const id = nextSpatialId(
                        "path",
                        draft.requiredPaths.map((path) => path.id),
                    );
                    update((current) => ({ ...current, requiredPaths: [...current.requiredPaths, { id, from: "", to: "", via: [], purpose: "" }] }));
                }}
                addLabel="添加路径"
            >
                {draft.requiredPaths.map((path, index) => (
                    <PathEditor
                        key={`${path.id}-${index}`}
                        path={path}
                        index={index}
                        nodeOptions={nodeOptions}
                        onChange={(next) => update((current) => ({ ...current, requiredPaths: replaceAt(current.requiredPaths, index, next) }))}
                        onRemove={() => update((current) => ({ ...current, requiredPaths: removeAt(current.requiredPaths, index) }))}
                    />
                ))}
            </EditorGroup>
        </>
    );
}

function TopologyNodeEditor({ node, index, onChange, onRemove }: { node: TopologyNodeDraft; index: number; onChange: (node: TopologyNodeDraft) => void; onRemove: () => void }) {
    return (
        <EditorItem title={node.id || `空间 ${index + 1}`} onRemove={onRemove}>
            <div className="grid grid-cols-2 gap-2">
                <TextField label="Node ID" value={node.id} onChange={(id) => onChange({ ...node, id })} />
                <TextField label="类型" value={node.kind} onChange={(kind) => onChange({ ...node, kind })} />
                <TextField label="标签" value={node.label} onChange={(label) => onChange({ ...node, label })} />
            </div>
            <PointFields value={node.position} onChange={(position) => onChange({ ...node, position })} />
        </EditorItem>
    );
}

function TopologyEdgeEditor({ edge, nodeOptions, onChange, onRemove }: { edge: TopologyEdgeDraft; nodeOptions: Array<{ value: string; label: string }>; onChange: (edge: TopologyEdgeDraft) => void; onRemove: () => void }) {
    return (
        <EditorItem title={`${edge.from || "?"} → ${edge.to || "?"}`} onRemove={onRemove}>
            <div className="grid grid-cols-2 gap-2">
                <SelectField label="起点" value={edge.from} options={nodeOptions} onChange={(from) => onChange({ ...edge, from })} />
                <SelectField label="终点" value={edge.to} options={nodeOptions} onChange={(to) => onChange({ ...edge, to })} />
                <TextField label="方向" value={edge.direction} onChange={(direction) => onChange({ ...edge, direction })} />
                <NumberField label="距离" value={edge.distance} min={0} onChange={(distance) => onChange({ ...edge, distance })} />
                <TextField label="可见性" value={edge.visibility} onChange={(visibility) => onChange({ ...edge, visibility })} />
                <TextField label="高差" value={edge.elevation} onChange={(elevation) => onChange({ ...edge, elevation })} />
                <TextField label="转场" value={edge.transition} onChange={(transition) => onChange({ ...edge, transition })} />
                <TextField label="连接类型" value={edge.connectionType} onChange={(connectionType) => onChange({ ...edge, connectionType })} />
                <TextField label="门关系" value={edge.doorRelation} onChange={(doorRelation) => onChange({ ...edge, doorRelation })} />
                <TextField label="窗关系" value={edge.windowRelation} onChange={(windowRelation) => onChange({ ...edge, windowRelation })} />
            </div>
        </EditorItem>
    );
}

function AnchorList({ title, detail, anchors, sceneId, fixed = false, onChange }: { title: string; detail: string; anchors: SceneAnchorDraft[]; sceneId: string; fixed?: boolean; onChange: (anchors: SceneAnchorDraft[]) => void }) {
    return (
        <EditorGroup
            title={title}
            detail={detail}
            onAdd={() => {
                const id = nextSpatialId(
                    fixed ? "fixed" : "movable",
                    anchors.map((anchor) => anchor.anchorId),
                );
                onChange([...anchors, { anchorId: id, sceneId, objectId: "", anchorType: "", position: { x: 0, y: 0, z: 0 }, orientation: "", mobility: fixed ? "fixed" : "movable", continuityLock: fixed, evidence: [] }]);
            }}
            addLabel="添加锚点"
        >
            {anchors.map((anchor, index) => (
                <EditorItem key={`${anchor.anchorId}-${index}`} title={anchor.anchorId || `锚点 ${index + 1}`} onRemove={() => onChange(removeAt(anchors, index))}>
                    <div className="grid grid-cols-2 gap-2">
                        <TextField label="Anchor ID" value={anchor.anchorId} onChange={(anchorId) => onChange(replaceAt(anchors, index, { ...anchor, anchorId }))} />
                        <TextField label="对象 ID" value={anchor.objectId} onChange={(objectId) => onChange(replaceAt(anchors, index, { ...anchor, objectId }))} />
                        <TextField label="锚点类型" value={anchor.anchorType} onChange={(anchorType) => onChange(replaceAt(anchors, index, { ...anchor, anchorType }))} />
                        <TextField label="移动性" value={anchor.mobility} onChange={(mobility) => onChange(replaceAt(anchors, index, { ...anchor, mobility }))} />
                        <TextField label="朝向" value={anchor.orientation} onChange={(orientation) => onChange(replaceAt(anchors, index, { ...anchor, orientation }))} />
                    </div>
                    <PointFields value={anchor.position} onChange={(position) => onChange(replaceAt(anchors, index, { ...anchor, position }))} />
                    <TextListField label="证据 ID" values={anchor.evidence} onChange={(evidence) => onChange(replaceAt(anchors, index, { ...anchor, evidence }))} />
                    <CheckboxField label="锁定连续性" checked={anchor.continuityLock} onChange={(continuityLock) => onChange(replaceAt(anchors, index, { ...anchor, continuityLock }))} />
                </EditorItem>
            ))}
        </EditorGroup>
    );
}

function CameraEditor({
    camera,
    index,
    nodeOptions,
    anchorOptions,
    cameraOptions,
    onChange,
    onRemove,
}: {
    camera: CameraAnchorDraft;
    index: number;
    nodeOptions: Array<{ value: string; label: string }>;
    anchorOptions: Array<{ value: string; label: string }>;
    cameraOptions: Array<{ value: string; label: string }>;
    onChange: (camera: CameraAnchorDraft) => void;
    onRemove: () => void;
}) {
    return (
        <EditorItem title={camera.cameraId || `机位 ${index + 1}`} onRemove={onRemove}>
            <div className="grid grid-cols-2 gap-2">
                <TextField label="Camera ID" value={camera.cameraId} onChange={(cameraId) => onChange({ ...camera, cameraId })} />
                <SelectField label="所属空间" value={camera.topologyNodeId} options={nodeOptions} onChange={(topologyNodeId) => onChange({ ...camera, topologyNodeId })} />
                <SelectField label="空间锚点" value={camera.anchorId} options={anchorOptions} onChange={(anchorId) => onChange({ ...camera, anchorId })} />
                <TextField label="目标" value={camera.target} onChange={(target) => onChange({ ...camera, target })} />
                <NumberField label="高度" value={camera.height} onChange={(height) => onChange({ ...camera, height })} />
                <TextField label="朝向" value={camera.direction} onChange={(direction) => onChange({ ...camera, direction })} />
                <TextField label="面对" value={camera.facing} onChange={(facing) => onChange({ ...camera, facing })} />
                <TextField label="镜头类别" value={camera.lensClass} onChange={(lensClass) => onChange({ ...camera, lensClass })} />
                <TextField label="动作轴侧" value={camera.axisSide} onChange={(axisSide) => onChange({ ...camera, axisSide })} />
                <SelectField label="反打机位" value={camera.reverseOf} options={cameraOptions.filter((option) => option.value !== camera.cameraId)} onChange={(reverseOf) => onChange({ ...camera, reverseOf })} />
            </div>
            <PointFields value={camera.position} onChange={(position) => onChange({ ...camera, position })} />
            <TextListField label="允许景别" values={camera.shotSizeRange} onChange={(shotSizeRange) => onChange({ ...camera, shotSizeRange })} />
            <NumberListField label="允许 FOV 范围（两个数）" values={camera.allowedFovRange} onChange={(allowedFovRange) => onChange({ ...camera, allowedFovRange })} />
            <TextListField label="运镜约束" values={camera.movementConstraints} onChange={(movementConstraints) => onChange({ ...camera, movementConstraints })} />
        </EditorItem>
    );
}

function ViewpointEditor({
    view,
    index,
    cameraOptions,
    viewAssetOptions,
    onChange,
    onRemove,
}: {
    view: ViewpointDraft;
    index: number;
    cameraOptions: Array<{ value: string; label: string }>;
    viewAssetOptions: Array<{ value: string; label: string }>;
    onChange: (view: ViewpointDraft) => void;
    onRemove: () => void;
}) {
    return (
        <EditorItem title={view.viewId || `视角 ${index + 1}`} onRemove={onRemove}>
            <div className="grid grid-cols-2 gap-2">
                <TextField label="View ID" value={view.viewId} onChange={(viewId) => onChange({ ...view, viewId })} />
                <SelectField label="机位" value={view.cameraAnchorId} options={cameraOptions} onChange={(cameraAnchorId) => onChange({ ...view, cameraAnchorId })} />
                <TextField label="面对" value={view.facing} onChange={(facing) => onChange({ ...view, facing })} />
                <TextField label="剧情用途" value={view.purpose} onChange={(purpose) => onChange({ ...view, purpose })} />
                <SelectField label="资产引用" value={view.assetReference} options={viewAssetOptions} onChange={(assetReference) => onChange({ ...view, assetReference })} />
            </div>
            <CheckboxField label="这是必需视角" checked={view.required} onChange={(required) => onChange({ ...view, required })} />
            <TextListField label="必须可见对象 ID" values={view.requiredVisibleObjectIds} onChange={(requiredVisibleObjectIds) => onChange({ ...view, requiredVisibleObjectIds })} placeholder="每行一个对象 ID" />
            <TextListField label="禁止出现对象 ID" values={view.forbiddenObjectIds} onChange={(forbiddenObjectIds) => onChange({ ...view, forbiddenObjectIds })} placeholder="每行一个对象 ID" />
        </EditorItem>
    );
}

function PathEditor({ path, index, nodeOptions, onChange, onRemove }: { path: RequiredPathDraft; index: number; nodeOptions: Array<{ value: string; label: string }>; onChange: (path: RequiredPathDraft) => void; onRemove: () => void }) {
    return (
        <EditorItem title={path.id || `路径 ${index + 1}`} onRemove={onRemove}>
            <div className="grid grid-cols-2 gap-2">
                <TextField label="路径 ID" value={path.id} onChange={(id) => onChange({ ...path, id })} />
                <SelectField label="起点" value={path.from} options={nodeOptions} onChange={(from) => onChange({ ...path, from })} />
                <SelectField label="终点" value={path.to} options={nodeOptions} onChange={(to) => onChange({ ...path, to })} />
                <TextField label="用途" value={path.purpose} onChange={(purpose) => onChange({ ...path, purpose })} />
            </div>
            <TextListField label="途经空间" values={path.via} onChange={(via) => onChange({ ...path, via })} placeholder="每行一个拓扑节点 ID" />
        </EditorItem>
    );
}

function EmptyState({ text }: { text: string }) {
    return <p className="py-2 text-[var(--fs-caption)] leading-5 text-foreground/42">{text}</p>;
}
