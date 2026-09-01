'use client';

import { useState } from 'react';
import {
  Plus,
  MessageSquare,
  Folder,
  FolderOpen,
  ChevronRight,
  MoreHorizontal,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import type { Conversation } from '@/domain/entities/conversation';
import type { Project } from '@/domain/entities/project';

// 项目区:每项可展开,内嵌属于它的对话。
export function ProjectSection({
  projects,
  focusedProjectId,
  onSelectProject,
  onCreateProject,
  onArchiveProject,
  onStartEditProject,
  conversations,
  activeId,
  onSelectConversation,
  onNewConversation,
  view,
}: {
  projects: Project[];
  focusedProjectId: string | null;
  onSelectProject: (id: string | null) => void;
  onCreateProject: () => void;
  onArchiveProject: (id: string) => Promise<unknown> | void;
  onStartEditProject: (p: Project) => void;
  conversations: Conversation[];
  activeId: string;
  onSelectConversation: (id: string) => void;
  onNewConversation: () => void;
  // 非 chat 视图下高亮要清掉,否则离开对话后侧栏还在假装"这条还是当前对话"。
  view: 'chat' | 'notes' | 'knowledge' | 'settings';
}) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const inChatView = view === 'chat';

  function toggleCollapse(projectId: string) {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(projectId)) next.delete(projectId);
      else next.add(projectId);
      return next;
    });
  }

  // URL 是 single source of truth:推 /?project=:id 后 hook effect 把 activeId 设回 DRAFT_ID,
  // 不需要先 onNewConversation 再 onSelectProject — 双 push 竞态且多此一举。
  function selectProject(projectId: string) {
    onSelectProject(projectId);
  }

  return (
    <div className="mb-2">
      <div className="text-xs font-medium text-muted-foreground px-4 py-1 flex items-center justify-between">
        <span>项目</span>
        <button
          onClick={onCreateProject}
          aria-label="新建项目"
          className="h-5 w-5 grid place-items-center rounded hover:bg-muted hover:text-foreground transition-colors"
        >
          <Plus className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="px-2 space-y-0.5">
        {projects.length === 0 && (
          <div className="px-2 py-1.5 text-xs text-muted-foreground/70">
            暂无项目
          </div>
        )}
        {projects.map((p) => {
          const isFocused = inChatView && p.id === focusedProjectId;
          const isCollapsed = collapsed.has(p.id);
          const childConvos = conversations.filter((c) => c.activeProjectId === p.id);
          return (
            <div key={p.id}>
              <div
                className={cn(
                  'group w-full px-2 py-1.5 rounded-md text-sm flex items-center gap-1.5 transition-colors cursor-pointer',
                  isFocused
                    ? 'bg-primary/10 text-foreground'
                    : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                )}
                onClick={() => selectProject(p.id)}
              >
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    toggleCollapse(p.id);
                  }}
                  aria-label={isCollapsed ? '展开' : '折叠'}
                  className="h-4 w-4 grid place-items-center rounded hover:bg-muted-foreground/20 transition-colors"
                >
                  <ChevronRight
                    className={cn(
                      'h-3.5 w-3.5 transition-transform',
                      !isCollapsed && 'rotate-90'
                    )}
                  />
                </button>
                {isCollapsed ? (
                  <Folder className="h-4 w-4 shrink-0" />
                ) : (
                  <FolderOpen className="h-4 w-4 shrink-0" />
                )}
                <span className="flex-1 truncate" title={p.title}>
                  {p.title}
                </span>

                {/* hover 时浮现:more 菜单(编辑 / 归档) + 新建会话 */}
                <div className="ml-auto flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          aria-label="项目操作"
                          onClick={(e: React.MouseEvent) => e.stopPropagation()}
                        >
                          <MoreHorizontal className="h-3.5 w-3.5" />
                        </Button>
                      }
                    />
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => onStartEditProject(p)}>
                        编辑
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        variant="destructive"
                        onClick={() => {
                          if (confirm(`确定归档项目「${p.title}」？`)) {
                            void onArchiveProject(p.id);
                          }
                        }}
                      >
                        归档项目
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    aria-label="新建会话"
                    onClick={(e) => {
                      e.stopPropagation();
                      selectProject(p.id);
                    }}
                  >
                    <Plus className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>

              {!isCollapsed && childConvos.length > 0 && (
                <div className="ml-7 mt-0.5 space-y-0.5">
                  {childConvos.map((c) => (
                    <button
                      key={c.id}
                      onClick={() => onSelectConversation(c.id)}
                      className={cn(
                        'w-full text-left px-2 py-1 rounded-md text-sm flex items-center gap-2 transition-colors',
                        inChatView && c.id === activeId
                          ? 'bg-primary/10 text-foreground'
                          : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                      )}
                    >
                      <MessageSquare className="h-3.5 w-3.5 shrink-0" />
                      <span className="flex-1 truncate">{c.title}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// 未归类对话;为空也露出"对话"标题,跟项目区视觉对齐。
export function ConversationsSection({
  conversations,
  activeId,
  onSelect,
  view,
}: {
  conversations: Conversation[];
  activeId: string;
  onSelect: (id: string) => void;
  // 仅 chat 视图下高亮当前对话;跳到笔记/知识库时整栏降为 hover 样式。
  view: 'chat' | 'notes' | 'knowledge' | 'settings';
}) {
  const inChatView = view === 'chat';
  const ungrouped = conversations.filter((c) => !c.activeProjectId);
  return (
    <div>
      <div className="text-xs font-medium text-muted-foreground px-4 py-1">
        对话
      </div>
      <div className="px-2 space-y-0.5">
        {ungrouped.map((c) => (
          <button
            key={c.id}
            onClick={() => onSelect(c.id)}
            className={cn(
              'w-full text-left px-2 py-1.5 rounded-md text-sm flex items-center gap-2 transition-colors',
              inChatView && c.id === activeId
                ? 'bg-primary/10 text-foreground'
                : 'hover:bg-muted text-muted-foreground hover:text-foreground'
            )}
          >
            <MessageSquare className="h-4 w-4 shrink-0" />
            <span className="flex-1 truncate">{c.title}</span>
          </button>
        ))}
      </div>
    </div>
  );
}