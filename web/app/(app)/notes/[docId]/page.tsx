// (app) 路由分组下的 /notes/:docId;AuthGuard + ChatLayout 由 (app)/layout.tsx 提供。
// docId 由 NotesPanel 客户端 usePathname + URL 派生值读取,这里不需 await。
export default function NoteDetailPage({
  params,
}: {
  params: Promise<{ docId: string }>;
}) {
  void params;
  return null;
}