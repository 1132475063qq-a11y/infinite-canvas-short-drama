# AI 短剧生产无限画布：继续开发入口

更新时间：2026-08-12

## 1. 先从这里开始

新 Codex 会话必须先完成以下检查，再修改代码：

```bash
cd /Users/xiangyuqin/Downloads/infinite-canvas-short-drama
git status -sb
git branch --show-current
git log -8 --oneline
```

预期分支：

```text
codex/phase1-film-semantic
```

主规划文档：

```text
/Users/xiangyuqin/Downloads/短剧AI无限画布_Production_Canvas_v2_超详细发展规划.md
```

参考源及使用边界：

- `/Users/xiangyuqin/Downloads/open-ai-canvas-main/`：主工程原始参考；当前仓库是继续开发主体。
- `/Users/xiangyuqin/Downloads/Infinite-Canvas-main.zip`：只参考 Provider Adapter、异步任务、Polling、WebSocket、Callback，不复制整体架构。
- `/Users/xiangyuqin/Downloads/影视短剧AgentTeam_v1.3.1_Conflict_Hardened.zip`：只作为 Agent、Skill、Artifact、QC、Retry、LOCKED、Change Request 的影视知识参考；不要直接运行文件模式。

## 2. 产品边界

目标是影视生产操作系统，不是普通无限画布、ComfyUI 节点工具或模型集合器。

必须坚持：

- 后端领域对象、Artifact、Task、Version 才是生产事实源。
- Canvas 是 Projection 和交互入口。
- 不推翻现有 React、Go、PostgreSQL、Redis 和 Canvas 工程。
- API Key 只能保存在后端 Secret/配置层，不能写入前端节点或提交到 Git。
- 第一条真实闭环使用《寄生广告》SC01。

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

常用验证命令：

```bash
cd /Users/xiangyuqin/Downloads/infinite-canvas-short-drama/web
bun run test
bun run typecheck
bun run build
```

开发环境启动：

```bash
cd /Users/xiangyuqin/Downloads/infinite-canvas-short-drama
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

- Artifact / ArtifactVersion
- GenerationAttempt / ProviderJob
- QCReport
- ChangeRequest
- AgentProfile / Topic

这些属于后续阶段，不得提前宣称完成。

## 6. 下一步：Phase 3 UI Shell

不要直接进入 Provider Gateway。最新版详细规划明确规定：

```text
Phase 3 = UI Shell
Phase 4 = Film Nodes
Phase 5 = Provider Gateway v2
```

Phase 1.5 已通过自动验收，下一会话不要重做模型。Phase 0 仍需另补数据库 schema 快照和恢复说明。

### Phase 3：UI Shell

下一阶段实现：

- 完整 Left Project Navigator
- Top Project Production Status
- 可扩展 Right Inspector Header/Tabs
- Bottom Shot Strip（不是现有 Timeline 编辑弹窗的简单替代）
- 1920×1080、1440×900、1280×720 三档布局验收

已有 UI 只能复用后增量升级：

- `web/src/components/canvas/canvas-project-sidebar.tsx`
- `web/src/pages/canvas/canvas-project-top-bar.tsx`
- `web/src/film/inspector/film-inspector.tsx`

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
5. 修改后运行测试、typecheck、build。
6. UI 功能必须做真实浏览器交互验证；静态测试不能替代点击、拖拽和刷新验证。
7. 对验收结果明确区分：代码存在、自动测试通过、浏览器交互通过、真实 Provider/媒体质量通过。

## 9. 给新 Codex 会话的第一条指令

```text
请先完整读取 CONTINUE-HERE.md 和
/Users/xiangyuqin/Downloads/短剧AI无限画布_Production_Canvas_v2_超详细发展规划.md。
检查当前分支、Git 状态和最近提交，不要重做 Phase 1/Phase 2。
Phase 1.5 已完成并通过 76/76 自动测试，不要重做。
下一步只进入 Phase 3 UI Shell，先输出本轮文件范围，并如实区分自动验证与浏览器验证。
```
