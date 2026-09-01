'use client';

import { useMemo, useState } from 'react';
import {
  NotebookPen,
  Search,
  X,
  type LucideIcon,
} from 'lucide-react';
import { Input } from '@/components/ui/input';
import {
  DropdownMenu,
  DropdownMenuContent,
} from '@/components/ui/dropdown-menu';
import { documentUseCases } from '@/application/chatContainer';
import type { Document, DocumentSource } from '@/domain/entities/document';
import { cn } from '@/lib/utils';
import { useAsyncState } from '@/presentation/hooks/useAsyncState';

const SOURCE_BADGES: Record<DocumentSource, { label: string; icon: LucideIcon; class: string }> = {
  note: {
    label: '笔记',
    icon: NotebookPen,
    class: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
  },
  knowledge: {
    label: '知识',
    icon: NotebookPen,
    class: 'bg-purple-500/10 text-purple-600 dark:text-purple-400',
  },
};

interface DocumentPickerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  // 弹层定位锚点 — 输入框外层 wrapper,base-ui 据此把弹层贴在上沿。
  anchorRef: React.RefObject<HTMLDivElement | null>;
  onPick: (doc: Document) => void;
}

// 文档选择器 — 挂在 @ 按钮上;每次 open 重拉,选完通知父组件,自身不直接改父组件的 open 状态。
export function DocumentPicker({
  open,
  onOpenChange,
  anchorRef,
  onPick,
}: DocumentPickerProps) {
  const [query, setQuery] = useState('');
  // 每次 open=true 都重拉 — 新建/超 60 的 doc 必须能选到,缓存会脏。
  const { data, loading, error } = useAsyncState(
    () => documentUseCases.list.execute({ limit: 60, offset: 0 }),
    [open]
  );
  const items = data?.items ?? [];

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return items;
    return items.filter((d) => d.title.toLowerCase().includes(q));
  }, [items, query]);

  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange}>
      <DropdownMenuContent
        anchor={anchorRef}
        align="start"
        side="top"
        sideOffset={0}
        className="w-80 p-0"
      >
        <div className="flex items-center gap-2 px-2.5 py-2 border-b">
          <Search className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索文档…"
            className="h-7 px-1 text-xs bg-transparent border-transparent focus-visible:bg-background"
          />
          {query && (
            <button
              type="button"
              onClick={() => setQuery('')}
              className="text-muted-foreground hover:text-foreground"
              aria-label="清空搜索"
            >
              <X className="h-3 w-3" />
            </button>
          )}
        </div>

        <div className="max-h-80 overflow-y-auto py-1">
          {error && (
            <div className="mx-2.5 my-2 px-3 py-2 rounded-md bg-destructive/10 text-xs text-destructive">
              {error}
            </div>
          )}

          {loading && items.length === 0 && (
            <div className="px-3 py-6 text-center text-xs text-muted-foreground">
              加载中…
            </div>
          )}

          {!loading && items.length === 0 && !error && (
            <EmptyHint />
          )}

          {!loading && items.length > 0 && filtered.length === 0 && (
            <div className="px-3 py-6 text-center text-xs text-muted-foreground">
              没有匹配「{query}」的文档
            </div>
          )}

          {filtered.map((doc) => {
            const badge = SOURCE_BADGES[doc.source] ?? SOURCE_BADGES.note;
            const Icon = badge.icon;
            return (
              <button
                key={doc.id}
                type="button"
                onClick={() => onPick(doc)}
                className="w-full flex items-center gap-2 px-2.5 py-1.5 text-left text-sm hover:bg-accent transition-colors focus-visible:outline-none focus-visible:bg-accent"
              >
                <Icon className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                <span className="flex-1 truncate">{doc.title}</span>
                <span
                  className={cn(
                    'shrink-0 inline-flex items-center px-1.5 h-4 rounded text-[10px] font-medium',
                    badge.class
                  )}
                >
                  {badge.label}
                </span>
              </button>
            );
          })}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function EmptyHint() {
  return (
    <div className="px-3 py-8 text-center">
      <div className="inline-flex flex-col items-center gap-2 text-muted-foreground">
        <div className="h-10 w-10 rounded-full bg-primary/10 text-primary grid place-items-center">
          <NotebookPen className="h-4 w-4" />
        </div>
        <div className="text-xs font-medium text-foreground">还没有文档</div>
        <div className="text-[11px] leading-relaxed">去笔记页创建一篇后再引用</div>
      </div>
    </div>
  );
}