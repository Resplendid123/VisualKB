'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import {
  Loader2,
  NotebookPen,
  RefreshCw,
  Trash2,
  Search,
  Wand2,
  X,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { documentUseCases } from '@/application/chatContainer';
import type {
  Document,
  ListDocumentsInput,
} from '@/domain/entities/document';
import { BookCard } from './book-card';
import { DraftCard } from './draft-card';
import { NoteEditor } from './note-editor';
import { useViewStore } from '@/presentation/stores/viewStore';
import { showToast } from './toast';
import { useAsyncState } from '@/presentation/hooks/useAsyncState';

// 笔记视图:A4 书本网格 + 顶栏搜索/多选删除;首格常驻虚线卡 + 号。
export function NotesPanel() {
  const fetchNotes = useCallback(
    () => documentUseCases.list.execute({ source: 'note', limit: 60, offset: 0 }),
    []
  );
  const { data, loading, error, reload } = useAsyncState(fetchNotes, []);
  const items = data?.items ?? [];
  const [query, setQuery] = useState('');
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [selectedId, setSelectedId] = useState<number | null>(null);
  // 编辑器来源信号 — 必须在 store clear 前捕获,否则 source 拿不到。
  // 网格点开 / 新建都保持默认 'notes',chat 预览 Pencil 点开会变成 'chat'。
  const [editorSource, setEditorSource] = useState<'chat' | 'notes'>('notes');
  const [isIngestingAll, setIsIngestingAll] = useState(false);

  // URL → selectedId 同步:URL 是 single source of truth。
  // path /notes/[docId] → selectedId = Number(docId);其他情况 → null(网格态)。
  const router = useRouter();
  const pathname = usePathname();
  useEffect(() => {
    const m = pathname?.match(/^\/notes\/([^/?]+)/);
    if (!m) {
      if (selectedId !== null) setSelectedId(null);
      return;
    }
    const id = Number(decodeURIComponent(m[1]));
    if (Number.isFinite(id) && id !== selectedId) setSelectedId(id);
  }, [pathname]); // eslint-disable-line react-hooks/exhaustive-deps

  // 消费 viewStore 里待打开的 docId(预览面板的「去编辑」会写进来)。
  // 写回 null 是为了 NotesPanel 卸载/重挂时不重复打开同一篇。
  const noteEditingDocId = useViewStore((s) => s.noteEditingDocId);
  const setNoteEditingDocId = useViewStore((s) => s.setNoteEditingDocId);
  useEffect(() => {
    if (noteEditingDocId === null) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setSelectedId(noteEditingDocId.id);
    setEditorSource(noteEditingDocId.from);
    setNoteEditingDocId(null);
  }, [noteEditingDocId, setNoteEditingDocId]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return items;
    return items.filter((d) => d.title.toLowerCase().includes(q));
  }, [items, query]);

  // 待 ingest 数 = 非 chunked 的笔记(chunk_status 0 dirty / 2 error)。
  const unchunkedCount = useMemo(
    () => items.filter((d) => d.chunk_status !== 1).length,
    [items]
  );

  // 全量派发笔记 ingest;note 上传时不自动 ingest,等 worker 周期或这里手动触发。
  const onIngestAll = useCallback(async () => {
    if (isIngestingAll || unchunkedCount === 0) return;
    setIsIngestingAll(true);
    try {
      const enqueued = await documentUseCases.ingestAll.execute('note');
      showToast(
        enqueued === 0
          ? '没有需要 ingest 的笔记'
          : `已派发 ${enqueued} 篇笔记到后台`,
        'success'
      );
      await reload();
    } catch (e) {
      showToast(
        e instanceof Error ? `ingest 失败: ${e.message}` : 'ingest 失败',
        'error'
      );
    } finally {
      setIsIngestingAll(false);
    }
  }, [isIngestingAll, unchunkedCount, reload]);

  const toggleSelect = useCallback((id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const clearSelection = useCallback(() => setSelectedIds(new Set()), []);

  const onDeleteSelected = useCallback(async () => {
    const ids = [...selectedIds];
    if (ids.length === 0) return;
    if (!confirm(`确定归档选中的 ${ids.length} 篇笔记？`)) return;
    try {
      await Promise.all(ids.map((id) => documentUseCases.archive.execute(id)));
      setSelectedIds(new Set());
      void reload();
    } catch (e) {
      showToast(e instanceof Error ? e.message : '归档失败', 'error');
    }
  }, [selectedIds, reload]);

  // 新建的笔记永远走 'notes' 来源 — 没必要继承上次 chat 预览留下的标志。
  const onCreated = useCallback(
    (docId: number) => {
      setEditorSource('notes');
      // URL 同步:推到 /notes/[docId],hook 内的 effect 反向同步 selectedId。
      router.push(`/notes/${docId}`);
      void reload();
    },
    [reload, router]
  );

  if (selectedId !== null) {
    return (
      <NoteEditor
        documentId={selectedId}
        source={editorSource}
        onBack={() => {
          // URL 是 single source of truth:推回 /notes,effect 同步回 selectedId=null。
          router.push('/notes');
          clearSelection();
        }}
        onSaved={() => {
          void reload();
        }}
        onDeleted={() => {
          router.push('/notes');
          clearSelection();
          void reload();
        }}
      />
    );
  }

  const hasQuery = query.trim().length > 0;

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between gap-3 px-5 py-3.5 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/70">
        <div className="flex items-center gap-2 shrink-0">
          <div className="h-7 w-7 rounded-md bg-primary/10 text-primary grid place-items-center">
            <NotebookPen className="h-3.5 w-3.5" />
          </div>
          <h3 className="text-sm font-semibold">笔记</h3>
          <span className="text-xs text-muted-foreground">{items.length}</span>
          {selectedIds.size > 0 && (
            <span className="ml-1 inline-flex items-center gap-1 px-1.5 h-5 rounded-full bg-primary/10 text-primary text-[10px] font-medium">
              已选 {selectedIds.size}
              <button
                onClick={clearSelection}
                className="hover:text-foreground"
                aria-label="清空选择"
              >
                <X className="h-2.5 w-2.5" />
              </button>
            </span>
          )}
        </div>

        <div className="flex items-center gap-1 flex-1 justify-end min-w-0">
          <div className="relative w-full max-w-xs">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索笔记…"
              className="h-8 pl-8 pr-7 text-xs bg-muted/40 border-transparent focus-visible:bg-background"
            />
            {query && (
              <button
                onClick={() => setQuery('')}
                aria-label="清空搜索"
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 px-2 text-xs gap-1"
            onClick={() => void onIngestAll()}
            disabled={isIngestingAll || unchunkedCount === 0}
            aria-label="一键 ingest"
            title="一键 ingest"
          >
            {isIngestingAll ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Wand2 className="h-3.5 w-3.5" />
            )}
            <span>一键 ingest ({unchunkedCount})</span>
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => void reload()}
            aria-label="刷新"
          >
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
          {selectedIds.size > 0 && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={() => void onDeleteSelected()}
              aria-label={`删除 ${selectedIds.size} 项`}
              title={`归档 ${selectedIds.size} 项`}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-5">
        {error && (
          <div className="text-xs text-destructive mb-3 px-3 py-2 rounded-md bg-destructive/10">
            {error}
          </div>
        )}

        <div className="grid grid-cols-5 gap-4">
          <DraftCard onCreated={onCreated} />
          {filtered.map((doc) => (
            <BookCard
              key={doc.id}
              doc={doc}
              selected={selectedIds.has(doc.id)}
              onToggleSelect={() => toggleSelect(doc.id)}
              onOpen={() => {
            // 每次从网格打开都重置来源 — 否则上次从 chat 过来留下的 'chat' 会让按钮变「回到会话」。
            setEditorSource('notes');
            // URL 同步:推到 /notes/[docId],effect 反向同步 selectedId。
            router.push(`/notes/${doc.id}`);
          }}
            />
          ))}
          {!loading && items.length === 0 && !hasQuery && <EmptyHint creating />}
          {!loading && items.length > 0 && filtered.length === 0 && hasQuery && (
            <div className="col-span-5 py-12 text-center">
              <div className="inline-flex flex-col items-center gap-2 text-muted-foreground">
                <Search className="h-6 w-6 opacity-40" />
                <div className="text-xs">没有匹配「{query}」的笔记</div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// 库里无笔记时的提示 — 给空状态一个柔和的引导。
function EmptyHint({ creating }: { creating?: boolean }) {
  return (
    <div className="col-span-5 py-12 text-center">
      <div className="inline-flex flex-col items-center gap-3 text-muted-foreground max-w-xs">
        <div className="h-12 w-12 rounded-full bg-primary/10 text-primary grid place-items-center">
          <NotebookPen className="h-5 w-5" />
        </div>
        {creating ? (
          <>
            <div className="text-sm font-medium text-foreground">还没有笔记</div>
            <div className="text-xs leading-relaxed">
              点击网格第一格的虚线卡,写下第一篇笔记。
            </div>
          </>
        ) : (
          <div className="text-xs">没有匹配的笔记。</div>
        )}
      </div>
    </div>
  );
}