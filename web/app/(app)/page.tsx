// (app) 路由分组下的首页;AuthGuard + ChatLayout 由 (app)/layout.tsx 提供。
// 留这个文件只是为了让 / 命中,实际渲染内容为空,layout 主导一切。
export default function HomePage() {
  return null;
}