'use client';

import { Check } from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatRelativeDate } from '@/lib/relativeTime';
import { useThemeStore, resolveDark } from '@/presentation/stores/themeStore';
import type { Document } from '@/domain/entities/document';

interface BookCardProps {
  doc: Document;
  onOpen?: () => void;
  selected?: boolean;
  onToggleSelect?: () => void;
}

// doc.id → 色相(0-360);Knuth 乘法哈希让连续 id 分布均匀,每本书刷新稳定。
function hueForId(id: number): number {
  return ((id * 2654435761) >>> 0) % 360;
}

// A4 笔记本:长方形 + 左侧 spine 暗示书脊;hover 露勾选框,点勾选多选,点本体打开。
export function BookCard({ doc, onOpen, selected, onToggleSelect }: BookCardProps) {
  const showCheckbox = onToggleSelect !== undefined;

  const mode = useThemeStore((s) => s.mode);
  const hue = hueForId(doc.id);
  const isDark = resolveDark(mode);
  // spine 饱和度较高的中暗色;border 极淡 tint;深色下 88% 变刺眼白边,改 22% 保持色调。
  const spineStyle = { backgroundColor: `hsl(${hue} 60% ${isDark ? 60 : 48}%)` };
  const borderStyle = {
    borderColor: `hsl(${hue} 50% ${isDark ? 22 : 88}%)`,
  };

  return (
    <div
      className={cn(
        'group relative flex flex-col rounded-md border bg-card text-card-foreground shadow-sm hover:shadow-lg hover:-translate-y-0.5 transition-all duration-200 text-left overflow-hidden',
        'aspect-[1/1.414]',
        selected && 'ring-2 ring-primary border-primary shadow-md'
      )}
      style={borderStyle}
    >
      <span
        className="absolute inset-y-0 left-0 w-1.5 transition-opacity group-hover:opacity-70"
        style={spineStyle}
        aria-hidden
      />

      {showCheckbox && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onToggleSelect?.();
          }}
          aria-label={selected ? '取消选中' : '选中'}
          aria-pressed={selected}
          className={cn(
            'absolute top-1.5 right-1.5 h-5 w-5 rounded-sm border grid place-items-center transition-all z-10 shadow-sm',
            selected
              ? 'bg-primary border-primary text-primary-foreground opacity-100'
              : 'bg-background/90 border-muted-foreground/40 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 hover:border-primary'
          )}
        >
          {selected && <Check className="h-3 w-3" />}
        </button>
      )}

      <button
        type="button"
        onClick={onOpen}
        title={doc.title}
        className="flex-1 flex flex-col outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
      >
        <div className="flex-1 px-3 pt-3 pb-2 overflow-hidden">
          <div className="text-sm font-semibold leading-snug line-clamp-3 text-left text-foreground/90">
            {doc.title}
          </div>
        </div>
        <div className="px-3 py-1.5 border-t bg-muted/40 text-[10px] text-muted-foreground flex items-center justify-between">
          <span>{formatRelativeDate(doc.created_at)}</span>
          <ChunkDot status={doc.chunk_status} />
        </div>
      </button>
    </div>
  );
}

function ChunkDot({ status }: { status: Document['chunk_status'] }) {
  // 0=脏/未 chunk,1=已 chunk,2=失败。屏幕阅读器读得到,但视觉仍是色点。
  const color =
    status === 1
      ? 'bg-emerald-500'
      : status === 2
        ? 'bg-destructive'
        : 'bg-muted-foreground/40';
  const label =
    status === 1 ? '已索引' : status === 2 ? '索引失败' : '待索引';
  return (
    <span
      className={cn('h-1.5 w-1.5 rounded-full', color)}
      role="img"
      aria-label={label}
      title={label}
    />
  );
}