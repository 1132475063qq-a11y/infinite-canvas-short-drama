# AI Creative Studio（短剧 + 电商）无限画布：继续开发入口

更新时间：2026-08-17

## 1. 先从这里开始

新 Codex 会话必须先完成以下检查，再修改代码：

```bash
cd <repository-root>
git status -sb
git branch --show-current
git log -8 --oneline
```

预期分支：

```text
codex/phase4-film-nodes
```

主规划：仓库外的 Film Production Canvas v2 设计文档只作为来源证据；当前实现与验收以仓库内合同、代码和 `pending-test.mdx` 为准。

参考源及使用边界：

- 原始主工程参考：当前仓库是继续开发主体。
- Provider Adapter、异步任务、Polling、WebSocket、Callback 参考归档：只借鉴协议和失败处理，不复制整体架构。
- 影视 Agent Team 参考归档：只作为 Agent、Skill、Artifact、QC、Retry、LOCKED、Change Request 的知识来源；不要直接运行文件模式。

合并产品的 Ecommerce 实施附录：

- `docs/production-canvas/ecommerce-prototype-addendum.md`：当前 Ecommerce Domain 的冻结合同、Prototype A/B 边界和下一步编码范围。
- `docs/production-canvas/ecommerce-agent-team-v1.2.1-codex-ready.md`：完整架构基准、P0 执行协议和失败/成本/数据边界。
- `docs/production-canvas/ecommerce-prototype-implementation-spec-v1.0.md`：直接给 Codex 的 Prototype 任务书。
- `docs/production-canvas/film-generation-request-contract.md`：Film `generation_request` 的版本、来源、安全和 Task 边界。
- Ecommerce 合并接管提示词和架构审查记录：均为仓库外来源证据；不把其中的设计文字当成已实现能力。

## 2. 产品边界

目标是以影视生产和电商创意生产为两个隔离 Domain 的 AI Creative Studio，不是普通无限画布、ComfyUI 节点工具或模型集合器。

必须坚持：

- 后端领域对象、Artifact、Task、Version 才是生产事实源。
- Canvas 是 Projection 和交互入口。
- 不推翻现有 React、Go、PostgreSQL、Redis 和 Canvas 工程。
- API Key 只能保存在后端 Secret/配置层，不能写入前端节点或提交到 Git。
- 第一条真实闭环使用《寄生广告》SC01。
- Ecommerce 不新建第二套画布；它复用 Shared Canvas Core，但不得把电商字段塞进 Film Domain 或 `FilmArtifact`。

## 3. 已完成状态

### Phase 0：基线（部分完成）

- 已创建基线 tag：`baseline/open-ai-canvas-main-20260811`。
- 已加入静态 Golden Fixture：`web/test/fixtures/parasite-ad-sc01.ts`。
- 已有 Golden Fixture 自动测试。
- 尚缺：独立数据库 schema 导出、恢复说明、数据库支持的《寄生广告》真实 Golden Project。

### Phase 1：Film Semantic Layer（基础完成）

已完成：

- `FilmNodeKind`
- `FilmNodeDomainRef`
- `FilmNodeState`
- `FilmEdgeType`
- 简化 Typed Port
- Scene、Shot、Character 生产节点投影
- 后端 Scene/Shot/Asset ID 绑定
- 右侧基础 Film Inspector
- 老 Canvas 节点兼容

### Phase 1.5：语义契约冻结（完成）

已完成：

- 冻结 26 种 `FilmNodeKind v2`；保留规划 Phase 4 明确需要的 `acting`，并补齐细分节点类型。
- 新增穷举的 `FilmPortSchema`：direction、type、role、required、multiple、accepts。
- Typed Connection 在原有影视规则矩阵上增加端口兼容和单输入容量校验，旧自由连接继续兼容。
- 统一 `CanvasDocumentV2`：schemaVersion、projectId、nodes、connections、viewport、groups、layout.gridSize。
- 旧 `edges`、`layout.grid` 和缺省字段可迁移为唯一 V2 表达；未知未来节点和更高 schemaVersion 保留。
- Store 创建、导入、恢复、节点更新及项目画布 fallback 已接入 V2 字段和 group projection。
- 新增 `CanvasDocumentV2` 迁移测试、端口合同测试，并将 Golden Fixture 升级为完整 V2 文档。

核心目录：

```text
web/src/film/domain/
web/src/film/inspector/
```

### Phase 2：Layout Engine（核心验收完成）

已完成：

- Grid Off / 8 / 16 / 24，设置持久化及历史恢复
- Grid Snap
- 6 屏幕像素 Alignment Guide
- 8 屏幕像素 Equal-gap Guide
- 左/中/右、上/中/下对齐
- 横向/纵向分布
- Spatial Index
- Collision Avoidance
- Scene Lane / Shot Lane
- Result 自动避让
- auto / manual / pinned
- 单事务布局提交

核心目录：

```text
web/src/lib/canvas/layout/
web/test/canvas-layout-engine.test.ts
```

### Phase 3：Production UI Shell（完成）

已完成：

- 左侧 15 项 Project Navigator：总览、故事、剧本、场景、角色、场地、道具、分镜、镜头、资产、声音、QC、Agent、待处理、交付。
- 顶部 Project Production Status：镜头进度、运行任务、QC 失败、Needs You。
- 右侧统一 Inspector Header 与按对象类型切换的 Tabs；Shot 已显示 DomainRef、Shot Contract、状态与证据。
- 底部当前 Scene 的 Shot Strip；点击已投影镜头会选中并定位画布节点。
- 影视导航、Shot Strip、Inspector 共用后端 Project/Scene/Shot/Asset/Task 与 Canvas Projection，不建立第二套事实源。
- 画布视图控制已避开 Shot Strip，不遮挡底部镜头卡。

核心目录：

```text
web/src/film/panels/
web/src/film/inspector/
web/src/components/canvas/canvas-project-sidebar.tsx
web/src/pages/canvas/canvas-project-top-bar.tsx
web/test/production-ui-shell.test.ts
```

### Phase 4/5：影视生产节点完成，Provider Gateway 单渠道事实链已落代码、待运行验证

已完成本批：

- Scene 增加内外景、时间与场地资产绑定。
- Shot 使用独立 `FilmArtifact` 保存不可变 Shot Contract 版本；Shot 只持有当前版本指针。
- 历史 Shot 首次读取时安全补齐 Shot Contract v1，新旧画布共用同一事实源。
- Character、Location、Prop 节点读取当前 AssetVersion 定义；Location/Prop 的 Inspector 保存会新增资产版本。
- Shot、Character、Location、Prop 使用领域专用节点卡；工具栏加入场地和道具入口。
- Inspector 已接通 Scene/Shot/Character/Location/Prop 的真实保存 API。
- 新增通用、不可变版本的 `FilmArtifact` 事实源，以及按 Shot + ArtifactType 查询当前版本的解析规则。
- 新增 Acting 节点：只负责 objective、obstacle、tactic、beat、performance、continuity locks；服务端拒绝外貌、体型、服装、声音、机位、焦段等越权字段。
- 新增 Video Prompt Pack 节点：保存场景上下文、参考、空间摄影、时间轴、声音表演、锁与最终编译提示词。
- Acting 与 Prompt Pack 每次保存都会创建新的后端 Artifact 版本并递增 Project revision；Canvas 只保存 DomainRef 投影。
- 新增 Shot → Acting、Shot → Prompt Pack、Acting → Prompt Pack 的 typed connection；自动布局按 Shot、Acting、Prompt Pack 横向排列。
- Inspector、领域卡片、节点菜单、刷新恢复和最新版本解析已全部接通。
- 新增 `generation_request` FilmArtifact：每次保存均冻结当前 Prompt Pack 的 Artifact ID、版本和 compiled prompt，作为未来 Provider Gateway 的可复现输入。
- Generation Request 追加 Shot Contract、Prompt Pack 与 Shot AssetVersion 到 SourceRefs；服务端拒绝 API Key、Token、Secret、Credential、Authorization、Password 和 Provider Key 字段。
- 新增 Prompt Pack → Generation Request typed connection、画布创建入口、专用节点卡、Inspector、显式“载入最新 Prompt Pack”动作和 Shot Pipeline 自动布局。
- 新增只读 `Generation Request → Task Draft` 合同：后端总是按画布指定的 Artifact ID 读取精确请求版本，编译出 provider-independent `gatewayInput`、来源、指纹和阻塞原因；它不会创建 Task、计费单、ProviderJob、Result 或 `DomainRef.taskId`。
- Inspector 的 Execution 标签可查看只读任务合同、后端渠道候选、费用确认、显式提交和只读执行历史；Generation Request 可显式保存为 draft / review / ready / locked。只有 ready / locked 可提交。
- 新增服务端 `CanvasProjectionPatch`：Task 绑定独立于浏览器整份 `CanvasProject.PayloadJSON`，只在 Canvas 节点仍精确指向同一 Generation Request Artifact 版本时叠加 `taskId`；浏览器保存会剥离伪造/旧的 `taskId`，整份画布替换也不会擦除仍存在 Canvas 的服务器 Patch。
- 新增单渠道原子提交 API：只接受 Canvas/节点、请求指纹、后端渠道 ID 和模型 key；在同一事务内锁定并复核 Project、Canvas、Generation Request、渠道、能力版本与价格快照，随后原子完成积分预留、标准 `canvas_image` / `canvas_video` Task 和 Projection Patch。
- 同一节点 + 同一 Generation Request 版本重复提交会返回原 Task，不重复预扣；积分不足、目标变化、路由变化或节点已有活动任务时，Task、账单、积分流水和 Patch 全部回滚。Worker 只能在事务提交后看到 Task。
- Task 输入只保存非敏感渠道 ID、模型 key、标准化运行参数和审计快照；API Key、Base URL、Secret 和 Headers 仍只存在后端渠道层。Worker 执行时再次解析密钥，并拒绝协议/能力版本已漂移的任务。
- 原子提交现在同时创建初始 `GenerationAttempt`；Worker 领取、租约恢复、失败、取消和 Retry 都保留独立 Attempt 历史，不再只依赖可变的 `Task.attempts`。终态和完成写入校验租约所有者，失效 Worker 不能关闭后继 Attempt。
- 真实上游任务号现在按 Worker 携带的 Attempt 编号保存为 `ProviderJob`；迟到观察仍归原 Attempt，不能覆盖新 Attempt 的 Task 指针；已确认成功不会被后续噪声轮询降级，已知上游 ID 但没有终态证据的失败会记录为 `uncertain`。
- 影视 Task 成功、Attempt 成功和 `Result(kind = film_generation_result)` 在同一事务落库；Result 关联精确 Generation Request ID/版本、Task、Attempt 与首个可访问媒体 URL。
- 新增只读执行历史 API，并在 Canvas 读取时从后端事实表临时投影 `generationAttemptId`、`providerJobId`、`resultId`；浏览器保存会和 `taskId` 一样剥离这些运行时 ID。
- 登录态浏览器已完成费用确认和显式图片提交，`gpt-image-2` 真实任务、Attempt、ProviderJob、Film Result、Canvas Patch 与积分结算均已验证；人物一致性三张受控图片合计 29/30。视频没有测试。
- 场景空间资产系统已加入 SceneManifest、Topology、FloorPlan、Anchor、View Coverage、Look Card 和结构化 Gate。Gate 依赖结构事实与人工证据，不是后端自动视觉识别。
- 空间 Pack 保存现在要求 `expectedVersion`，过期页面返回冲突；删除领域项目会拦截活动或状态待核对的执行；Film Task 使用显式 `domainProjectId` / `canvasId`，旧数据由 Attempt 回填。

尚未完成：视频 Provider 验收、自动视觉 QC、端用户 Retry/QC UI、跨 Scene 的项目级 Location/开口连续性，以及《寄生广告》数据库 Golden Project。

### Ecommerce Creative Studio：Prototype A Provider-free 合同阶段

已完成：

- 完成 Film V2 + Ecommerce Creative Studio 的合并架构审查和文档同步。
- 冻结 `short-drama` / `ecommerce` 两种 Project Domain 的隔离原则。
- 冻结三个 Ecommerce Agent：`ProductIntelligenceAgent`、`CreativeDirectorAgent`、`SceneDirectorAgent`。
- 冻结两个 Prototype Skill：`still-life.lifestyle-tabletop`、`fashion.natural-walk`。
- 冻结 ProductDNA、CreativeDirection、ScenePlan、CreativeShotPlan、ExpectedRelationGraph、GenerationJob、GeneratedAsset、QAReport 的命名和版本原则。
- 冻结 Template-Based Skill Executor、Provider Bake-off、三层 QA、A/B/C Benchmark 与 GO/MODIFY/STOP 门禁。
- 补齐完整审查记录要求的 P0 顺序：Schema、Skill 文件协议、Executor、ExpectedRelationGraph、Benchmark Evaluation、Provider Evaluation。
- 明确 Benchmark 采用 1–5 分、Blind Preference 和效果量/趋势；不把统计显著性、固定胜率、示例成本或固定保存天数写成硬条件。
- 明确 Prototype UI 可以是 CLI、简单表单、Developer panel 或最小 Canvas 投影，不先做完整 Ecommerce Canvas。

当前明确未完成：

- Ecommerce Agent Runtime、完整 Ecommerce Canvas 和真实 Provider 接入。
- 真实图片 Provider Bake-off、真实生成、媒体质量和人工盲测。

已开始但仍属于 Provider-free 合同层：

- `web/src/ecommerce/**` 已加入 ProductDNA、CreativeDirection、ScenePlan、CreativeShotPlan、ExpectedRelationGraph、Artifact 版本、Skill Executor、Result Grid 和 Basic QA 的纯 TypeScript 合同。
- `web/src/ecommerce/skills/definitions/still-life.lifestyle-tabletop/**` 已加入第一份可加载 Skill 文件协议和四个结构化 variation。
- `web/test/ecommerce-prototype.test.ts` 已覆盖 Prototype A 的四 variation、不可变 revision、关系图绑定和 `UNCERTAIN` QA 语义。
- 后端已加入独立 `EcommerceArtifact` 表、版本追加 API、项目类型校验和权限边界。
- 项目创建页已加入 `short-drama` / `ecommerce` 选择；电商项目进入独立 Prototype A 开发面板，不进入 Film 章节/画布投影。
- 电商项目的资产页使用独立商品资产投影，只允许引用/移除商品媒体，不显示角色卡和短剧专用字段。
- 开发面板可运行无 Provider 合同链，追加保存 ProductDNA、CreativeDirection、ScenePlan、CreativeShotPlan、QAReport，并从后端恢复四槽位 Result Grid。
- `web/src/ecommerce/evaluation/provider-evaluation.ts` 已加入 Provider Evaluation 记录合同：候选能力、统一设置指纹、Case、attempt、结果/成本/失败证据、1–5 分维度、Blind Preference 和显式 GO/MODIFY/STOP 决策；它不会调用真实 Provider 或自动判定 GO。
- `web/test/ecommerce-provider-evaluation.test.ts` 已覆盖计划骨架、设置指纹、成功/失败尝试、评分、盲测引用和显式决策边界；本次未运行测试，按项目验证纪律保留为待验证。
- 当前仍没有宣称 Ecommerce Agent Runtime、真实 Provider、浏览器媒体生成或媒体质量已经完成。

正式边界见：`docs/production-canvas/ecommerce-prototype-addendum.md`。本节区分“合同已冻结”和“Provider-free 合同骨架已实现”，不代表完整功能已实现。

核心文件：

```text
backend/internal/model/models_project.go
backend/internal/service/project_shot.go
backend/internal/repository/repository.go
web/src/film/domain/film-node-resolver.ts
web/src/film/nodes/film-node-card.tsx
web/src/film/inspector/film-inspector.tsx
```

最近五个本地阶段提交：

```text
a0698bd feat(layout): complete phase 2 canvas layout engine
6384aec feat(canvas): focus film connections and scene lanes
b017dcb fix(film): expose scene frame connection handles
50e529b feat(film): enforce typed production connections
e77c747 fix(film): preserve legacy nodes and clean scene records
```

## 4. 已验证范围

最近一次已提交版本的完整验证结果：

- Web 自动测试：70/70 通过。
- TypeScript typecheck：通过。
- Web production build：通过。
- 浏览器实测：Scene Lane、`SC01-SH001 · Shot Contract`、Grid 8→16、刷新持久化、恢复 8 后再次刷新持久化。
- 浏览器控制台只观察到已有 Ant Design deprecated 警告，没有本阶段新增运行错误。

Phase 1.5 当前工作区验证（2026-08-12）：

- Web 自动测试：76/76 通过。
- TypeScript typecheck：通过。
- Web production build：通过。
- `git diff --check`：通过。
- 本阶段没有修改 UI，因此没有新增浏览器交互验收；此前 Phase 1/2 浏览器结果不等于 Phase 1.5 自动测试结果。

Phase 3 验证（2026-08-12）：

- Web 自动测试：78/78 通过。
- TypeScript typecheck：通过。
- Web production build：通过（仅保留已有 chunk-size warning）。
- `git diff --check`：通过。
- 浏览器实测：1920×1080、1440×900、1280×720 均无页面级横向或纵向溢出；左侧 15 项导航、顶部状态、底部 Shot Strip、右侧 Inspector 正常显示。
- 浏览器交互：点击 Shot Strip 可选中真实 Shot Projection，导航同步到“镜头”，Inspector 显示 Shot ID、Scene ID、状态与 Evidence；Camera Tab 可切换。
- 历史验收时 `localhost:3000` 被另一份 checkout 占用，因此使用 `http://localhost:3001`；不要把其他本地页面误当成本分支。
- 本阶段页面没有新增运行错误；控制台只观察到已有 Ant Design Modal deprecated warning。

Phase 4 第一批验证（2026-08-12）：

- 后端 repository/service 自动测试：通过。
- Web 自动测试：79/79 通过；typecheck、production build、`git diff --check` 通过。
- 浏览器实测：历史空数组项目可正常打开；Shot Card 显示真实 Shot Contract、时长、景别与状态。
- Inspector 实测：Shot Contract v1 历史补齐成功；编辑保存生成 v2/v3，刷新后仍从后端版本读取。
- 新增节点菜单可见：影视场景、影视镜头、角色资产、场地资产、道具资产。

Phase 4 Acting / Prompt Pack 验证（2026-08-13）：

- 后端 repository/service/handler 自动测试：通过。
- Web 自动测试：88/88 通过；TypeScript typecheck、production build、`git diff --check` 通过。
- 浏览器真实创建：选中 Shot 后可创建 Acting v1，再保存 objective、obstacle、tactic、beat 为 v2。
- 浏览器真实创建：从 Acting 创建 Prompt Pack v1，保存场景上下文、格式、风格和 compiled prompt 为 v2。
- 刷新后节点总数、Acting v2、Prompt Pack v2 和字段内容均从后端/Canvas 投影正确恢复；搜索可定位远端节点。
- 数据库核对：Shot → Acting、Shot → Prompt Pack、Acting → Prompt Pack 三条连接均持久化；横向坐标顺序为 Shot、Acting、Prompt Pack。
- 最终新浏览器会话无 console error/warning；本阶段同时修复表单受控状态和两个 Ant Design Modal 弃用属性。
- 当前仓库继续使用 `http://localhost:3001` 验收，避免与 3000 端口的其他 checkout 混淆。

常用验证命令：

```bash
cd <repository-root>/web
bun run test
bun run typecheck
bun run build
```

开发环境启动：

```bash
cd <repository-root>
LOCAL_UID=$(id -u) LOCAL_GID=$(id -g) docker compose -f docker-compose.dev.yml up --build
```

浏览器地址：`http://localhost:3000`

不要删除 Docker 数据卷；当前开发数据应继续保留。

## 5. 规划审查结论

详细规划的总体方向正确，但它同时包含“当前事实”和“未来目标”，不能把规划文字直接当作已实现能力。

当前后端确实已有：

- Project、ProjectUnit、Scene、Shot
- Asset、AssetVersion、ShotAssetReference
- Workflow、Task、Message、Result
- 任务 Polling、Cancel、文本 SSE

当前尚没有正式独立模型：

- 通用 Agent Artifact / ArtifactVersion（影视节点当前使用的是已落库的 `FilmArtifact` 不可变版本行）
- QCReport
- ChangeRequest
- AgentProfile / Topic

这些属于后续阶段，不得提前宣称完成。

## 6. 下一步：Film Phase 5 与 Ecommerce Prototype A 并行但隔离

不要越过当前单渠道原子边界直接接入多家 Provider 或完整生成系统。最新版详细规划明确规定：

```text
Film：
Phase 3 = UI Shell
Phase 4 = Film Nodes
Phase 5 = Provider Gateway v2

Ecommerce：

文档合同冻结
→ Prototype A 无 Provider 运行骨架
→ Provider Evaluation 记录合同（已完成）
→ 真实 Provider Bake-off（下一步）
→ Prototype A 真实生成/QA
→ Prototype B
→ GO / MODIFY / STOP
```

Phase 3 已通过自动与浏览器验收，下一会话不要重做 UI Shell。Phase 0 仍需另补数据库 schema 快照和恢复说明。

### Phase 4 已完成 / Phase 5 当前边界

已完成：

- 已接入 `GenerationRequest` 版本事实模型、Prompt Pack typed port、Inspector、版本/状态显示和基础布局。
- 已建立只读 `GenerationRequest → Task Draft` 合同，冻结精确 Artifact 版本、Gateway 输入、任务类型、状态/阻塞原因与指纹；Canvas 仍只做 Projection。
- 已建立只读 Provider Route Catalog：仅从后端 system channel 中列出与冻结请求媒体类型匹配的模型，并返回脱敏的能力、计费和就绪状态；它不创建 Task、不预扣积分、不调用 Provider，也不回写 Canvas。
- 已建立 revision-safe `CanvasProjectionPatch`：服务端 Task 绑定与完整浏览器 Canvas 文档分离，读取时仅按精确 Generation Request 版本叠加，浏览器无法伪造或擦除绑定。
- 已建立图片/视频单渠道原子提交：精确请求、服务端路由、能力/价格、Task、积分预留和 Patch 在事务边界内复核并提交；重复请求幂等，失败全回滚。
- 已建立 Attempt/ProviderJob/Film Result 事实链：提交创建 queued Attempt，Worker 领取、租约恢复和 Retry 保留编号历史，上游任务号归属具体 Attempt，成功 Task 与 Result 同事务落库。
- 已建立只读执行历史 API 和 Canvas 运行时投影；浏览器文档不拥有 Task、Attempt、ProviderJob 或 Result ID。
- API Key 继续只存在后端 Secret/渠道配置层，任何 Generation 节点都不得保存密钥。

下一批只可讨论：

- 先为 Inspector 增加明确的费用确认、提交和执行历史交互；读取 Inspector 不得自动提交。
- 再使用非生产测试输入完成一个受控单 Provider 接受测试，明确区分 API 状态、Result 持久化和真实媒体质量。
- 不要同时接入多家 Provider、QC/Retry 全链或 Agent Runtime。

优先复用并增量升级：

- `web/src/film/nodes/film-node-card.tsx`
- `web/src/film/inspector/film-inspector.tsx`
- `web/src/film/domain/`
- 后端 Scene / Shot / Asset / AssetVersion 现有服务与 API

Ecommerce 下一步只做：

- 在现有 Provider Evaluation 记录合同和开发面板基础上，接入用户明确提供且实际可调用的候选渠道，固定统一输入、成本记录和盲测数据结构并完成真实 Bake-off。
- 保持 `web/src/ecommerce/**` 和后端 Ecommerce 合同独立，不修改 `web/src/film/**` 的语义。
- 真实生成前不扩展完整 Ecommerce Canvas，不接入多家 Provider，也不实现完整 Ecommerce V1。
- 不提前接入多家 Provider，不实现完整 Ecommerce V1，不把 Ecommerce 目标写成已完成。

## 7. 明确未完成的 Phase 2 扩展项

核心 Phase 2 用例已经通过，但详细设计中以下内容仍应记录为待办，不得误报完成：

- 用户可见的 `Tidy Selection`、`Tidy Scene`、`Tidy Project` 三个独立入口。
- 完整 subgraph 分层、拓扑排序、业务排序、最小位移优化。
- Edge Routing。
- 真实浏览器拖拽方式的 equal-gap、多选分布验收。
- 1000 节点真实 UI 性能基准；现有 1000 项 Spatial Index 测试不等于浏览器性能证明。

## 8. 新会话执行规则

1. 先读本文件和完整规划。
2. 先运行 `git status -sb`，不要覆盖用户改动。
3. 不重做已完成的 Phase 1/Phase 2。
4. 每次只做当前阶段，不提前接模型、不提前实现复杂 Agent Runtime。
5. 按 `AGENTS.md` 的当前验证纪律执行；默认不自动运行测试、typecheck、build，用户明确要求后再选择最小充分验证。
6. UI 功能必须做真实浏览器交互验证；静态测试不能替代点击、拖拽和刷新验证。
7. 对验收结果明确区分：代码存在、自动测试通过、浏览器交互通过、真实 Provider/媒体质量通过。
8. 进入 Ecommerce 工作前必须先读 `docs/production-canvas/ecommerce-prototype-addendum.md`；Ecommerce 当前 Gate 是 `READY FOR PROTOTYPE IMPLEMENTATION`，不是 `READY FOR FULL V1`。
9. Film 与 Ecommerce 共享 Canvas Core，但必须保持 Domain、Artifact、Skill、Evaluator 和权限隔离。

## 9. 给新 Codex 会话的第一条指令

```text
请先完整读取 CONTINUE-HERE.md、
docs/production-canvas/ecommerce-prototype-addendum.md 和
当前环境可用的仓库外 Film 主规划（仅作为设计来源）。
检查当前分支、Git 状态和最近提交，不要重做 Phase 1/Phase 2。
Film Phase 3 已完成并通过 78/78 自动测试与三档浏览器验收，不要重做。
Phase 4 已完成 Scene、Shot、Character、Location、Prop、Acting、Prompt Pack、Generation Request、只读 Task Draft、只读 Provider Route Catalog、revision-safe Canvas Projection Patch，以及图片/视频单渠道原子 Task 提交边界。
Phase 5 的 GenerationAttempt / ProviderJob / Film Result 数据合同、事务写入、Retry 历史、只读执行 API、Canvas 运行时投影和费用确认提交 UI 已写入当前分支；受控图片接受测试已完成，但本次空间、视频投影与渠道目录改动仍需针对性验证，视频尚未验收。Film 下一步先完成这些回归，再做端用户 Retry/QC UI、跨 Scene 连续性与数据库 Golden Project；不直接接入多家 Provider。Ecommerce 已完成 Prototype A 无 Provider 合同骨架和 Provider Evaluation 记录合同，下一步只在用户提供真实渠道/素材后执行 Bake-off。两条线都不要提前接入 Full Ecommerce V1 或复杂 Agent Runtime。
```
