# UI 设计系统规范

本规范以 `web/src/styles/globals.css` 的真实 Token 为唯一来源。新增 UI 必须先复用现有 Token；需要新 Token 时按 Primitive → Semantic → Component 的顺序补齐，并同时检查亮色、暗色和 `prefers-reduced-motion`。

## 设计原则

- 工作台优先：高频创作操作紧凑、可扫描，避免用大段说明文字或嵌套卡片占据工作区。
- 两套主题同等可读：不直接硬编码浅色背景、深色文字、阴影或边框。
- 焦点可访问：指针点击不保留无意义焦点框，键盘访问保留 `:focus-visible`。
- 图标使用已有 `lucide-react` 或 Ant Design 图标；重要入口应同时有图标和中文文字。
- 画布、节点、Dock、Panel 和 Modal 必须使用各自 Component Token，不在页面里手写层级值或浮层尺寸。

## Token 分层

| 层级 | 作用 | 当前示例 |
| --- | --- | --- |
| Primitive | 与主题无关的基础值 | `--space-*`、`--fs-*`、`--r-*`、`--motion-*`、`--z-*`、`--palette-*` |
| Semantic | 随主题变化的颜色与表面 | `--bg`、`--fg`、`--border-semantic`、`--surface-card`、`--card-surface` |
| Component | 指定组件的尺寸、边框、投影和层级 | `--node-*`、`--dock-*`、`--panel-*`、`--modal-*` |

新增值的顺序必须是：先新增 Primitive，再让 Semantic 引用它，最后让 Component 引用 Semantic。组件中禁止用裸 Tailwind 任意值或孤立的颜色、圆角、阴影、间距与 `z-index`。

## 卡片与列表

- 页面区块不是卡片；只给重复内容项、Modal 或真正需要边界的工具表面使用卡片。
- 默认表面使用 `var(--surface-card)`，悬停使用 `var(--surface-card-hover)`；需要更明确的库卡片时使用 `--card-surface`、`--card-border`、`--card-elevation` 和其 hover 版本。
- 圆角按现有组件选择 `--r-xl` 或 `--r-2xl`；不要为局部卡片再定义新的圆角数值。
- 新卡片的 hover 使用项目基线：`transform 240ms var(--motion-ease-out)` 与最多 `translateY(-4px)`。现存历史组件有 `-3px` 或 `-6px` 的例外；改造它们时先合并到本规范，而不是追加覆盖样式。
- 除非内容语义明确，避免“卡片里再套卡片”。列表、表格和工具栏使用非浮动分区和分隔线表达层次。

## 画布与浮层

- 画布浮层使用 `--z-canvas` 到 `--z-loader` 的既有层级，不自行写 `z-[N]`。
- Dock 引用 `--dock-*`，节点引用 `--node-*`，右侧/浮动面板引用 `--panel-*`，Modal 引用 `--modal-*`。
- 画布事件必须考虑 `data-canvas-no-zoom` 与 `data-canvas-wheel-scroll`，Popovers / Dropdowns / Modal 不能被画布滚轮或拖拽劫持。
- Modal 的 Ant Design 内容外壳是 `.ant-modal-container`；优先使用组件 `styles` 和范围受限的 class，而不是全局 `.ant-modal-*` 覆盖。

## 动效与状态

- 动效时长和缓动使用 `--motion-dur-*-calc` 与 `--motion-ease-*`，以便自动服从 `--motion-scale`。
- 所有新增非必要动效必须在 `prefers-reduced-motion` 和 `.no-motion` 下自然降级；不添加持续干扰创作的装饰动画。
- 状态色使用语义状态 Token / 组件状态样式，不能单凭颜色传达失败、进行中、成功或阻塞。

## 实现检查

1. 先在 `globals.css` 找可复用 Token，再写组件样式。
2. 检查明暗主题、键盘焦点、窄屏文本换行和浮层滚动边界。
3. 第三方组件先核对当前依赖版本与实际 DOM，再添加受限覆盖。
4. 新增或改造通用组件时，更新本规范或其对应 Component Token；不要把规则留在单页末尾覆盖中。
