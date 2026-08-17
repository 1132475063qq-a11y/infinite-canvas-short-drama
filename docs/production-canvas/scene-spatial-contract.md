# 场景空间连续性合同

## 目标与边界

`scene-asset-design` 不再输出一组松散参考图，而是输出可版本化、可验证、可供镜头生成读取的空间模型。

核心规则：

> 视频生成模型只能在锁定空间内表演和摄影，不负责重新设计空间。

本合同只管理场景事实、资产证据与确定性门禁，不调用图片或视频 Provider。场景资产没有通过 `Spatial Continuity Gate` 时，不得进入正式镜头生成。

## 当前实现状态

| 状态 | 当前边界 |
| --- | --- |
| `IMPLEMENTED` | 场景级 Pack/Gate Schema、解析与原子版本持久化、确定性 Validator、`PASS / UNCERTAIN / FAIL / NEEDS_YOU`、服务端 Prompt Pack 空间编译、Generation Request/Task Draft/原子提交门禁，以及场景节点 `Spatial` 标签中的结构化 Pack 编辑器与 Gate 问题定位。 |
| `TEST CONTRACT` | 已写入黑市诊所完整室内外拓扑、平面区域、门窗、固定/可移动锚点、8 个 Camera/View、可达路径，以及 PASS、FAIL、UNCERTAIN、NEEDS_YOU、版本漂移、伪造 Prompt 和过期空间版本回归。历史 Go 验收记录见 `pending-test.mdx`；当前工作树仍需复跑，不能把测试文件或历史结果描述成当前分支的完整验收。 |
| `MOCKED` | 黑市诊所中的资产 ID 和证据是结构化测试 Fixture，不是模型生成图片，也不是 Provider 媒体质量证明。 |
| `MANUAL ONLY` | 当前由用户在场景节点的 `Spatial` 标签中填写 SceneManifest、拓扑、平面图、锚点、机位、视角、资产证据和 Look Card；资产审批、证据确认和 Gate Issue 修正仍需人工或外部编排。 |
| `NOT IMPLEMENTED` | `production-breakdown` 自动产出 SceneManifest、自动 Missing View Detection、真实图片/视频空间 E2E、自动成片视觉诊断执行器。 |
| `FUTURE` | SceneHierarchy 细化、按 Shot Requirements 自动补齐缺失视角、正式 Agent Runtime/Skill 编排和 QC/Retry 执行器。 |

仓库当前没有 `scene-asset-design`、`production-breakdown`、`lira-image-prompts`、`cinedance-video-director`、`ai-shot-diagnostics` 这五个命名 Skill 的可执行定义。本文为它们规定数据合同和下游边界，不能把合同存在描述成 Agent Runtime 已接通。

## 生产顺序

```text
SCRIPT
  ↓
production-breakdown
  ↓
SceneManifest
  ↓
SceneTopologyGraph
  ↓
SpatialFloorPlan
  ↓
Character / Prop Anchors
  ↓
CameraAnchorPlan
  ↓
ViewpointCoverageMatrix
  ↓
MasterSceneAsset + Required ViewAssets
  ↓
Interior / Exterior SceneLookCard
  ↓
Spatial Continuity Gate
  ↓ PASS
LOCKED_SCENE_ASSET_PACK
  ↓
Image Prompt / Video Director / Shot Diagnostics
```

目标流程禁止从剧本中的一个地点词直接进入生图；`production-breakdown` 应先明确剧情会经过的空间、事件、人物、道具、前后场景与出入口。当前仓库尚未实现这一步的自动生成器，SceneManifest 由调用方显式提交并由 Gate 校验。

## Artifact 模型

空间事实使用 `FilmArtifact` 的场景作用域：

```text
scope = scene
scopeId = sceneId
artifactType = scene_asset_pack | spatial_continuity_gate
```

`scene_asset_pack` 是来源事实，`spatial_continuity_gate` 是后端确定性校验结果。两者在同一场景锁和数据库事务中追加版本，Gate 记录精确的 `packArtifactId`、`packArtifactVersion` 和 `packVersion`。下游不会再创建重复的 `LockedSceneAssetPack` Artifact；它使用 Prompt Pack 内由服务端生成的只读投影。

Pack 的核心结构：

```json
{
  "schemaVersion": 1,
  "sceneId": "scene-clinic",
  "sceneManifest": {},
  "sceneTopologyGraph": {},
  "spatialFloorPlan": {},
  "fixedAnchors": [],
  "movableAnchors": [],
  "cameraAnchorPlan": {},
  "viewpointCoverageMatrix": {},
  "masterSceneAsset": {},
  "viewAssets": [],
  "interiorLookCard": {},
  "exteriorLookCard": {},
  "forbiddenChanges": [],
  "authorityConflicts": [],
  "evidence": [],
  "requiredPaths": []
}
```

### SceneManifest

记录 `sceneId`、父地点、内外景、剧情事件、人物、必需道具、出入口、前后场景和连续性优先级。剧情事件不能为空，否则 Gate 至少为 `UNCERTAIN`。

### SceneTopologyGraph

节点代表房间、门口、街角、街道等可辨识空间；边记录方向、距离、可见性、高差、转场与门窗关系。人物或道具的 `requiredPaths` 必须在有向拓扑中可达。

### SpatialFloorPlan

使用局部逻辑坐标系记录范围、功能区和门窗开口。功能区可用 `topologyNodeId` 映射到拓扑节点，门窗 `connectsTo` 必须指向存在且有拓扑边连接的空间。室内场景必须提供平面图；室外场景也使用同一二维布局合同描述可拍区域。固定锚点、可移动锚点和机位不得越界。

### Anchors 与 CameraAnchorPlan

固定锚点用于墙、门窗、操作台、患者椅等不能漂移的对象；可移动锚点用于允许调度但仍需追踪的人物和道具。机位记录 `sceneId`、`topologyNodeId`、位置、朝向、景别范围、轴线侧和反打关系。正反打机位必须互相声明、位于同一动作轴侧，并在可识别方位下保持相反朝向。

### ViewpointCoverageMatrix

视角由剧情和镜头需求驱动，不机械生成正面、侧面、背面和 45 度图。每个必需视角必须引用存在的机位、说明用途、绑定存在且属于该 View 的资产，并确保必需/禁止对象都来自当前场景锚点。

### SceneLookCard

室内、室外分别锁定基础色、强调色、阴影色、霓虹色、光源方向、色温、反差、时间天气、材质规则与禁止渲染项。混合场景用相同 `lookFamilyId` 建立视觉族，并可用 `inheritsFrom` 声明内外景继承。被连续性锁定的 Look Card 任一字段都不允许静默漂移。

## 下游空间编译边界

场景关联的 Prompt Pack 保存时，浏览器只能提交创作提示和 `viewId` / `cameraAnchorId` 选择。服务端会重新读取当前 `PASS` 且 `ready/locked` 的 Pack/Gate，生成 `lockedSceneAssetPack` 投影，并将以下精确版本写入 Prompt Pack：

```json
{
  "spatialPackArtifactId": "scene-pack-vN",
  "spatialPackArtifactVersion": 3,
  "spatialGateArtifactId": "scene-gate-vN",
  "spatialGateArtifactVersion": 3,
  "cameraAnchorId": "C01",
  "viewId": "V01",
  "spatialBindingStatus": "PASS"
}
```

`compiledPrompt` 由服务端把创作提示与 JSON 空间投影拼接而成，浏览器不能提交或覆盖最终编译结果。投影包含 Scene Manifest、Topology Graph、Floor Plan、固定/可移动锚点、选定机位及反打机位、视角覆盖、主场景/视角资产、Look Card、必需路径和禁止变化。

Generation Request 只能冻结该 Prompt Pack 的同一版本；Task Draft 和原子提交会再次检查投影引用是否仍指向当前 `PASS` Pack/Gate。场景版本变化后，旧 Prompt Pack 只会得到阻断，必须重新选择视角并编译。无 `SceneID` 的历史镜头仍沿用旧提示词兼容路径，但不会被宣称为拥有空间模型。

## 门禁状态

| 状态 | 含义 | 是否允许正式生成 |
| --- | --- | --- |
| `PASS` | 结构、证据、覆盖和连续性检查全部通过 | 是 |
| `UNCERTAIN` | 缺少结构、证据或必要视角，无法证明连续性 | 否 |
| `FAIL` | 存在确定性矛盾、漂移、越界或不可达路径 | 否 |
| `NEEDS_YOU` | 权威来源冲突，必须由用户裁定 | 否 |

主要校验包括：

- Scene Manifest、拓扑、平面图、机位和视角覆盖是否完整；
- 门窗、固定家具、空间比例、左右关系与反打机位是否成立；
- 平面区域和 Camera Anchor 是否映射到存在的拓扑节点；
- 必需视角绑定的资产、可见对象和禁止对象是否真实存在且没有冲突；
- 人物和道具路线是否物理可达；
- 室内外连接及同一开口的关系是否一致；
- 光源方向和 Look Card 是否漂移；
- 主场景图和必需视角是否绑定已验证证据；
- 是否仍混用旧版自由文本 Location Definition；
- 是否存在需要人工裁决的权威冲突。

`draft` / `review` 允许保存未通过的 Pack 以便修正；`ready` / `locked` 只接受结构校验为 `PASS` 的 Pack。反过来，结构完整但仍处于 `draft` / `review` 的 Pack 会得到 `SCENE_PACK_NOT_READY`，Gate 保持 `UNCERTAIN`，不能进入正式生成。失败草稿会成为当前可见版本并暂停生成，但不会替换最近一次 `PASS` 且已 `ready` / `locked` 的连续性基线，因此修正后可以回到已锁定布局。并发保存使用期望版本校验，过期写入返回冲突。

## HTTP 接口

```text
GET  /projects/:projectId/scenes/:sceneId/asset-pack
POST /projects/:projectId/scenes/:sceneId/asset-pack
```

POST 请求示例：

```json
{
  "status": "draft",
  "responsibleAgentId": "scene-asset-design",
  "expectedVersion": 0,
  "payload": {
    "schemaVersion": 1,
    "sceneId": "scene-clinic"
  }
}
```

路径中的 `sceneId` 是权威目标，客户端 Payload 不能改写它。`expectedVersion` 必须等于页面当前加载的 Pack 版本；首次保存为 `0`。后端不会替客户端读取最新版后继续保存，过期页面会收到 `409 Conflict`，从而阻止旧页面覆盖新版本。保存操作原子返回 `packArtifact`、`gateArtifact` 和解析后的 `gate`。

## 生成提交门禁

关联 `SceneID` 的 Generation Request 在任务草稿阶段读取最新 Pack/Gate，并将两者加入 Gateway 来源引用。只有 Gate 为 `PASS`，请求才会进入 Provider 路由和提交。

原子提交事务再次锁定并核对：

1. Generation Request 仍是精确的 `ready` / `locked` 版本；
2. Gate Artifact ID/版本没有变化；
3. Gate 仍为 `PASS`；
4. Gate 指向当前最新 Pack 的精确 Artifact ID/版本；
5. 场景、画布、渠道、能力和计费快照仍与提交前一致。

任一条件变化都拒绝提交，不创建 Task、不预留积分、不写 Canvas Patch。没有 `SceneID` 的历史镜头暂时保留兼容行为，但其 `spatialGateReady` 不代表已经拥有空间模型；新建镜头应始终关联场景。

## 下游消费规则

- `production-breakdown` 负责 Scene Manifest 的剧情事实来源；
- `scene-asset-design` 负责补全空间包并处理 Gate Issue，不直接绕过门禁；
- 图像提示词和视频导演通过 Prompt Pack 的只读空间投影消费锁定 Pack、机位、视角和 Look Card；Provider 只接收编译后的结果；
- 视频导演只能选择已声明 Camera Anchor 和可达调度路径；
- 镜头诊断错误码与 Gate Issue 的映射已在本合同中固定，但仓库当前没有独立的自动成片诊断执行器，不能把合同映射描述成自动 QC 结果；
- Provider 不得修改、补猜或覆盖空间合同。

诊断合同接受 `FIXED_PROP_DRIFT`、`DOOR_POSITION_DRIFT`、`WINDOW_POSITION_DRIFT`、
`CAMERA_GEOMETRY_CONFLICT`、`REVERSE_SHOT_CONFLICT`、`SCENE_TOPOLOGY_CONFLICT`、
`FLOORPLAN_CONFLICT`、`LOOK_DRIFT`、`PATH_IMPOSSIBLE` 和
`SCENE_REDESIGN_DETECTED`，并归一到现有 Gate Issue 词表。未知诊断保持
`SCENE_DIAGNOSTIC_UNMAPPED`，不能静默当作通过。

黑市诊所用例是首个结构化回归基准：覆盖诊所内部、诊所门、门口、外墙、街角、主街和右侧街道延伸，锁定操作台、患者椅、桥接椅、垃圾桶、墙面标志和管线，并提供 8 个剧情视角。完整包应为 `PASS`；固定锚点、门窗、Look Card、无效视角资产或正反打跨轴应为 `FAIL`；缺证据或视角应为 `UNCERTAIN`；权威冲突应为 `NEEDS_YOU`。该 Fixture 不调用或伪造图片/视频模型结果。
