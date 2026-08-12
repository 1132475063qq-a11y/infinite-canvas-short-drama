# 新账号续接说明：AI 短剧生产无限画布

更新时间：2026-08-12

## 1. 当前项目位置

本地正式仓库：

```text
/Users/xiangyuqin/Downloads/infinite-canvas-short-drama
```

GitHub 仓库：

```text
https://github.com/1132475063qq-a11y/infinite-canvas-short-drama
```

当前开发分支：

```text
codex/phase3-production-ui-shell
```

Phase 3 提交：

```text
23e41b5 feat(canvas): complete phase 3 production ui shell
```

## 2. 换 Codex 账号时

只更换 Codex / ChatGPT 账号，不会修改本地 Git 仓库、GitHub 登录状态、分支或 Docker 数据。

新账号进入 Codex 后：

1. 打开本地目录 `/Users/xiangyuqin/Downloads/infinite-canvas-short-drama`。
2. 把下面“新会话第一条指令”发给新会话。
3. 不要重新克隆项目，不要重新初始化 Git，不要删除 Docker volume。
4. 新会话必须先检查当前分支、Git 状态和交接文档，再开始 Phase 4。

## 3. 新会话第一条指令

```text
请接手本地项目：
/Users/xiangyuqin/Downloads/infinite-canvas-short-drama

先完整读取：
1. /Users/xiangyuqin/Downloads/infinite-canvas-short-drama/NEW-ACCOUNT-HANDOFF.md
2. /Users/xiangyuqin/Downloads/infinite-canvas-short-drama/CONTINUE-HERE.md
3. /Users/xiangyuqin/Downloads/短剧AI无限画布_Production_Canvas_v2_超详细发展规划.md
4. 仓库内 AGENTS.md

然后执行 git status -sb、git branch --show-current、git log -8 --oneline，确认当前分支为 codex/phase3-production-ui-shell，且 Phase 3 提交 23e41b5 已存在。

不要重做 Phase 1、Phase 1.5、Phase 2、Phase 3。下一步只进入 Phase 4 Film Nodes：先审计 Scene、Shot、Character、Location、Prop 的后端事实源、字段缺口、API 和 Inspector 接入点，再输出修改文件范围；确认边界后直接实现、测试、构建并做真实浏览器验收。不要提前进入 Provider Gateway 或 Agent Runtime。
```

## 4. 当前已经完成

- Phase 1：Film Semantic Layer 基础。
- Phase 1.5：26 种 FilmNodeKind、Typed Port、CanvasDocumentV2 契约冻结。
- Phase 2：Grid、Snap、Alignment、Distribution、Scene Lane、Collision 核心能力。
- Phase 3：完整 Project Navigator、顶部生产状态、Inspector Shell、Bottom Shot Strip。
- Phase 3 自动测试：78/78 通过。
- TypeScript typecheck 和 production build 通过。
- 1920×1080、1440×900、1280×720 三档真实浏览器验收通过。

详细状态和未完成项以 `CONTINUE-HERE.md` 为准。

## 5. 下一步唯一范围

```text
Phase 4 Film Nodes
```

本阶段只做：

- Scene 节点升级。
- Shot Contract 节点升级。
- Character / Location / Prop 节点升级。
- 对应 Inspector 真实字段、版本与领域引用接入。
- 后端 Project / Scene / Shot / Asset / AssetVersion 是事实源；Canvas 仅做 Projection。

本阶段禁止：

- Provider Gateway。
- 图片或视频模型接入。
- Agent Runtime。
- 多人协作。
- 把规划文档中的未来模型误报为已经实现。

## 6. 本地运行说明

标准开发启动方式：

```bash
cd /Users/xiangyuqin/Downloads/infinite-canvas-short-drama
LOCAL_UID=$(id -u) LOCAL_GID=$(id -g) docker compose -f docker-compose.dev.yml up --build
```

标准地址：

```text
http://localhost:3000
```

注意：当前机器上 `/Users/xiangyuqin/Desktop/Infinite-Canvas-main` 曾占用 `3000` 端口。Phase 3 验收时，本仓库前端临时使用 `http://localhost:3001`。如果看到旧版侧栏或旧 Inspector，先检查端口对应的进程和工作目录，不要误判代码没有更新。

## 7. 如果还要更换 GitHub 账号

换 Codex 账号不要求换 GitHub 账号。只有明确要把 GitHub 登录也切换时才执行：

```bash
gh auth status
gh auth logout -h github.com
gh auth login -h github.com -p https -w
gh auth setup-git
gh auth status
```

如果新 GitHub 账号没有原仓库写权限，可选择：

1. 继续使用原 GitHub 账号推送；或
2. 给新账号添加仓库协作者权限；或
3. Fork 后把新仓库设置为新的 remote。

不要把 GitHub Token、API Key、Cookie 或登录码写入本文件、项目文件或 Git 提交。

## 8. 恢复检查

新会话开始时运行：

```bash
cd /Users/xiangyuqin/Downloads/infinite-canvas-short-drama
git status -sb
git branch --show-current
git log -8 --oneline
git remote -v
git rev-list --left-right --count HEAD...@{upstream}
```

正确状态应包括：

```text
codex/phase3-production-ui-shell
23e41b5 feat(canvas): complete phase 3 production ui shell
0 0
```

最后一行 `0 0` 表示本地与 GitHub 对应分支完全同步。
