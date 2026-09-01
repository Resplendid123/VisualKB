'use client';

import { useEffect, useMemo, useState } from 'react';
import {
  ChevronLeft,
  ChevronRight,
  FileText,
  Loader2,
  NotebookPen,
  Pencil,
  RefreshCw,
  X,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { documentUseCases } from '@/application/chatContainer';
import { usePanelStore } from '@/presentation/stores/panelStore';
import { useViewStore } from '@/presentation/stores/viewStore';
import type { DocumentDetail } from '@/domain/entities/document';
import { MarkdownView } from '@/lib/markdown';
import { API_BASE } from '@/infra/http/AuthClient';

// 预览面板 file 模式 — 渲染 panelStore.previewDocId 对应的文档;docId=null 显示空态。
export function DocumentPreviewView() {
  const docId = usePanelStore((s) => s.previewDocId);
  const setPreviewDocId = usePanelStore((s) => s.setPreviewDocId);
  const attachedDocs = usePanelStore((s) => s.attachedDocs);
  const setView = useViewStore((s) => s.setView);
  const setNoteEditingDocId = useViewStore((s) => s.setNoteEditingDocId);

  const [doc, setDoc] = useState<DocumentDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // refreshKey +1 触发重拉 — 复用同一份 effect,不重写一遍。
  const [refreshKey, setRefreshKey] = useState(0);

  // previewDocId 落在 attachedDocs 且总数 > 1 时显示 prev/next + 位置指示。
  const navIndex = useMemo(
    () => (docId == null ? -1 : attachedDocs.findIndex((d) => d.id === docId)),
    [attachedDocs, docId]
  );
  const canNavigate = navIndex >= 0 && attachedDocs.length > 1;
  const goPrev = () => {
    if (!canNavigate) return;
    const i = (navIndex - 1 + attachedDocs.length) % attachedDocs.length;
    setPreviewDocId(attachedDocs[i].id);
  };
  const goNext = () => {
    if (!canNavigate) return;
    const i = (navIndex + 1) % attachedDocs.length;
    setPreviewDocId(attachedDocs[i].id);
  };
  const refresh = () => setRefreshKey((k) => k + 1);

  useEffect(() => {
    if (docId == null) {
      // 同步 setState 挪到 async IIFE — 避开 set-state-in-effect 警告。
      void (async () => {
        setDoc(null);
        setLoading(false);
        setError(null);
      })();
      return;
    }
    let cancelled = false;
    void (async () => {
      if (cancelled) return;
      setLoading(true);
      setError(null);
      try {
        const d = await documentUseCases.get.execute(docId);
        if (cancelled) return;
        setDoc(d);
      } catch (e) {
        if (cancelled) return;
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [docId, refreshKey]);

  if (docId == null) {
    return (
      <EmptyState
        title="文件预览"
        hint="点输入框左下角的 @ 选择一篇文档,会在这里显示。"
      />
    );
  }

  const sourceLabel =
    doc?.source === 'note' ? '笔记' : doc?.source === 'knowledge' ? '知识' : doc?.source;

  return (
    <div className="flex flex-col h-full">
      <header className="px-5 py-3 border-b flex items-center gap-2 min-w-0">
        <FileText className="h-4 w-4 text-muted-foreground shrink-0" />
        {canNavigate && (
          <>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0"
              onClick={goPrev}
              aria-label="上一个文档"
              title="上一个文档"
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-xs text-muted-foreground tabular-nums shrink-0">
              {navIndex + 1} / {attachedDocs.length}
            </span>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0"
              onClick={goNext}
              aria-label="下一个文档"
              title="下一个文档"
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </>
        )}
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium truncate">{doc?.title}</div>
          <div className="text-xs text-muted-foreground mt-0.5">
            {sourceLabel} · {doc?.content_type}
          </div>
        </div>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0"
          disabled={loading}
          onClick={refresh}
          aria-label="刷新"
          title="刷新"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0"
          onClick={() => {
            // 标记来源 = chat,让笔记编辑器右上角按钮显示"回到会话"而不是"去对话"。
            if (doc) {
              setNoteEditingDocId({ id: doc.id, from: 'chat' });
              setView('notes');
            }
          }}
          aria-label="去编辑"
          title="去编辑"
        >
          <Pencil className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0"
          onClick={() => setPreviewDocId(null)}
          aria-label="清空预览"
        >
          <X className="h-3.5 w-3.5" />
        </Button>
      </header>

      <PreviewBody docId={docId} doc={doc} loading={loading} error={error} source={doc?.source} />
    </div>
  );
}

// 预览正文:loading/error/pdf/markdown/empty 共用,header 由各调用方画。
export function PreviewBody({
  docId,
  doc,
  loading,
  error,
  source,
}: {
  docId: number;
  doc: DocumentDetail | null;
  loading: boolean;
  error: string | null;
  source: DocumentDetail['source'] | undefined;
}) {
  if (loading && !doc) {
    return (
      <div className="px-4 py-6 text-xs text-muted-foreground flex items-center gap-2">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        加载中…
      </div>
    );
  }

  if (error) {
    return (
      <div className="mx-4 my-3 px-3 py-2 rounded-md bg-destructive/10 text-xs text-destructive">
        {error}
      </div>
    );
  }

  if (!doc) return null;

  const hasContent = doc.content.trim().length > 0;

  return (
    <div className="flex-1 overflow-y-auto px-6 py-5">
      {doc.content_type === 'pdf' ? (
        <iframe
          src={`${API_BASE}/api/v1/documents/${docId}/file`}
          className="w-full h-full min-h-[60vh] bg-white rounded-md border"
          title={doc.title}
        />
      ) : hasContent ? (
        <MarkdownView content={doc.content} />
      ) : (
        <div className="text-sm text-muted-foreground/70 space-y-1">
          <div className="italic">（文档正文为空）</div>
          <div className="text-xs">
            {source === 'note'
              ? '新建的笔记还没保存内容,先去笔记页写点东西再 @ 过来。'
              : '该文档尚未写入正文。'}
          </div>
        </div>
      )}
    </div>
  );
}

function EmptyState({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="px-6 py-12 text-center">
      <div className="inline-flex flex-col items-center gap-3 text-muted-foreground max-w-xs">
        <div className="h-12 w-12 rounded-full bg-primary/10 text-primary grid place-items-center">
          <NotebookPen className="h-5 w-5" />
        </div>
        <div className="text-sm font-medium text-foreground">{title}</div>
        <div className="text-xs leading-relaxed">{hint}</div>
      </div>
    </div>
  );
}
