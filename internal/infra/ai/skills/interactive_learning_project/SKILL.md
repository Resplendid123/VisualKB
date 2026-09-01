---
name: interactive_learning_project
description: 当用户给出一个知识主题(物理/算法/经济/任何概念),希望把它"拆解 + 做成可视化"时使用本 skill。流程是:从第一性原理拆出最小公理与衍生结论 → 设计一个 Vue3 + TypeScript 交互,让用户能拖动参数/改变条件直观看到结果。Agent 应先读取用户已有的相关笔记/知识库做事实校准,再创建项目、生成 Vue 代码、跑 build 验证。
---

# Interactive Learning Project — Vue3 可视化学习项目

把一个抽象概念变成"动手就能看见"的网页。用户能拖滑块、改输入、点按钮,实时看到 公式/曲线/状态变化。

## 何时触发

匹配用户的请求形如:

- "帮我做一个 … 的可视化"
- "我想理解 …,做个交互页面"
- "用 Vue 演示一下 …"
- "把 … 做成能玩的东西"

不触发:用户只想口述/讨论概念 (无产物)、只想查 KB 已有笔记、做生产级应用 (那是真正的工程,不是学习项目)。

## 总流程

```
1. 澄清范围      → ask_user_tool(必要时)
2. 摄入上下文    → search_kb + list_documents + read_document
3. 第一性拆解    → 写一份"公理 → 不变量 → 衍生结论"清单
4. 设计交互      → 决定哪些参数可调,哪些量随它们变
5. 创建项目      → create_project(bind 当前对话)
6. 脚手架        → bash: 在 /workspace/project 下用 Vite 起 Vue3 + TS 模板
7. 写组件        → bash: 写 .vue / .ts,严格按 Composition API + <script setup lang="ts">
8. 验证          → bash: npm install && build && dev smoke
9. 总结          → 用 Markdown 给用户讲清"公理 → 衍生"对照表 + 使用指南
```

## 步骤详解

### 1. 澄清范围 (只在必要时)

如果用户主题本身模糊 (例:"做个人工智能"),先用 `ask_user_tool` 缩小到具体子主题 (机器学习?神经网络?CNN?梯度下降?)—
不要替用户瞎猜。

如果用户主题清晰且自带素材 ("我贴了 Transformer 论文,你帮我做 attention 可视化"), 直接进步骤 2。

### 2. 摄入用户上下文

在动手拆解之前,先去翻用户的知识库 — 用户可能已经有笔记/资料:

```
list_documents(source="knowledge")   # 知识库目录
list_documents(source="note")        # 个人笔记
search_kb(query="<主题关键词>")        # 语义检索
read_document(document_id=...)       # 精读匹配项
```

如果检索到高度相关的文档,把它当作"事实基准",以它为锚,只补足它没说清的部分。

### 3. 第一性原理拆解

每个学习项目都要先在对话里 (给用户看)输出一份拆解清单,再开始写代码。结构:


### 4. 设计交互

**原则**:

- **可玩性 > 美观**:滑块/按钮立刻响应,延迟 < 16ms。
- **可视化优于文字**:能用图 (SVG/Canvas/曲线)就别用大段公式。
- **公式可见但不抢戏**:关键公式放在角落或 hover tooltip,主体是动态图。

### 5. 创建项目

调 `create_project` 工具,把当前对话绑到一个新项目上,这一步要先于写代码:

- title: 简短、可读 (例:"单摆可视化"、"注意力机制演示")
- cwd: 一般填 `/workspace/project/<slug>`(slug 用 kebab-case 英文)

create_project 会同步把对话→项目的关联建好,后续 bash 写入的文件就落在该项目下。

### 6. 脚手架

```bash
# Vite + Vue + TS 官方模板
cd /workspace/project  # bash 已经在 cwd,不必 cd;这里只是示意
npm create vite@latest . -- --template vue-ts   # 注意 `.` 表示当前目录

npm install 
```

`npm create vite` 在已有非空目录可能拒绝 — 如果项目根目录已经有别的文件 (`README.md` 等),先 `ls` 看一眼,必要时用
`--force` 或先移走。

### 7. 写 Vue 代码

**文件结构 **:

```
src/
  main.ts
  App.vue                # 顶层布局
  components/
  views/
  styles/
```

**约定**:

- 每个 `.vue` 顶层 `<script setup lang="ts">`,不要 Options API。
- props/emits 用 `defineProps<...>()` / `defineEmits<...>()` 类型化。
- 共享状态放 composable,组件只负责展示 + 触发事件。
- 数值精度默认 3 位有效数 (`toPrecision(3)`),避免抖动。
- 颜色用 Tailwind class 或 CSS var,别硬编码 hex。

**SVG 优先**:大多数学习可视化 (SVG path / circle / line)用 SVG 就够, 无须引入 D3。d3 只在需要复杂比例尺/力导向时再加。

### 8. 验证

```bash
npm install              # 装依赖
npm build                # 必须能过 — 至少类型检查 + Vite 打包
npm dev --host 0.0.0.0   # 起 dev server,后台跑或短时启动验证无报错
```

`npm build` 是硬门槛:任何 TS 类型错或打包错都必须修。

- **build 产物必须落到 `/workspace/dist`**:sandbox 预览路由(`/workspace/dist` → CDN)只看这一个目录。Vite 在 `vite.config.ts` 设 `build.outDir: '/workspace/dist'`(默认 `dist/` 会被忽略);若构建后还在 `dist/`,再 `cp -r dist/* /workspace/dist/`。Next.js 同理(`next.config.js` 的 `distDir: '/workspace/dist'`,或 build 后 `cp -r .next/static/ /workspace/dist/_next/static/`)。

### 9. 总结

给用户回复时包含:

1. **拆解清单**(公理 → 衍生),让用户看到"我理解对了"。
2. **关键文件**(`App.vue` / composable 路径),告诉用户代码组织。
3. **怎么玩**:列 inputs 和它们影响的 observables。
4. **可选扩展**:留 1–2 个"如果想加 X,改 Y"的钩子,方便用户迭代。

## 反模式 (不要做)

- ❌ 用 `<template>` 里嵌大段 JS 表达式算坐标 — 抽到 computed。
- ❌ `any` 类型 — TS 严格模式,组件 props 必须类型化。
- ❌ 引入与可视化无关的重量级库 (UI framework / 状态管理) — 单页够用。
- ❌ 把 build 报错藏起来 (改 `// @ts-ignore`) — 修真的。
- ❌ 写完不验证就交付 — 至少 `pnpm build` 必须绿。

## 失败兜底

- 用户主题太宽 → 用 `ask_user_tool` 收窄,别瞎猜方向。
- KB 检索结果冲突 → 把冲突点列给用户,让他裁决,不要自作主张融合。
- Vite 创建失败 (目录非空) → 先 `ls` 看看,清理无关文件再试,或换子目录。

## 与其他工具协作

- `ask_user_tool`:只在主题真模糊时调,选项 ≤ 4 个。
- `write_memory`:项目交付后,如果你从用户主题里推断出"用户的口味"或
  "用户对某种知识、可视化的偏好",顺手 append 一下 — 但只在确实学到了东西时, 避免噪音。