'use client';

import { useEffect, useRef, useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { PanelRightOpen, PanelRightClose } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import { ChatSidebar } from './chat-sidebar';
import { ChatInterface } from './chat-interface';
import { ChatSidePanel } from './chat-side-panel';
import { NotesPanel } from './notes-panel';
import { showToast } from './toast';
import { KnowledgePanel } from './knowledge-panel';
import { SettingsPanel } from './settings-panel';
import { usePanelStore } from '@/presentation/stores/panelStore';
import { useViewStore } from '@/presentation/stores/viewStore';
import { useConversations } from '@/presentation/hooks/useConversations';
import { useSystemThemeSync } from '@/presentation/hooks/useSystemThemeSync';

export function ChatLayout() {
  // 全局跟 OS 主题同步 — 顶层挂,settings 打开/关闭都不影响监听。
  useSystemThemeSync();
  const chat = useConversations();

  const sidePanelOpen = usePanelStore((s) => s.open);
  const toggleSidePanel = usePanelStore((s) => s.toggle);
  const setSidePanelOpen = usePanelStore((s) => s.setOpen);
  const panelWidth = usePanelStore((s) => s.width);

  const view = useViewStore((s) => s.view);
  const setView = useViewStore((s) => s.setView);
  const pathname = usePathname();
  const router = useRouter();
  const isNotes = view === 'notes';
  const isKnowledge = view === 'knowledge';
  const isSettings = view === 'settings';

  // pathname → view 同步 — 单一视图真相源;刷新 / 直接打开 /notes 都能落到正确面板。
  useEffect(() => {
    const v = pathnameToView(pathname);
    if (v && v !== view) setView(v);
  }, [pathname, view, setView]);

  // 笔记编辑器请求的 chat one-shot 意图;切到 chat 时消费并清空。
  const pendingChatIntent = useViewStore((s) => s.pendingChatIntent);
  const setPendingChatIntent = useViewStore((s) => s.setPendingChatIntent);
  // chat 对象每渲染是新字面量;把 newConversation 用 ref 跨渲染捕获,effect deps 干净。
  // 同步挪到 effect 里 — 之前 render 期间 ref 赋值会被 react-hooks/refs 抓到。
  const newConversationRef = useRef(chat.newConversation);
  useEffect(() => {
    newConversationRef.current = chat.newConversation;
  }, [chat.newConversation]);

  // 关闭后滞后 200ms 卸载,让宽度过渡播完再清内部 DOM。
  // 打开路径用「render 期间条件 setState」同步派生(React 官方模式),避开 effect 同步 setState 警告;
  // 关闭路径用 setTimeout 滞后,回调里的 setState 跑在 effect body 之外。
  const [shouldRenderPanel, setShouldRenderPanel] = useState(sidePanelOpen);
  if (sidePanelOpen && !shouldRenderPanel) {
    setShouldRenderPanel(true);
  }
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (sidePanelOpen) {
      if (closeTimerRef.current) {
        clearTimeout(closeTimerRef.current);
        closeTimerRef.current = null;
      }
      return;
    }
    if (!shouldRenderPanel) return;
    closeTimerRef.current = setTimeout(() => {
      setShouldRenderPanel(false);
      closeTimerRef.current = null;
    }, 200);
    return () => {
      if (closeTimerRef.current) {
        clearTimeout(closeTimerRef.current);
        closeTimerRef.current = null;
      }
    };
  }, [sidePanelOpen, shouldRenderPanel]);

  // 应用笔记编辑器发来的「去对话」意图:起草稿会话 + 预览面板显示 + 自动 @ 当前笔记。
  useEffect(() => {
    if (view !== 'chat' || pendingChatIntent === null) return;
    if (pendingChatIntent.kind !== 'new-conversation') return;
    const doc = pendingChatIntent.doc;
    newConversationRef.current();
    const panel = usePanelStore.getState();
    panel.setOpen(true);
    panel.setMode('file');
    panel.setPreviewDocId(doc.id);
    panel.addAttachedDoc(doc);
    setPendingChatIntent(null);
  }, [view, pendingChatIntent, setPendingChatIntent]);

  // 兜底:用户在 effect 跑前手动切走,清掉 intent,避免再切回 chat 时误重放。
  useEffect(() => {
    if (view !== 'chat' && pendingChatIntent !== null) {
      setPendingChatIntent(null);
    }
  }, [view, pendingChatIntent, setPendingChatIntent]);

  const previewButton = (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant={sidePanelOpen ? 'secondary' : 'ghost'}
            size="icon"
            className="h-8 w-8"
            onClick={toggleSidePanel}
            aria-label={sidePanelOpen ? '关闭预览面板' : '打开预览面板'}
          >
            {sidePanelOpen ? (
              <PanelRightClose className="h-4 w-4" />
            ) : (
              <PanelRightOpen className="h-4 w-4" />
            )}
          </Button>
        }
      />
      <TooltipContent>
        {sidePanelOpen ? '关闭预览面板' : '打开预览面板'}
      </TooltipContent>
    </Tooltip>
  );

  return (
    <div className="h-screen flex bg-background overflow-hidden relative">
      <ChatSidebar
        conversations={chat.sidebarConversations}
        activeId={chat.sidebarActiveId}
        onSelect={chat.selectConversation}
        onNew={chat.newConversation}
        projects={chat.projects}
        focusedProjectId={chat.focusedProjectId}
        // URL 是 single source of truth:切项目推到 /?project=:id,hook 内的 effect 反向同步 focus+activeId。
        onSelectProject={(id) => {
          if (id) router.push(`/?project=${encodeURIComponent(id)}`);
          else router.push('/');
        }}
        onCreateProject={() => void chat.createProject('未命名')}
        onRenameProject={(id, title) => chat.renameProject(id, title)}
        onArchiveProject={(id) => void chat.archiveProject(id)}
      />

      <main className="flex-1 flex flex-col min-w-0">
        {isKnowledge ? (
          <KnowledgePanel />
        ) : isNotes ? (
          <NotesPanel />
        ) : isSettings ? (
          <SettingsPanel />
        ) : (
          <ChatInterface
            title={chat.active.title}
            messages={chat.active.messages}
            pending={chat.pending}
            onSend={chat.sendMessage}
            onStop={chat.stop}
            onEditMessage={chat.editMessage}
            onDeleteConversation={() => {
              // deleteActive 失败会 throw,接住给个 toast,不让失败静默。
              void Promise.resolve(chat.deleteActive()).catch((e: unknown) => {
                showToast(
                  e instanceof Error ? `归档失败: ${e.message}` : '归档失败',
                  'error'
                );
              });
            }}
            currentProject={chat.focusedOrActiveProject}
            projects={chat.projects}
            onPickProject={(id) => void chat.pickProject(id)}
            // 只有草稿 + 聚焦项目的情况可清;真实对话绑了 active project 时不传回调,chip 也不显示 X。
            onClearProject={
              !chat.activeProject && chat.focusedProjectId
                ? () => router.push('/')
                : undefined
            }
            pendingQuestion={chat.pendingQuestion}
            onAnswerQuestion={chat.answerQuestion}
          />
        )}
      </main>

      {!isKnowledge && !isNotes && !isSettings && (
        <div
          className="shrink-0 overflow-hidden transition-[width] duration-200 ease-out"
          style={{ width: sidePanelOpen ? panelWidth : 0 }}
        >
          {shouldRenderPanel && (
            <ChatSidePanel
              onClose={() => setSidePanelOpen(false)}
              conversationId={chat.sidebarActiveId}
              activeProjectPreviewUrl={(chat.focusedOrActiveProject as { previewUrl?: string } | null)?.previewUrl}
            />
          )}
        </div>
      )}

      {!isNotes && !isKnowledge && !isSettings && (
        <div className="absolute top-3 right-3 z-20">{previewButton}</div>
      )}
    </div>
  );
}

// pathname → view 映射;/c/:id 也归 chat(对话视图)。
function pathnameToView(p: string): 'chat' | 'notes' | 'knowledge' | 'settings' | null {
  if (p.startsWith('/notes')) return 'notes';
  if (p.startsWith('/knowledge')) return 'knowledge';
  if (p.startsWith('/settings')) return 'settings';
  if (p === '/' || p.startsWith('/c/') || p.startsWith('/admin')) return 'chat';
  return null;
}
