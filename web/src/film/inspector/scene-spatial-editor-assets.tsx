import { Button } from "antd";

import type { ProjectAsset } from "@/services/api/projects";
import type { EvidenceDraft, SceneAssetPackDraft, SceneAssetReferenceDraft, SceneLookCardDraft } from "./scene-spatial-editor-model";
import { createSceneLookCardDraft, nextSpatialId } from "./scene-spatial-editor-model";
import { CheckboxField, EditorGroup, EditorItem, SelectField, TextField, TextListField, removeAt, replaceAt } from "./scene-spatial-editor-controls";

type Props = { draft: SceneAssetPackDraft; assets: ProjectAsset[]; update: (updater: (current: SceneAssetPackDraft) => SceneAssetPackDraft) => void };

export function SceneSpatialAssetsPanel({ draft, assets, update }: Props) {
    const assetOptions = [{ value: "", label: "从项目资产选择" }, ...assets.map((asset) => ({ value: asset.id, label: `${asset.title} · ${asset.category}` }))];
    const viewOptions = [{ value: "", label: "主场景资产" }, ...draft.viewpointCoverageMatrix.views.map((view) => ({ value: view.viewId, label: view.viewId }))];
    const evidenceOptions = [{ value: "", label: "未绑定证据" }, ...draft.evidence.map((item) => ({ value: item.id, label: item.id }))];
    const mode = draft.sceneManifest.interiorExterior;
    const showInterior = Boolean(draft.interiorLookCard) || mode === "interior" || mode === "mixed";
    const showExterior = Boolean(draft.exteriorLookCard) || mode === "exterior" || mode === "mixed";

    return (
        <>
            <EditorGroup title="场景资产与证据" detail="主场景和每个必需视角都要有可追溯资产，并绑定已确认的证据。">
                <SceneAssetReferenceEditor
                    title="主场景资产"
                    value={draft.masterSceneAsset}
                    assetOptions={assetOptions}
                    viewOptions={viewOptions}
                    evidenceOptions={evidenceOptions}
                    onChange={(masterSceneAsset) => update((current) => ({ ...current, masterSceneAsset }))}
                />
                <div className="mt-3 border-t border-border/60 pt-3">
                    <div className="mb-2 flex items-center justify-between">
                        <span className="text-xs font-semibold text-foreground/80">视角资产</span>
                        <Button size="small" onClick={() => update((current) => ({ ...current, viewAssets: [...current.viewAssets, { assetId: "", uri: "", role: "view", viewId: "", evidenceRef: "" }] }))}>
                            添加资产
                        </Button>
                    </div>
                    {draft.viewAssets.map((asset, index) => (
                        <EditorItem key={`${asset.assetId}-${asset.uri}-${index}`} title={asset.viewId || `视角资产 ${index + 1}`} onRemove={() => update((current) => ({ ...current, viewAssets: removeAt(current.viewAssets, index) }))}>
                            <SceneAssetReferenceEditor
                                title=""
                                value={asset}
                                assetOptions={assetOptions}
                                viewOptions={viewOptions}
                                evidenceOptions={evidenceOptions}
                                onChange={(next) => update((current) => ({ ...current, viewAssets: replaceAt(current.viewAssets, index, next) }))}
                            />
                        </EditorItem>
                    ))}
                </div>
                <div className="mt-3 border-t border-border/60 pt-3">
                    <div className="mb-2 flex items-center justify-between">
                        <span className="text-xs font-semibold text-foreground/80">证据</span>
                        <Button
                            size="small"
                            onClick={() => {
                                const id = nextSpatialId(
                                    "evidence",
                                    draft.evidence.map((item) => item.id),
                                );
                                update((current) => ({ ...current, evidence: [...current.evidence, { id, kind: "scene_asset", sourceRef: "", verified: false }] }));
                            }}
                        >
                            添加证据
                        </Button>
                    </div>
                    {draft.evidence.map((item, index) => (
                        <EvidenceEditor
                            key={`${item.id}-${index}`}
                            item={item}
                            index={index}
                            assetOptions={assetOptions}
                            onChange={(next) => update((current) => ({ ...current, evidence: replaceAt(current.evidence, index, next) }))}
                            onRemove={() => update((current) => ({ ...current, evidence: removeAt(current.evidence, index) }))}
                        />
                    ))}
                </div>
            </EditorGroup>

            <EditorGroup title="SceneLookCard" detail="色板和光源方向属于场景连续性事实。室内和室外分别锁定，混合场景需使用同一视觉族。">
                {showInterior ? (
                    draft.interiorLookCard ? (
                        <LookCardEditor
                            title="室内 Look Card"
                            value={draft.interiorLookCard}
                            otherCard="exteriorLookCard"
                            onChange={(interiorLookCard) => update((current) => ({ ...current, interiorLookCard }))}
                            onRemove={() => update((current) => ({ ...current, interiorLookCard: undefined }))}
                        />
                    ) : (
                        <Button size="small" onClick={() => update((current) => ({ ...current, interiorLookCard: createSceneLookCardDraft() }))}>
                            添加室内色卡
                        </Button>
                    )
                ) : null}
                {showExterior ? (
                    draft.exteriorLookCard ? (
                        <LookCardEditor
                            title="室外 Look Card"
                            value={draft.exteriorLookCard}
                            otherCard="interiorLookCard"
                            onChange={(exteriorLookCard) => update((current) => ({ ...current, exteriorLookCard }))}
                            onRemove={() => update((current) => ({ ...current, exteriorLookCard: undefined }))}
                        />
                    ) : (
                        <Button size="small" onClick={() => update((current) => ({ ...current, exteriorLookCard: createSceneLookCardDraft() }))}>
                            添加室外色卡
                        </Button>
                    )
                ) : null}
                {!showInterior && !showExterior ? (
                    <div className="flex flex-wrap gap-2">
                        <Button size="small" onClick={() => update((current) => ({ ...current, interiorLookCard: createSceneLookCardDraft() }))}>
                            添加室内色卡
                        </Button>
                        <Button size="small" onClick={() => update((current) => ({ ...current, exteriorLookCard: createSceneLookCardDraft() }))}>
                            添加室外色卡
                        </Button>
                    </div>
                ) : null}
            </EditorGroup>
        </>
    );
}

function SceneAssetReferenceEditor({
    title,
    value,
    assetOptions,
    viewOptions,
    evidenceOptions,
    onChange,
}: {
    title: string;
    value: SceneAssetReferenceDraft;
    assetOptions: Array<{ value: string; label: string }>;
    viewOptions: Array<{ value: string; label: string }>;
    evidenceOptions: Array<{ value: string; label: string }>;
    onChange: (value: SceneAssetReferenceDraft) => void;
}) {
    return (
        <div className="space-y-2">
            {title ? <span className="block text-[var(--fs-caption)] font-medium text-foreground/68">{title}</span> : null}
            <div className="grid grid-cols-2 gap-2">
                <SelectField label="项目资产" value={value.assetId} options={assetOptions} onChange={(assetId) => onChange({ ...value, assetId })} />
                <TextField label="资产 ID" value={value.assetId} onChange={(assetId) => onChange({ ...value, assetId })} />
                <TextField label="资源 URI" value={value.uri} onChange={(uri) => onChange({ ...value, uri })} />
                <TextField label="资产角色" value={value.role} onChange={(role) => onChange({ ...value, role })} />
                <SelectField label="所属 View" value={value.viewId} options={viewOptions} onChange={(viewId) => onChange({ ...value, viewId })} />
                <SelectField label="证据" value={value.evidenceRef} options={evidenceOptions} onChange={(evidenceRef) => onChange({ ...value, evidenceRef })} />
            </div>
        </div>
    );
}

function EvidenceEditor({ item, index, assetOptions, onChange, onRemove }: { item: EvidenceDraft; index: number; assetOptions: Array<{ value: string; label: string }>; onChange: (item: EvidenceDraft) => void; onRemove: () => void }) {
    return (
        <EditorItem title={item.id || `证据 ${index + 1}`} onRemove={onRemove}>
            <div className="grid grid-cols-2 gap-2">
                <TextField label="Evidence ID" value={item.id} onChange={(id) => onChange({ ...item, id })} />
                <TextField label="类型" value={item.kind} onChange={(kind) => onChange({ ...item, kind })} />
                <SelectField label="项目资产来源" value={item.sourceRef} options={assetOptions} onChange={(sourceRef) => onChange({ ...item, sourceRef })} />
                <TextField label="来源引用" value={item.sourceRef} onChange={(sourceRef) => onChange({ ...item, sourceRef })} />
            </div>
            <CheckboxField label="已人工确认该证据" checked={item.verified} onChange={(verified) => onChange({ ...item, verified })} />
        </EditorItem>
    );
}

function LookCardEditor({ title, value, otherCard, onChange, onRemove }: { title: string; value: SceneLookCardDraft; otherCard: "interiorLookCard" | "exteriorLookCard"; onChange: (value: SceneLookCardDraft) => void; onRemove: () => void }) {
    return (
        <div className="border-t border-border/60 py-3">
            <div className="mb-2 flex items-center justify-between gap-2">
                <span className="text-xs font-semibold text-foreground/80">{title}</span>
                <Button size="small" type="text" danger onClick={onRemove}>
                    移除
                </Button>
            </div>
            <div className="space-y-2">
                <div className="grid grid-cols-2 gap-2">
                    <TextField label="视觉族 ID" value={value.lookFamilyId} onChange={(lookFamilyId) => onChange({ ...value, lookFamilyId })} />
                    <SelectField
                        label="继承"
                        value={value.inheritsFrom}
                        options={[
                            { value: "", label: "不继承" },
                            { value: otherCard, label: otherCard === "interiorLookCard" ? "室内 Look Card" : "室外 Look Card" },
                        ]}
                        onChange={(inheritsFrom) => onChange({ ...value, inheritsFrom })}
                    />
                    <TextField label="光源方向" value={value.lightingDirection} onChange={(lightingDirection) => onChange({ ...value, lightingDirection })} />
                    <TextField label="色温" value={value.lightingTemperature} onChange={(lightingTemperature) => onChange({ ...value, lightingTemperature })} />
                    <TextField label="反差" value={value.contrastLevel} onChange={(contrastLevel) => onChange({ ...value, contrastLevel })} />
                    <TextField label="时间" value={value.timeOfDay} onChange={(timeOfDay) => onChange({ ...value, timeOfDay })} />
                    <TextField label="天气" value={value.weather} onChange={(weather) => onChange({ ...value, weather })} />
                </div>
                <TextListField label="基础色板" values={value.basePalette} onChange={(basePalette) => onChange({ ...value, basePalette })} placeholder="每行一个色值，例如 #1A2024" />
                <TextListField label="强调色板" values={value.accentPalette} onChange={(accentPalette) => onChange({ ...value, accentPalette })} />
                <TextListField label="阴影色板" values={value.shadowPalette} onChange={(shadowPalette) => onChange({ ...value, shadowPalette })} />
                <TextListField label="霓虹色板" values={value.neonPalette} onChange={(neonPalette) => onChange({ ...value, neonPalette })} />
                <TextListField label="材质规则" values={value.materialRules} onChange={(materialRules) => onChange({ ...value, materialRules })} />
                <TextListField label="渲染规则" values={value.renderingRules} onChange={(renderingRules) => onChange({ ...value, renderingRules })} />
                <TextListField label="禁止渲染" values={value.forbiddenRendering} onChange={(forbiddenRendering) => onChange({ ...value, forbiddenRendering })} />
                <CheckboxField label="锁定视觉连续性" checked={value.continuityLock} onChange={(continuityLock) => onChange({ ...value, continuityLock })} />
            </div>
        </div>
    );
}
