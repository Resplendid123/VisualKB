import { ChatLayout } from '@/presentation/components/chat-layout';
import AuthGuard from '@/presentation/components/auth-guard';

// 共享给 /、/c/:id、/notes、/notes/:id、/knowledge、/settings 的布局 — 跨这些页面的导航不重建 <ChatLayout>,本地状态不丢。路由分组 (app) 不会出现在 URL 里。
export default function AppLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AuthGuard>
      <ChatLayout />
    </AuthGuard>
  );
}