// (app) 路由分组下的 /c/:id;AuthGuard + ChatLayout 由 (app)/layout.tsx 提供。
// URL 参数在 ChatLayout 客户端用 usePathname + URL 派生值读取,这里不需 await。
export default function ConversationPage() {
  return null;
}