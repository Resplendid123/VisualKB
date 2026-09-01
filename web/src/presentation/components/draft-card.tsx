'use client';

import { useState } from 'react';
import { Plus, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { documentUseCases } from '@/application/chatContainer';

interface DraftCardProps {
  onCreated: (docId: number) => void;
}

// 网格第一格的虚线书本:中间 + 号,点击直接创建默认笔记并打开编辑器。
export function DraftCard({ onCreated }: DraftCardProps) {
  const [submitting, setSubmitting] = useState(false);

  const onClick = async () => {
    if (submitting) return;
    setSubmitting(true);
    try {
      const res = await documentUseCases.create.execute({
        source: 'note',
        title: '未命名笔记',
        // 后端 CreateDocument 拒绝空 content;占位用换行,trim 会自动剥掉,编辑器里看不到。
        content: '\n',
      });
      onCreated(res.document_id);
    } catch {
      setSubmitting(false);
    }
  };

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={submitting}
      aria-label="新建笔记"
      className={cn(
        'group relative flex flex-col rounded-md border-2 border-dashed border-primary/40 bg-primary/5 hover:bg-primary/10 hover:border-primary/60 transition-all duration-200 overflow-hidden text-left',
        'aspect-[1/1.414] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 hover:shadow-md'
      )}
    >
      <span className="absolute inset-y-0 left-0 w-1.5 bg-primary/20 group-hover:bg-primary/40 transition-colors" />
      <div className="flex-1 flex flex-col items-center justify-center text-primary/60 group-hover:text-primary transition-colors">
        {submitting ? (
          <Loader2 className="h-7 w-7 animate-spin" />
        ) : (
          <Plus className="h-8 w-8 transition-transform group-hover:scale-110" strokeWidth={1.5} />
        )}
        <span className="mt-1.5 text-[10px]">新建笔记</span>
      </div>
      <div className="border-t border-primary/15 bg-background/30 px-3 py-1.5 text-[10px] text-muted-foreground/70">
        点击创建
      </div>
    </button>
  );
}