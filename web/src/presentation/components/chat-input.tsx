'use client';

import { Send, Square, AtSign, NotebookPen, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import type { Project, ActiveProject } from '@/domain/entities/project';
import type { Document } from '@/domain/entities/document';
import { ProjectPicker } from './project-picker';
import { cn } from '@/lib/utils';

interface ChatInputProps {
  value: string;
  onChange: (v: string) => void;
  onSend: () => void;
  onStop: () => void;
  pending: boolean;
  // 当前 chip 应展示的项目:undefined/null → chip 不渲染。
  currentProject?: Project | ActiveProject | null;
  // 可选的所有项目,给 picker 下拉用;没传就不渲染 picker。
  projects?: Project[];
  onPickProject?: (id: string) => void;
  // 草稿+聚焦项目时才传;点了清掉 focused,chip 消失。
  onClearProject?: () => void;
  // 已选文档列表,跟在 @ 按钮右边展示多个 chip;空数组不显示。
  selectedDocuments?: Document[];
  // @ 按钮的 ref — 点击切换 picker。
  atButtonRef?: React.RefObject<HTMLButtonElement | null>;
  // 外层包裹 div 的 ref — DocumentPicker 用它定位到输入框上沿。
  containerRef?: React.RefObject<HTMLDivElement | null>;
  // 点击 @ 按钮 → 切换 DocumentPicker 开/关。
  onTogglePicker: () => void;
  // chip 上的 X → 清掉单个文档(不影响预览面板)。
  onRemoveDocument: (id: number) => void;
}

export function ChatInput({
  value,
  onChange,
  onSend,
  onStop,
  pending,
  currentProject,
  projects,
  onPickProject,
  onClearProject,
  selectedDocuments,
  atButtonRef,
  containerRef,
  onTogglePicker,
  onRemoveDocument,
}: ChatInputProps) {
  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      onSend();
    }
  }

  return (
    <div className="px-4 py-4 bg-background">
      <div className="max-w-2xl mx-auto flex flex-col gap-1.5">
        {currentProject && (
          <ProjectPicker
            project={currentProject}
            projects={projects ?? []}
            onPick={onPickProject}
            onClear={onClearProject}
          />
        )}

        <div
          ref={containerRef}
          className="rounded-2xl border bg-background shadow-sm focus-within:ring-2 focus-within:ring-ring/30 focus-within:border-ring transition-shadow overflow-hidden"
        >
          <Textarea
            value={value}
            onChange={(e) => onChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="输入消息，回车发送（Shift+Enter 换行）"
            disabled={pending}
            rows={1}
            className="block w-full resize-none border-0 shadow-none focus-visible:ring-0 pl-3 pr-3 py-3 bg-transparent rounded-none"
          />

          <div className="flex items-center justify-between px-2 py-1.5">
            <div className="flex items-center gap-1.5 min-w-0 flex-1 flex-wrap">
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      ref={atButtonRef}
                      onClick={onTogglePicker}
                      size="icon"
                      variant="ghost"
                      className="h-8 w-8 shrink-0 rounded-xl"
                      aria-label="选择文档"
                      aria-haspopup="dialog"
                    >
                      <AtSign className="h-4 w-4" />
                    </Button>
                  }
                />
                <TooltipContent>选择文档(@)</TooltipContent>
              </Tooltip>
              {selectedDocuments?.map((doc) => (
                <SelectedDocumentChip
                  key={doc.id}
                  doc={doc}
                  onClear={() => onRemoveDocument(doc.id)}
                />
              ))}
            </div>

            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    onClick={pending ? onStop : onSend}
                    disabled={!pending && !value.trim()}
                    size="icon"
                    className="h-8 w-8 shrink-0 rounded-xl"
                    aria-label={pending ? '停止' : '发送'}
                  >
                    {pending ? (
                      <Square className="h-4 w-4" />
                    ) : (
                      <Send className="h-4 w-4" />
                    )}
                  </Button>
                }
              />
              <TooltipContent>{pending ? '停止' : '发送'}</TooltipContent>
            </Tooltip>
          </div>
        </div>
      </div>
    </div>
  );
}

// 已选文档 chip — 图标(按 source 切色)+ title + 清除。
function SelectedDocumentChip({
  doc,
  onClear,
}: {
  doc: Document;
  onClear: () => void;
}) {
  const isKnowledge = doc.source === 'knowledge';
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 h-7 pl-1.5 pr-1 rounded-md text-xs min-w-0 max-w-[14rem] shrink-0',
        isKnowledge
          ? 'bg-purple-500/10 text-purple-700 dark:text-purple-300'
          : 'bg-blue-500/10 text-blue-700 dark:text-blue-300'
      )}
    >
      <NotebookPen className="h-3 w-3 shrink-0" />
      <span className="truncate" title={doc.title}>{doc.title}</span>
      <button
        type="button"
        onClick={onClear}
        aria-label="取消文档引用"
        className="ml-0.5 inline-flex items-center justify-center h-4 w-4 rounded hover:bg-black/10 dark:hover:bg-white/10 shrink-0"
      >
        <X className="h-2.5 w-2.5" />
      </button>
    </span>
  );
}