'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, BookOpen, Settings, NotebookPen, type LucideIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';
import type { Conversation } from '@/domain/entities/conversation';
import type { Project } from '@/domain/entities/project';
import { useViewStore } from '@/presentation/stores/viewStore';
import { EditProjectDialog } from './edit-project-dialog';
import { ProjectSection, ConversationsSection } from './sidebar-sections';

interface ChatSidebarProps {
  conversations: Conversation[];
  activeId: string;
  onSelect: (id: string) => void;
  onNew: () => void;
  projects: Project[];
  // 当前聚焦的项目 id;null 表示未聚焦;"新建会话"会绑到该项目。
  focusedProjectId: string | null;
  onSelectProject: (id: string | null) => void;
  onCreateProject: () => void;
  onRenameProject: (id: string, title: string) => Promise<unknown> | void;
  onArchiveProject: (id: string) => Promise<unknown> | void;
}

interface NavItem {
  icon: LucideIcon;
  label: string;
  onClick?: () => void;
}

export function ChatSidebar({
  conversations,
  activeId,
  onSelect,
  onNew,
  projects,
  focusedProjectId,
  onSelectProject,
  onCreateProject,
  onRenameProject,
  onArchiveProject,
}: ChatSidebarProps) {
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const view = useViewStore((s) => s.view);
  const router = useRouter();

  // 任何要回到 chat 上下文的操作,先切回 chat 视图(notes 视图下 main 是笔记页)。
  const goChat = () => router.push('/');
  const goNotes = () => router.push('/notes');
  const goKnowledge = () => router.push('/knowledge');

  const navItems: NavItem[] = [
    // onNew 内部已 router.push('/'),不用 goChat 双 push。
    { icon: Plus, label: '新建会话', onClick: () => onNew() },
    {
      icon: NotebookPen,
      label: '笔记',
      onClick: () => (view === 'notes' ? goChat() : goNotes()),
    },
    {
      icon: BookOpen,
      label: '知识库',
      onClick: () => (view === 'knowledge' ? goChat() : goKnowledge()),
    },
  ];

  return (
    <aside className="w-60 shrink-0 border-r flex flex-col h-full bg-muted/20">
      <div className="px-4 py-3 flex items-center gap-2">
        <div className="h-6 w-6 rounded-md bg-primary text-primary-foreground grid place-items-center text-[10px] font-bold">
          KB
        </div>
        <span className="font-semibold text-sm">KB</span>
      </div>

      <nav className="p-2 space-y-0.5">
        {navItems.map((item) => {
          const Icon = item.icon;
          // "新建会话"是动作按钮,不应该有持久的 active 高亮。
          const active =
            (item.label === '笔记' && view === 'notes') ||
            (item.label === '知识库' && view === 'knowledge');
          return (
            <button
              key={item.label}
              onClick={item.onClick}
              className={cn(
                'w-full text-left px-2 py-1.5 rounded-md text-sm flex items-center gap-2 transition-colors',
                active
                  ? 'bg-primary/10 text-foreground'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              )}
            >
              <Icon className="h-4 w-4" />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>

      <Separator />

      <div className="flex-1 overflow-y-auto py-2 min-h-0">
        <ProjectSection
          projects={projects}
          focusedProjectId={focusedProjectId}
          onSelectProject={(id) => {
            // 单次 push:由上层 router.push('/?project=:id') 完成导航;不双 push 避免竞态。
            onSelectProject(id);
          }}
          onCreateProject={() => {
            // createProject 自己 router.push('/?project=:id'),这里只切视图。
            onCreateProject();
          }}
          onArchiveProject={onArchiveProject}
          onStartEditProject={setEditingProject}
          conversations={conversations}
          activeId={activeId}
          view={view}
          onSelectConversation={(id) => {
            // chat.selectConversation 已 router.push('/c/:id'),无需双 push。
            onSelect(id);
          }}
          onNewConversation={() => {
            // chat.newConversation 已 router.push('/'),无需双 push。
            onNew();
          }}
        />

        <ConversationsSection
          conversations={conversations}
          activeId={activeId}
          view={view}
          onSelect={(id) => {
            // chat.selectConversation 已 router.push('/c/:id'),无需双 push。
            onSelect(id);
          }}
        />
      </div>

      <Separator />

      <div className="p-2 flex flex-col gap-0.5">
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            'justify-start gap-2',
            view === 'settings' && 'bg-primary/10 text-foreground'
          )}
          onClick={() => router.push('/settings')}
        >
          <Settings className="h-4 w-4" />
          设置
        </Button>
      </div>

      <EditProjectDialog
        project={editingProject}
        onClose={() => setEditingProject(null)}
        onSave={onRenameProject}
      />
    </aside>
  );
}