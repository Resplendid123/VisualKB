'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Database,
  FileText,
  Loader2,
  NotebookPen,
  RefreshCw,
  Trash2,
  Wand2,
  X,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { documentUseCases, treeUseCases } from '@/application/chatContainer';
import type { Document, DocumentDetail } from '@/domain/entities/document';
import type { TreeNode } from '@/domain/entities/tree';
import { findNodeByDocId, getAncestorNames } from '@/domain/repositories/treeRepository';
import { useViewStore } from '@/presentation/stores/viewStore';
import { showToast } from './toast';
import { PreviewBody } from './document-preview-view';
import { FolderRenameDialog } from './folder-rename-dialog';
import { PathTree } from './path-tree';
import { ConfirmDialog } from './confirm-dialog';

// 知识库视图:左树(含 doc rows + 拖拽移动)|右单文档预览;上传/删除/移动集中在树或预览头。
export function KnowledgePanel() {
  const selectedDocId = useViewStore((s) => s.knowledgeSelectedDocId);
  const setSelectedDocId = useViewStore((s) => s.setKnowledgeSelectedDocId);
  const expanded = useViewStore((s) => s.knowledgeExpandedFolderIds);
  const setKnowledgeFolderExpanded = useViewStore(
    (s) => s.setKnowledgeFolderExpanded
  );
  const resetKnowledgeExpanded = useViewStore(
    (s) => s.resetKnowledgeExpanded
  );

  const [treeNodes, setTreeNodes] = useState<TreeNode[]>([]);
  const [items, setItems] = useState<Document[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [dropTargetNodeId, setDropTargetNodeId] = useState<number | null>(null);
  const [renameTarget, setRenameTarget] = useState<number | null>(null);
  const [draftFolder, setDraftFolder] = useState<{
    parentNodeId: number | null;
    name: string;
  } | null>(null);
  const [previewDocId, setPreviewDocId] = useState<number | null>(null);
  const [pendingUploadParent, setPendingUploadParent] = useState<
    number | null | undefined
  >(undefined);
  const [deleteFolderTarget, setDeleteFolderTarget] = useState<{
    id: number;
    name: string;
    folderCount: number;
    docCount: number;
  } | null>(null);
  const [archiveDocTarget, setArchiveDocTarget] = useState<{
    id: number;
    title: string;
  } | null>(null);
  const [isIngestingAll, setIsIngestingAll] = useState(false);
  const [sidebarWidth, setSidebarWidth] = useState(288);
  const sidebarDragRef = useRef<{ startX: number; startWidth: number } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const reloadTree = useCallback(async () => {
    try {
      const res = await treeUseCases.list.execute();
      setTreeNodes(res.nodes);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  // tree 内部展示所有 knowledge 文档;选中目录只影响 right pane 的"下一步操作"上下文。
  const reloadDocs = useCallback(async () => {
    try {
      const res = await documentUseCases.list.execute({
        source: 'knowledge',
        limit: 500,
        offset: 0,
      });
      setItems(res.items);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  // 挂载时同步拉一次树和文档。setState 全放进 await 之后的 Promise 回调,避开 react-hooks/set-state-in-effect。
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const [tree, docs] = await Promise.all([
          treeUseCases.list.execute(),
          documentUseCases.list.execute({
            source: 'knowledge',
            limit: 500,
            offset: 0,
          }),
        ]);
        if (cancelled) return;
        setTreeNodes(tree.nodes);
        setItems(docs.items);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // 预览 doc 的祖先路径直接从 treeNodes 派生,tree 刷新时自动同步,无需 useEffect。
  const previewFolderPath = useMemo(
    () => (previewDocId == null ? null : getAncestorNames(treeNodes, previewDocId)),
    [treeNodes, previewDocId]
  );

  // 待 ingest 数 = 非 chunked 的 knowledge 文档(chunk_status 0 dirty / 2 error)。
  const unchunkedCount = useMemo(
    () => items.filter((d) => d.chunk_status !== 1).length,
    [items]
  );

  // 全量派发 ingest;后端只返 enqueued 数,实际进度靠 tree/docs 刷新观察。
  const onIngestAll = async () => {
    if (isIngestingAll || unchunkedCount === 0) return;
    setIsIngestingAll(true);
    try {
      const enqueued = await documentUseCases.ingestAll.execute('knowledge');
      showToast(
        enqueued === 0
          ? '没有需要 ingest 的文档'
          : `已派发 ${enqueued} 个文档到后台`,
        'success'
      );
      await reloadDocs();
      await reloadTree();
    } catch (e) {
      showToast(
        e instanceof Error ? `ingest 失败: ${e.message}` : 'ingest 失败',
        'error'
      );
    } finally {
      setIsIngestingAll(false);
    }
  };

  // 侧栏拖拽:pointer capture 拖到抬起,clamp [200, 480];ref 持有起点的 x/width,避免每次重渲染抖动。
  const onSidebarResizeStart = (e: React.PointerEvent<HTMLDivElement>) => {
    sidebarDragRef.current = { startX: e.clientX, startWidth: sidebarWidth };
    e.currentTarget.setPointerCapture(e.pointerId);
  };
  const onSidebarResizeMove = (e: React.PointerEvent<HTMLDivElement>) => {
    const s = sidebarDragRef.current;
    if (!s) return;
    const dx = e.clientX - s.startX;
    setSidebarWidth(Math.min(480, Math.max(200, s.startWidth + dx)));
  };
  const onSidebarResizeEnd = (e: React.PointerEvent<HTMLDivElement>) => {
    sidebarDragRef.current = null;
    if (e.currentTarget.hasPointerCapture(e.pointerId)) {
      e.currentTarget.releasePointerCapture(e.pointerId);
    }
  };

  const onUploadFile = async (file: File, parent: number | null) => {
    setError(null);
    try {
      const title = file.name.replace(/\.[^.]+$/, '');
      const isPDF =
        file.type === 'application/pdf' || /\.pdf$/i.test(file.name);
      if (isPDF) {
        await documentUseCases.upload.execute({
          source: 'knowledge',
          title,
          parent_tree_id: parent,
          file,
        });
      } else {
        const text = await file.text();
        await documentUseCases.create.execute({
          source: 'knowledge',
          title,
          parent_tree_id: parent,
          content: text,
        });
      }
      await reloadDocs();
      await reloadTree();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  // 把文档的所有祖先目录全部展开,使预览切换时可见。
  const expandAncestors = (docId: number) => {
    const node = findNodeByDocId(treeNodes, docId);
    let cur = node?.parentId ?? null;
    while (cur != null) {
      setKnowledgeFolderExpanded(cur, true);
      cur = treeNodes.find((n) => n.id === cur)?.parentId ?? null;
    }
  };

  const openPreview = (doc: Document) => {
    setSelectedDocId(doc.id);
    setPreviewDocId(doc.id);
    expandAncestors(doc.id);
  };

  const closePreview = () => {
    setPreviewDocId(null);
    setSelectedDocId(null);
  };

  // 拖拽移动:tree 里的 doc row 是 source,folder row 是 target。
  // 走 /documents/:id/move(吃 documents.id + parent_tree_id),不走 /tree/folder/:id/move(那个只接 folder)。
  const handleDropDoc = async (docId: number, targetFolderId: number | null) => {
    const node = findNodeByDocId(treeNodes, docId);
    if (!node) {
      setDropTargetNodeId(null);
      return;
    }
    if (node.parentId === targetFolderId) {
      setDropTargetNodeId(null);
      return;
    }
    setError(null);
    try {
      await documentUseCases.move.execute(docId, targetFolderId);
      await reloadTree();
      await reloadDocs();
      if (targetFolderId != null) setKnowledgeFolderExpanded(targetFolderId, true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setDropTargetNodeId(null);
    }
  };

  // 拖动 folder 落到另一个 folder 上 —— TreeService.MoveNode(sourceFolderId, targetFolderId)。
  const handleMoveFolder = async (
    sourceFolderId: number,
    targetFolderId: number | null
  ) => {
    setError(null);
    try {
      const res = await treeUseCases.moveNode.execute(sourceFolderId, targetFolderId);
      await reloadTree();
      await reloadDocs();
      setKnowledgeFolderExpanded(res.id, true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setDropTargetNodeId(null);
    }
  };

  // 目录操作 —— create / delete / rename 都通过 treeUseCases。
  const onStartCreateFolder = (parent: number | null) => {
    setDraftFolder({ parentNodeId: parent, name: '' });
  };

  const onDraftChange = (name: string) => {
    setDraftFolder((prev) => (prev ? { ...prev, name } : prev));
  };

  const onDraftCommit = async () => {
    if (!draftFolder) return;
    const trimmed = draftFolder.name.trim();
    if (trimmed === '') {
      setDraftFolder(null);
      return;
    }
    const parent = draftFolder.parentNodeId;
    setDraftFolder(null);
    try {
      const res = await treeUseCases.createFolder.execute(parent, trimmed);
      setKnowledgeFolderExpanded(res.id, true);
      if (parent != null) setKnowledgeFolderExpanded(parent, true);
      await reloadTree();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const onDraftCancel = () => setDraftFolder(null);

  // PathTree 请求"在此目录下上传":记下 parent,触发隐藏的文件输入框。
  const onUploadRequest = (parent: number | null) => {
    setPendingUploadParent(parent);
    fileInputRef.current?.click();
  };

  // 递归统计 folder 子树里的 folder 数(含自身)和 doc 指针数,用于删除前的提示。
  const countSubtreeImpact = (folderId: number) => {
    const byParent = new Map<number | null, TreeNode[]>();
    for (const n of treeNodes) {
      const arr = byParent.get(n.parentId) ?? [];
      arr.push(n);
      byParent.set(n.parentId, arr);
    }
    let folderCount = 0;
    let docCount = 0;
    const walk = (id: number) => {
      folderCount += 1;
      for (const c of byParent.get(id) ?? []) {
        if (c.isFolder) walk(c.id);
        else docCount += 1;
      }
    };
    walk(folderId);
    return { folderCount, docCount };
  };

  const onDeleteFolder = (folderId: number) => {
    const node = treeNodes.find((n) => n.id === folderId);
    if (!node) return;
    const { folderCount, docCount } = countSubtreeImpact(folderId);
    setDeleteFolderTarget({
      id: folderId,
      name: node.name,
      folderCount,
      docCount,
    });
  };

  const confirmDeleteFolder = async () => {
    if (deleteFolderTarget == null) return;
    setError(null);
    try {
      await treeUseCases.deleteFolder.execute(deleteFolderTarget.id);
      // 选中/预览的 doc 若属于被删的子目录,后端 list 已经不带回来;主动清掉高亮避免脏状态。
      if (previewDocId != null && !treeNodes.some((n) => n.docId === previewDocId)) {
        setPreviewDocId(null);
        setSelectedDocId(null);
      }
      resetKnowledgeExpanded();
      setDeleteFolderTarget(null);
      await reloadTree();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      throw e;
    }
  };

  const onRenameFolder = async (newName: string) => {
    if (renameTarget === null) return;
    setError(null);
    try {
      await treeUseCases.renameNode.execute(renameTarget, newName);
      await reloadTree();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      throw e;
    }
  };

  const onDeletePreview = () => {
    if (previewDocId == null) return;
    const doc = items.find((d) => d.id === previewDocId);
    setArchiveDocTarget({
      id: previewDocId,
      title: doc?.title ?? `#${previewDocId}`,
    });
  };

  const confirmArchiveDoc = async () => {
    if (archiveDocTarget == null) return;
    setError(null);
    try {
      await documentUseCases.archive.execute(archiveDocTarget.id);
      setArchiveDocTarget(null);
      closePreview();
      await reloadDocs();
      await reloadTree();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      throw e;
    }
  };

  return (
    <div className="flex h-full min-h-0">
      <aside
        className="shrink-0 border-r flex flex-col bg-muted/20"
        style={{ width: sidebarWidth }}
      >
        <div className="flex items-center justify-between px-3 py-3 border-b">
          <div className="flex items-center gap-2">
            <div className="h-6 w-6 rounded-md bg-primary/10 text-primary grid place-items-center">
              <Database className="h-3.5 w-3.5" />
            </div>
            <h3 className="text-sm font-semibold">知识库</h3>
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs gap-1"
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
              className="h-7 w-7"
              onClick={() => {
                void reloadTree();
                void reloadDocs();
              }}
              aria-label="刷新"
            >
              <RefreshCw className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
        <PathTree
          nodes={treeNodes}
          documents={items}
          onOpenDoc={openPreview}
          selectedDocId={selectedDocId}
          expanded={expanded}
          onToggleFolder={(id, open) => setKnowledgeFolderExpanded(id, open)}
          dropTargetNodeId={dropTargetNodeId}
          onDropDoc={handleDropDoc}
          onMoveFolder={handleMoveFolder}
          onSetDropTarget={setDropTargetNodeId}
          onCreateFolder={onStartCreateFolder}
          onUploadRequest={onUploadRequest}
          onRenameRequest={(id) => setRenameTarget(id)}
          onDeleteRequest={onDeleteFolder}
          draftFolder={draftFolder}
          onDraftFolderChange={onDraftChange}
          onDraftFolderCommit={() => void onDraftCommit()}
          onDraftFolderCancel={onDraftCancel}
        />
        {/* 隐藏的 file input;由每个目录行的 "+" → "上传文件" 触发。 */}
        <input
          ref={fileInputRef}
          type="file"
          accept=".md,.markdown,.txt,.pdf,text/markdown,text/plain,application/pdf"
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f && pendingUploadParent !== undefined) {
              void onUploadFile(f, pendingUploadParent);
            }
            setPendingUploadParent(undefined);
            e.currentTarget.value = '';
          }}
        />
      </aside>

      <div
        className="w-1 shrink-0 cursor-col-resize hover:bg-primary/30 active:bg-primary/40 transition-colors focus-visible:w-1.5 focus-visible:bg-primary/30 outline-none"
        role="separator"
        aria-orientation="vertical"
        aria-valuenow={sidebarWidth}
        aria-valuemin={200}
        aria-valuemax={480}
        aria-label="调整目录栏宽度"
        tabIndex={0}
        onKeyDown={(e) => {
          // 跟侧栏一致:←/→ 调宽度,Shift 加步,Home/End 跳边界。
          // 目录栏在左侧,语义反一下:← 缩,→ 宽。
          const step = e.shiftKey ? 64 : 16;
          if (e.key === 'ArrowLeft') {
            e.preventDefault();
            setSidebarWidth(Math.min(480, Math.max(200, sidebarWidth - step)));
          } else if (e.key === 'ArrowRight') {
            e.preventDefault();
            setSidebarWidth(Math.min(480, Math.max(200, sidebarWidth + step)));
          } else if (e.key === 'Home') {
            e.preventDefault();
            setSidebarWidth(200);
          } else if (e.key === 'End') {
            e.preventDefault();
            setSidebarWidth(480);
          }
        }}
        onPointerDown={onSidebarResizeStart}
        onPointerMove={onSidebarResizeMove}
        onPointerUp={onSidebarResizeEnd}
        onPointerCancel={onSidebarResizeEnd}
      />

      <div className="flex-1 flex flex-col min-w-0">
        <div className="flex-1 overflow-y-auto">
          {error && (
            <div className="text-xs text-destructive mx-5 mt-3 px-3 py-2 rounded-md bg-destructive/10">
              {error}
            </div>
          )}
          {previewDocId == null ? (
            <PreviewEmptyHint />
          ) : (
            <DocumentPreview
              key={previewDocId}
              docId={previewDocId}
              onClose={closePreview}
              onDelete={() => void onDeletePreview()}
              folderPath={previewFolderPath}
            />
          )}
        </div>
      </div>

      <FolderRenameDialog
        folderId={renameTarget}
        onClose={() => setRenameTarget(null)}
        onSubmit={onRenameFolder}
      />
      <ConfirmDialog
        open={deleteFolderTarget !== null}
        onOpenChange={(o) => !o && setDeleteFolderTarget(null)}
        title={`删除目录「${deleteFolderTarget?.name ?? ''}」?`}
        description={
          deleteFolderTarget && (
            <span>
              将递归删除该目录下
              {deleteFolderTarget.folderCount > 0 && (
                <>
                  {' '}
                  {deleteFolderTarget.folderCount} 个子目录
                </>
              )}
              {deleteFolderTarget.docCount > 0 && (
                <>
                  {' '}和 {deleteFolderTarget.docCount} 个文档
                </>
              )}
              ,操作不可撤销。
            </span>
          )
        }
        confirmLabel="删除"
        variant="destructive"
        onConfirm={confirmDeleteFolder}
      />
      <ConfirmDialog
        open={archiveDocTarget !== null}
        onOpenChange={(o) => !o && setArchiveDocTarget(null)}
        title={`归档文档「${archiveDocTarget?.title ?? ''}」?`}
        description="归档后文档不再出现在知识库,但 S3 对象仍保留;操作不可撤销。"
        confirmLabel="归档"
        variant="destructive"
        onConfirm={confirmArchiveDoc}
      />
    </div>
  );
}

function DocumentPreview({
  docId,
  onClose,
  onDelete,
  folderPath,
}: {
  docId: number;
  onClose: () => void;
  onDelete: () => void | Promise<void>;
  folderPath: string[] | null;
}) {
  const [doc, setDoc] = useState<DocumentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // 上层用 key={previewDocId} 触发重挂载,doc 切换时 loading/error/doc 都是初始态,无需在 effect 里同步重置。
  useEffect(() => {
    let cancelled = false;
    documentUseCases.get
      .execute(docId)
      .then((d) => {
        if (!cancelled) setDoc(d);
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [docId]);

  const sourceLabel =
    doc?.source === 'note' ? '笔记' : doc?.source === 'knowledge' ? '知识' : doc?.source;

  return (
    <div className="flex flex-col h-full">
      <header className="px-5 py-3 border-b flex items-center gap-2 min-w-0">
        <FileText className="h-4 w-4 text-muted-foreground shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium truncate">{doc?.title}</div>
          <div className="text-xs text-muted-foreground mt-0.5 flex items-center gap-1 min-w-0">
            <span>{sourceLabel}</span>
            <span className="opacity-50">·</span>
            <span>{doc?.content_type}</span>
            <span className="opacity-50">·</span>
            <BreadcrumbPath path={folderPath} />
          </div>
        </div>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
          onClick={() => void onDelete()}
          aria-label="归档"
          title="归档"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0"
          onClick={onClose}
          aria-label="关闭预览"
          title="关闭预览"
        >
          <X className="h-3.5 w-3.5" />
        </Button>
      </header>

      <PreviewBody docId={docId} doc={doc} loading={loading} error={error} source={doc?.source} />
    </div>
  );
}

// 知识库面包屑:home / 父目录1 / 父目录2,空时降级显示 home。
function BreadcrumbPath({ path }: { path: string[] | null }) {
  const segments = path && path.length > 0 ? path : ['home'];
  return (
    <span className="flex items-center gap-1 min-w-0 truncate">
      {segments.map((seg, i) => (
        <span key={i} className="flex items-center gap-1 min-w-0">
          {i > 0 && <span className="opacity-50">/</span>}
          <span className="truncate">{seg}</span>
        </span>
      ))}
    </span>
  );
}

function PreviewEmptyHint() {
  return (
    <div className="py-16 text-center">
      <div className="inline-flex flex-col items-center gap-3 text-muted-foreground max-w-xs">
        <div className="h-12 w-12 rounded-full bg-primary/10 text-primary grid place-items-center">
          <NotebookPen className="h-5 w-5" />
        </div>
        <div className="text-sm font-medium text-foreground">选择一个文档预览</div>
        <div className="text-xs leading-relaxed">
          从左侧目录树中点文档,这里会展示正文;
          也可直接拖到目录树上完成移动,或在目录的「+」里新建/上传。
        </div>
      </div>
    </div>
  );
}