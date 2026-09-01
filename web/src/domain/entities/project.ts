// 用户拥有的工作项目;name 是文件系统 slug(后端生成),title 是用户可编辑显示名。
export interface Project {
  id: string;
  name: string;
  title: string;
  // 容器内 cwd,iframe src 拼接用,不展示给用户。
  cwd: string;
  // ready / archived 等。
  status: string;
  updatedAt: string;
}

// 当前 active project 精简视图;null 表示还没建 / active 被清了。
export interface ActiveProject {
  id: string;
  name: string;
  title: string;
  cwd: string;
  // Sandbox-stamped CDN URL; "" until controller stamps it.
  previewUrl?: string;
  updatedAt: string;
}