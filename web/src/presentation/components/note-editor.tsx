'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  ArrowLeft,
  Columns2,
  CornerUpLeft,
  Eye,
  FileEdit,
  Loader2,
  MessageSquarePlus,
  Trash2,
  Check,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { documentUseCases } from '@/application/chatContainer';
import { MarkdownView } from '@/lib/markdown';
import { useViewStore } from '@/presentation/stores/viewStore';
import { useAsyncState } from '@/presentation/hooks/useAsyncState';
import { showToast } from './toast';

interface NoteEditorProps {
  documentId: number;
  // 编辑器打开来源:chat 表示从会话预览面板过来,按钮变「回到会话」。
  source: 'chat' | 'notes';
  onBack: () => void;
  onSaved?: () => void;
  onDeleted?: () => void;
}

type ViewMode = 'edit' | 'preview' | 'split';

const VIEW_MODE_OPTIONS: {
  value: ViewMode;
  label: string;
  Icon: typeof FileEdit;
}[] = [
  { value: 'edit', label: '编辑', Icon: FileEdit },
  { value: 'preview', label: '预览', Icon: Eye },
  { value: 'split', label: '分屏', Icon: Columns2 },
];

// 算「第几行 · 第几列 · 总字数 · 已选字数」,光标或选区变化时更新。
function computeStatus(content: string, start: number, end: number) {
  const before = content.slice(0, start);
  const line = before.split('\n').length;
  const col = start - before.lastIndexOf('\n');
  return { line, col, chars: content.length, selected: end - start };
}

// 笔记编辑器:顶栏 + 左编辑/右 Markdown 预览。
export function NoteEditor({ documentId, source, onBack, onSaved, onDeleted }: NoteEditorProps) {
  const { data: doc, loading, error } = useAsyncState(
    () => documentUseCases.get.execute(documentId),
    [documentId]
  );
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [savedAt, setSavedAt] = useState<number | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('split');
  // 状态栏:行/列/总字数/已选字数;光标或选区变化时更新。
  const [status, setStatus] = useState({ line: 1, col: 1, chars: 0, selected: 0 });
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // 文档载入完成时,把标题 / 正文一次性搬到本地可编辑状态。
  useEffect(() => {
    if (!doc) return;
    setTitle(doc.title);
    const stripped = doc.content.replace(/\u200B/g, '').trim();
    setContent(stripped === '' ? '' : doc.content);
  }, [doc]);

  const trimmedContent = content.trim();
  const canSave = !loading && !saving && trimmedContent.length > 0;

  const onSave = useCallback(async () => {
    if (!canSave) return;
    setSaving(true);
    try {
      await documentUseCases.update.execute(
        documentId,
        content,
        title.trim() || '未命名笔记'
      );
      setSavedAt(Date.now());
      onSaved?.();
    } catch (e) {
      showToast(e instanceof Error ? e.message : '保存失败', 'error');
    } finally {
      setSaving(false);
    }
  }, [canSave, content, documentId, title, onSaved]);

  const setPendingChatIntent = useViewStore((s) => s.setPendingChatIntent);
  const router = useRouter();
  // 拉一次最新 doc — chip 显示已保存的标题,而不是编辑器里的脏 title。
  const onGoToChat = useCallback(async () => {
    try {
      const d = await documentUseCases.get.execute(documentId);
      setPendingChatIntent({ kind: 'new-conversation', doc: d });
      router.push('/');
    } catch {
      showToast('无法打开会话,请重试', 'error');
    }
  }, [documentId, setPendingChatIntent, router]);

  // onSelect 覆盖点击/键盘移动/拖选;onChange 单独调,因为部分浏览器键入时不一定触发 onSelect。
  const updateStatus = useCallback(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    setStatus(computeStatus(content, ta.selectionStart, ta.selectionEnd));
  }, [content]);

  // Ctrl/Cmd+S 全局快捷保存 — 用 ref 捕获最新 onSave,effect deps 保持空。
  const saveRef = useRef(onSave);
  useEffect(() => {
    saveRef.current = onSave;
  }, [onSave]);
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        void saveRef.current();
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const onDelete = useCallback(async () => {
    if (deleting) return;
    if (!confirm(`确定归档笔记「${title || '未命名笔记'}」？`)) return;
    setDeleting(true);
    try {
      await documentUseCases.archive.execute(documentId);
      onDeleted?.();
    } catch (e) {
      showToast(e instanceof Error ? e.message : '归档失败', 'error');
      setDeleting(false);
    }
  }, [deleting, documentId, title, onDeleted]);

  // Tab/Shift+Tab 缩进/反缩进,光标在多行中间也对每行生效。
  function handleEditorKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key !== 'Tab') return;
    e.preventDefault();
    const ta = e.currentTarget;
    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    const value = content;
    const indent = '  ';
    const before = value.slice(0, start);
    const after = value.slice(end);
    if (e.shiftKey) {
      const lineStart = before.lastIndexOf('\n') + 1;
      const head = value.slice(0, lineStart);
      const sel = value.slice(lineStart, end);
      const dedented = sel.replace(/^( {2})/gm, '');
      const removed = sel.length - dedented.length;
      const next = head + dedented + after;
      setContent(next);
      requestAnimationFrame(() => {
        ta.selectionStart = lineStart + Math.max(0, start - lineStart - (start === end ? indent.length : Math.min(indent.length, removed)));
        ta.selectionEnd = lineStart + Math.max(0, end - lineStart - removed);
      });
      return;
    }
    if (start === end) {
      const next = before + indent + after;
      setContent(next);
      requestAnimationFrame(() => {
        ta.selectionStart = ta.selectionEnd = start + indent.length;
      });
      return;
    }
    const lineStart = before.lastIndexOf('\n') + 1;
    const head = value.slice(0, lineStart);
    const sel = value.slice(lineStart, end);
    const indented = sel.replace(/^/gm, indent);
    const added = indented.length - sel.length;
    const next = head + indented + after;
    setContent(next);
    requestAnimationFrame(() => {
      ta.selectionStart = start + indent.length;
      ta.selectionEnd = end + added;
    });
  }

  return (
    <div className="flex flex-col h-full">
      <div className="shrink-0 flex items-center justify-between px-5 py-3 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/70 gap-3">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 shrink-0"
            onClick={onBack}
            aria-label="返回笔记列表"
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="无标题"
            className="h-9 max-w-md text-base font-semibold border-transparent bg-transparent hover:bg-muted/40 focus-visible:bg-background px-2"
            disabled={loading}
          />
          {savedAt && !saving && !error && (
            <span className="inline-flex items-center gap-1 px-2 h-5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 text-[10px] font-medium">
              <Check className="h-3 w-3" />
              已保存
            </span>
          )}
          {saving && (
            <span className="inline-flex items-center gap-1 px-2 h-5 rounded-full bg-muted text-muted-foreground text-[10px] font-medium">
              <Loader2 className="h-3 w-3 animate-spin" />
              保存中
            </span>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <ViewModeToggle value={viewMode} onChange={setViewMode} />
          {source === 'chat' ? (
            // 从会话预览面板过来 — 直接切回 chat,preview 面板侧原样保留。
            <Button
              variant="ghost"
              size="sm"
              className="h-8 px-3 text-xs gap-1.5"
              onClick={() => router.push('/')}
              aria-label="回到会话"
              title="回到会话"
            >
              <CornerUpLeft className="h-3.5 w-3.5" />
              回到会话
            </Button>
          ) : (
            // 网格点开:起草稿会话 + 自动 @ 当前笔记 + 展开预览面板。
            <Button
              variant="ghost"
              size="sm"
              className="h-8 px-3 text-xs gap-1.5"
              onClick={() => void onGoToChat()}
              aria-label="去对话"
              title="去对话"
            >
              <MessageSquarePlus className="h-3.5 w-3.5" />
              去对话
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
            onClick={() => void onDelete()}
            disabled={loading || deleting}
            aria-label="归档笔记"
            title="归档"
          >
            {deleting ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Trash2 className="h-3.5 w-3.5" />
            )}
          </Button>
          <Button
            size="sm"
            className="h-8 px-3"
            onClick={() => void onSave()}
            disabled={!canSave}
          >
            {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : '保存'}
          </Button>
        </div>
      </div>

      <div
        className={
          viewMode === 'split'
            ? 'flex-1 min-h-0 grid grid-cols-2 gap-px bg-border'
            : 'flex-1 min-h-0 grid grid-cols-1'
        }
      >
        {/* min-h-0 让 grid cell 内的 flex 列能真的约束到行高,否则 textarea 撑高页面而 pane 不滚。 */}
        <div
          className={
            viewMode === 'preview'
              ? 'hidden'
              : 'flex flex-col min-h-0 bg-background'
          }
        >
          <PaneLabel>编辑</PaneLabel>
          <Textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => {
              setContent(e.target.value);
              // 键入后光标位置变化,onSelect 不一定准时,主动刷一次。
              requestAnimationFrame(updateStatus);
            }}
            onSelect={updateStatus}
            onKeyDown={handleEditorKeyDown}
            placeholder="开始写点什么…"
            className="flex-1 min-h-0 rounded-none border-0 font-mono text-[13px] leading-7 resize-none focus-visible:ring-0 focus-visible:border-0 px-5 py-4"
            disabled={loading}
          />
        </div>
        <div
          className={
            viewMode === 'edit'
              ? 'hidden'
              : 'flex flex-col min-h-0 bg-background overflow-hidden'
          }
        >
          <PaneLabel>预览</PaneLabel>
          <div className="flex-1 min-h-0 overflow-y-auto px-6 py-5">
            {error && (
              <div className="text-xs text-destructive mb-3 px-3 py-2 rounded-md bg-destructive/10">
                {error}
              </div>
            )}
            {trimmedContent.length === 0 ? (
              <div className="text-sm text-muted-foreground/70 italic">
                {viewMode === 'split' ? '右侧会显示 Markdown 预览。' : '内容为空时无预览可显示。'}
              </div>
            ) : (
              <MarkdownView content={trimmedContent} />
            )}
          </div>
        </div>
      </div>

      <StatusBar status={status} />
    </div>
  );
}

// 分栏顶部的小标签 — 编辑/预览。
function PaneLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="shrink-0 px-5 py-2 text-[10px] uppercase tracking-wider font-medium text-muted-foreground border-b bg-muted/30">
      {children}
    </div>
  );
}

// 底部状态栏:行/列/总字数/已选字数,光标或选区变化时实时更新。
function StatusBar({
  status,
}: {
  status: { line: number; col: number; chars: number; selected: number };
}) {
  return (
    <div className="shrink-0 px-5 py-1.5 bg-muted/40 text-[11px] text-muted-foreground flex items-center justify-end gap-3 tabular-nums">
      <span>
        第 {status.line} 行 · 第 {status.col} 列
      </span>
      <span>· {status.chars} 字</span>
      {status.selected > 0 && <span>· 已选 {status.selected}</span>}
    </div>
  );
}

// 编辑/预览/分屏 三态切换 — radiogroup,跟主题切换同款 segmented control 样式。
function ViewModeToggle({
  value,
  onChange,
}: {
  value: ViewMode;
  onChange: (v: ViewMode) => void;
}) {
  return (
    <div
      role="radiogroup"
      aria-label="视图模式"
      className="inline-flex rounded-md border bg-muted/40 p-0.5"
    >
      {VIEW_MODE_OPTIONS.map(({ value: v, label, Icon }) => {
        const active = v === value;
        return (
          <button
            key={v}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={label}
            title={label}
            onClick={() => onChange(v)}
            className={
              active
                ? 'inline-flex items-center gap-1 h-7 px-2 rounded text-[11px] font-medium bg-background shadow-sm text-foreground'
                : 'inline-flex items-center gap-1 h-7 px-2 rounded text-[11px] font-medium text-muted-foreground hover:text-foreground'
            }
          >
            <Icon className="h-3.5 w-3.5" />
            {label}
          </button>
        );
      })}
    </div>
  );
}