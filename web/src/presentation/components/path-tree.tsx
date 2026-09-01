'use client';

import {
  ChevronRight,
  FilePlus2,
  FileText,
  Folder,
  FolderOpen,
  FolderPlus,
  Home,
  MoreVertical,
  Plus,
} from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { cn } from '@/lib/utils';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import type { Document } from '@/domain/entities/document';
import type { TreeNode } from '@/domain/entities/tree';

export interface PathTreeProps {
  // 后端返回的扁平邻接表节点;组件按 parentId 构造层级。
  nodes: TreeNode[];
  // 文档列表;通过 TreeNode.docId 挂到 folder 下,展开时内联渲染 doc row。
  documents?: Document[];
  // 点击 doc row 时回调,通常用于打开预览。
  onOpenDoc?: (doc: Document) => void;
  // 当前选中的文档 id;命中时 doc row 高亮。目录永不接高亮。
  selectedDocId: number | null;
  // 已展开的目录 id 映射;父组件(viewStore)持有,跨视图切换保留。
  expanded: Record<number, boolean>;
  // 目录行点击时回调,父组件用其更新 viewStore 中的 expanded。
  onToggleFolder: (id: number, open: boolean) => void;
  // 拖拽中悬停的目标 tree node id;用于高亮。父组件维护。
  dropTargetNodeId?: number | null;
  // 父组件负责从 dataTransfer 读 docId 并调用 treeUseCases.moveNode。
  onDropDoc?: (docId: number, targetFolderId: number | null) => void;
  // folder 拖到另一个 folder 上:target=null 移到根级。
  onMoveFolder?: (sourceFolderId: number, targetFolderId: number | null) => void;
  // dragOver 时把当前 row 的 tree node id 报给父组件作为高亮目标。
  onSetDropTarget?: (id: number | null) => void;
  // "+" 菜单两项入口:新建文件夹/上传文件;parent=null = 根级。
  onCreateFolder?: (parent: number | null) => void;
  onUploadRequest?: (parent: number | null) => void;
  // 目录 row 右键菜单:重命名/删除空目录。
  onRenameRequest?: (folderId: number) => void;
  onDeleteRequest?: (folderId: number) => void;
  // VSCode 风格内联输入框:Enter 提交/Esc 取消;parentNodeId=null = home 下创建。
  draftFolder?: { parentNodeId: number | null; name: string } | null;
  onDraftFolderChange?: (name: string) => void;
  onDraftFolderCommit?: () => void;
  onDraftFolderCancel?: () => void;
}

interface UITreeNode {
  id: number | null; // null = 虚拟根
  name: string;
  isFolder: boolean;
  docId: number | null;
  count: number;
  docs: Document[];
  children: UITreeNode[];
  // 子树里 folder 最深偏移量(自身=0;无 folder 子=0;有 folder 子=1+...);拖拽时预估撞 MaxFolderDepth。
  subtreeMaxDepth: number;
}

// 与后端 document.MaxFolderDepth=3 对齐:folder 最多 3 层。
const MAX_FOLDER_DEPTH = 3;

// 把扁平邻接表 nodes 构造层级树:第一层 parentId=null,递归往下,只搭骨架不计数。
function buildTree(nodes: TreeNode[]): UITreeNode {
  const root: UITreeNode = {
    id: null,
    name: 'home',
    isFolder: true,
    docId: null,
    count: 0,
    docs: [],
    children: [],
    subtreeMaxDepth: 0,
  };
  const byParent = new Map<number | null, UITreeNode[]>();
  for (const n of nodes) {
    if (!n.isFolder) continue; // doc 指针在 attachDocs 阶段挂
    const uiNode: UITreeNode = {
      id: n.id,
      name: n.name,
      isFolder: true,
      docId: null,
      count: 0,
      docs: [],
      children: [],
      subtreeMaxDepth: 0,
    };
    const arr = byParent.get(n.parentId) ?? [];
    arr.push(uiNode);
    byParent.set(n.parentId, arr);
  }
  const wire = (parent: UITreeNode) => {
    parent.children = (byParent.get(parent.id) ?? []).sort((a, b) =>
      a.name.localeCompare(b.name)
    );
    for (const c of parent.children) wire(c);
  };
  wire(root);
  return root;
}

// 沿 folder 子树递归算最深层级偏移:叶子=0,有 folder 子=1+子偏移;拖拽前预估避免撞 MaxFolderDepth。
function computeSubtreeMaxDepth(node: UITreeNode): number {
  if (node.children.length === 0) {
    node.subtreeMaxDepth = 0;
    return 0;
  }
  let max = 0;
  for (const c of node.children) {
    max = Math.max(max, computeSubtreeMaxDepth(c) + 1);
  }
  node.subtreeMaxDepth = max;
  return max;
}

// 把每个 doc-pointer 按 docId 挂到对应 folder 下;root.docs 收 tree 不存在的孤儿(backfill 后空集)。
function attachDocs(root: UITreeNode, docs: Document[], nodes: TreeNode[]) {
  const docById = new Map<number, Document>();
  for (const d of docs) docById.set(d.id, d);
  const uiById = new Map<number, UITreeNode>();
  const collect = (n: UITreeNode) => {
    if (n.id != null) uiById.set(n.id, n);
    for (const c of n.children) collect(c);
  };
  collect(root);
  for (const tn of nodes) {
    if (tn.isFolder) continue;
    if (tn.docId == null) continue;
    const doc = docById.get(tn.docId);
    if (!doc) continue;
    const parent = tn.parentId == null ? root : uiById.get(tn.parentId);
    if (!parent) continue;
    parent.docs.push(doc);
  }
}

// 文件计数:不算目录本身,只算 docs.length + 后代 docs 累加。
function computeFileCounts(node: UITreeNode): number {
  let n = node.docs.length;
  for (const c of node.children) {
    n += computeFileCounts(c);
  }
  node.count = n;
  return n;
}

// 知识库侧栏目录树;只高亮 doc 行,目录行负责展开 + 拖放。
export function PathTree({
  nodes,
  documents,
  onOpenDoc,
  selectedDocId,
  expanded,
  onToggleFolder,
  dropTargetNodeId = null,
  onDropDoc,
  onMoveFolder,
  onSetDropTarget,
  onCreateFolder,
  onUploadRequest,
  onRenameRequest,
  onDeleteRequest,
  draftFolder,
  onDraftFolderChange,
  onDraftFolderCommit,
  onDraftFolderCancel,
}: PathTreeProps) {
  const [query, setQuery] = useState('');

  const tree = useMemo(() => {
    const t = buildTree(nodes);
    if (documents) attachDocs(t, documents, nodes);
    computeFileCounts(t);
    computeSubtreeMaxDepth(t);
    return t;
  }, [nodes, documents]);

  const q = query.trim().toLowerCase();
  const visible = q ? filterTree(tree, q) : tree;
  const rowMenu = !!(onRenameRequest || onDeleteRequest);

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="px-3 py-2 border-b flex items-center gap-1.5">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索目录…"
          className="h-7 text-xs bg-muted/40 border-transparent focus-visible:bg-background"
        />
      </div>
      <div className="flex-1 overflow-y-auto py-1">
        <TreeRow
          node={{
            id: null,
            name: 'home',
            isFolder: true,
            docId: null,
            count: tree.count,
            docs: visible.docs,
            children: visible.children,
            subtreeMaxDepth: 0,
          }}
          depth={0}
          expanded={expanded}
          onToggleFolder={onToggleFolder}
          alwaysOpen
          IconClosed={Home}
          IconOpen={Home}
          selectedDocId={selectedDocId}
          onOpenDoc={onOpenDoc}
          dropTargetNodeId={dropTargetNodeId}
          onDropDoc={onDropDoc}
          onMoveFolder={onMoveFolder}
          onSetDropTarget={onSetDropTarget}
          onRenameRequest={onRenameRequest}
          onDeleteRequest={onDeleteRequest}
          isRowMenuShown={rowMenu}
          draftFolder={draftFolder}
          onDraftFolderChange={onDraftFolderChange}
          onDraftFolderCommit={onDraftFolderCommit}
          onDraftFolderCancel={onDraftFolderCancel}
          onCreateFolder={onCreateFolder}
          onUploadRequest={onUploadRequest}
        />
        {q && visible.children.length === 0 && visible.docs.length === 0 && (
          <div className="px-3 py-6 text-center text-xs text-muted-foreground">
            没有匹配的目录
          </div>
        )}
      </div>
    </div>
  );
}

function filterTree(node: UITreeNode, q: string): UITreeNode {
  const filteredChildren = node.children
    .map((c) => filterTree(c, q))
    .filter((c) => c.name.toLowerCase().includes(q) || c.children.length > 0);
  // 保留:本节点名匹配 / 是根 / 子树有命中 / 自己的 docs 里有命中。
  // 之前只看 name+children,会吞掉"doc 名匹配但藏在名字不匹配的文件夹里"的情况。
  const selfDocsMatch = node.docs.some((d) => d.title.toLowerCase().includes(q));
  const keep =
    node.id === null ||
    node.name.toLowerCase().includes(q) ||
    filteredChildren.length > 0 ||
    selfDocsMatch;
  const docs = keep ? node.docs.filter((d) => d.title.toLowerCase().includes(q)) : [];
  return {
    id: node.id,
    name: node.name,
    isFolder: node.isFolder,
    docId: node.docId,
    count: node.count,
    docs,
    children: filteredChildren,
    subtreeMaxDepth: node.subtreeMaxDepth,
  };
}

interface TreeRowProps {
  node: UITreeNode;
  depth: number;
  expanded: Record<number, boolean>;
  onToggleFolder: (id: number, open: boolean) => void;
  alwaysOpen?: boolean;
  IconClosed?: typeof Folder;
  IconOpen?: typeof FolderOpen;
  selectedDocId: number | null;
  onOpenDoc?: (doc: Document) => void;
  dropTargetNodeId: number | null;
  onDropDoc?: (docId: number, targetFolderId: number | null) => void;
  onMoveFolder?: (sourceFolderId: number, targetFolderId: number | null) => void;
  onSetDropTarget?: (id: number | null) => void;
  onRenameRequest?: (folderId: number) => void;
  onDeleteRequest?: (folderId: number) => void;
  isRowMenuShown: boolean;
  draftFolder?: { parentNodeId: number | null; name: string } | null;
  onDraftFolderChange?: (name: string) => void;
  onDraftFolderCommit?: () => void;
  onDraftFolderCancel?: () => void;
  onCreateFolder?: (parent: number | null) => void;
  onUploadRequest?: (parent: number | null) => void;
}

function TreeRow({
  node,
  depth,
  expanded,
  onToggleFolder,
  alwaysOpen,
  IconClosed = Folder,
  IconOpen = FolderOpen,
  selectedDocId,
  onOpenDoc,
  dropTargetNodeId,
  onDropDoc,
  onMoveFolder,
  onSetDropTarget,
  onRenameRequest,
  onDeleteRequest,
  isRowMenuShown,
  draftFolder,
  onDraftFolderChange,
  onDraftFolderCommit,
  onDraftFolderCancel,
  onCreateFolder,
  onUploadRequest,
}: TreeRowProps) {
  const hasChildren = node.children.length > 0;
  const hasDocs = node.docs.length > 0;
  const canExpand = hasChildren || hasDocs;
  const isOpen = alwaysOpen || (node.id != null && !!expanded[node.id]);
  const accepts = onDropDoc !== undefined || onMoveFolder !== undefined;
  // node.id === null 时 dropTargetNodeId 也用 null 表示"无目标",会撞 id 所以显式排除 home(根仍允许 drop,只是不画 ring)。
  const isDropTarget =
    node.id !== null && dropTargetNodeId === node.id && accepts;
  const isRoot = node.id === null;
  const canEditThis = isRowMenuShown && !isRoot;
  const canAddHere = !!(onCreateFolder || onUploadRequest);
  // folder 本身可拖(拖到另一个 folder 上 = moveNode 到那个 folder 下);home 代表"所有 knowledge",不可拖。
  const canDragThis = !isRoot && !!onMoveFolder;
  // 最深层级不允许再新建 folder,但仍能上传文件 — 与后端 MaxFolderDepth 对齐。
  const canCreateFolderHere = !!onCreateFolder && depth < MAX_FOLDER_DEPTH;

  return (
    <div className="group/row">
      <div className="flex items-center gap-0.5 relative">
        <button
          type="button"
          draggable={canDragThis}
          onClick={() => {
            if (alwaysOpen || node.id == null) return;
            if (canExpand) onToggleFolder(node.id, !isOpen);
          }}
          onDragStart={(e) => {
            if (!canDragThis || node.id == null) return;
            e.dataTransfer.effectAllowed = 'move';
            e.dataTransfer.setData('application/x-folder-path', String(node.id));
            // 把子树深度透传,目标 drop 时预估整体深度,避免撞 MaxFolderDepth。
            e.dataTransfer.setData(
              'application/x-folder-subtree-depth',
              String(node.subtreeMaxDepth)
            );
          }}
          onDragEnd={() => onSetDropTarget?.(null)}
          onDragOver={(e) => {
            if (!accepts) return;
            e.preventDefault();
            e.dataTransfer.dropEffect = 'move';
            onSetDropTarget?.(node.id);
          }}
          // dragLeave 故意不清理:DocRow 与 button 是兄弟,鼠标移到同 folder 内 DocRow 会误清高亮;统一由源侧 dragEnd 清。
          onDrop={(e) => {
            if (!accepts) return;
            e.preventDefault();
            onSetDropTarget?.(null);
            const folderSrc = e.dataTransfer.getData('application/x-folder-path');
            if (folderSrc) {
              const srcId = Number(folderSrc);
              if (!Number.isFinite(srcId) || srcId <= 0) return;
              if (srcId === node.id) return; // drop 到自身,后端 service 会拒
              // 深度预估:target depth + 1(自身) + 源 subtree 偏移 > MAX_FOLDER_DEPTH 就拒。
              const sd = Number(
                e.dataTransfer.getData('application/x-folder-subtree-depth')
              );
              const srcSubtree = Number.isFinite(sd) ? sd : 0;
              if (depth + 1 + srcSubtree > MAX_FOLDER_DEPTH) {
                onSetDropTarget?.(null);
                window.alert('目录嵌套最多 3 层,无法移到此位置');
                return;
              }
              onMoveFolder?.(srcId, node.id);
              return;
            }
            if (!onDropDoc) return;
            const raw = e.dataTransfer.getData('text/plain');
            const id = Number(raw);
            if (!Number.isFinite(id) || id <= 0) return;
            onDropDoc(id, node.id);
          }}
          className={cn(
            'flex-1 min-w-0 flex items-center gap-1 px-2 py-1 text-xs rounded-md transition-colors text-left',
            'focus:outline-none focus-visible:outline-none',
            isDropTarget &&
              'ring-2 ring-primary bg-primary/25 text-foreground shadow-sm',
            !isDropTarget &&
              'text-muted-foreground hover:bg-muted hover:text-foreground'
          )}
          style={{ paddingLeft: 8 + depth * 12 }}
        >
          {canExpand && !alwaysOpen ? (
            <ChevronRight
              className={cn(
                'h-3 w-3 shrink-0 transition-transform',
                isOpen && 'rotate-90'
              )}
            />
          ) : (
            <span className="w-3 shrink-0" />
          )}
          {isOpen ? (
            <IconOpen className="h-3.5 w-3.5 shrink-0" />
          ) : (
            <IconClosed className="h-3.5 w-3.5 shrink-0" />
          )}
          <span className="truncate flex-1">{node.name}</span>
          <span className="text-[10px] text-muted-foreground/70 tabular-nums">
            {node.count}
          </span>
        </button>
        {canAddHere && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <button
                  type="button"
                  aria-label="新建到此目录"
                  title="新建到此目录"
                  className={cn(
                    'h-5 w-5 rounded-sm grid place-items-center text-muted-foreground hover:text-foreground focus-visible:opacity-100 transition-opacity',
                    isRoot ? 'opacity-100' : 'opacity-0 group-hover/row:opacity-100'
                  )}
                >
                  <Plus className="h-3 w-3" />
                </button>
              }
            />
            <DropdownMenuContent align="end" sideOffset={2}>
              {canCreateFolderHere && (
                <DropdownMenuItem
                  onClick={() => onCreateFolder(node.id)}
                >
                  <FolderPlus className="h-3.5 w-3.5 mr-1.5" />
                  新建文件夹
                </DropdownMenuItem>
              )}
              {onUploadRequest && (
                <DropdownMenuItem
                  onClick={() => onUploadRequest(node.id)}
                >
                  <FilePlus2 className="h-3.5 w-3.5 mr-1.5" />
                  上传文件
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
        {canEditThis && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <button
                  type="button"
                  aria-label="目录操作"
                  className="h-5 w-5 rounded-sm grid place-items-center text-muted-foreground opacity-0 group-hover/row:opacity-100 hover:text-foreground focus-visible:opacity-100 transition-opacity"
                >
                  <MoreVertical className="h-3 w-3" />
                </button>
              }
            />
            <DropdownMenuContent align="end" sideOffset={2}>
              {onRenameRequest && node.id != null && (
                <DropdownMenuItem onClick={() => onRenameRequest(node.id!)}>
                  重命名
                </DropdownMenuItem>
              )}
              {onDeleteRequest && node.id != null && (
                <DropdownMenuItem
                  variant="destructive"
                  onClick={() => onDeleteRequest(node.id!)}
                >
                  删除目录
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
      {isOpen && (
        <>
          {node.children.map((child) => (
            <TreeRow
              key={child.id ?? 'root'}
              node={child}
              depth={depth + 1}
              expanded={expanded}
              onToggleFolder={onToggleFolder}
              selectedDocId={selectedDocId}
              onOpenDoc={onOpenDoc}
              dropTargetNodeId={dropTargetNodeId}
              onDropDoc={onDropDoc}
              onMoveFolder={onMoveFolder}
              onSetDropTarget={onSetDropTarget}
              onRenameRequest={onRenameRequest}
              onDeleteRequest={onDeleteRequest}
              isRowMenuShown={isRowMenuShown}
              draftFolder={draftFolder}
              onDraftFolderChange={onDraftFolderChange}
              onDraftFolderCommit={onDraftFolderCommit}
              onDraftFolderCancel={onDraftFolderCancel}
              onCreateFolder={onCreateFolder}
              onUploadRequest={onUploadRequest}
            />
          ))}
          {node.docs.map((doc) => (
            <DocRow
              key={`doc-${doc.id}`}
              depth={depth + 1}
              doc={doc}
              isSelected={doc.id === selectedDocId}
              onOpen={onOpenDoc}
              onSetDropTarget={onSetDropTarget}
              parentFolderId={node.id}
              onDropDoc={onDropDoc}
              onMoveFolder={onMoveFolder}
            />
          ))}
          {draftFolder && draftFolder.parentNodeId === node.id && (
            <DraftInputRow
              depth={depth + 1}
              value={draftFolder.name}
              onChange={onDraftFolderChange}
              onCommit={onDraftFolderCommit}
              onCancel={onDraftFolderCancel}
            />
          )}
        </>
      )}
    </div>
  );
}

// 与后端 document.ValidateName 对齐:1-64 字符(Unicode,允许中文),排除 ASCII 控制字符 / 换行。
const SEGMENT_MAX = 64;
const SEGMENT_RE = /^[^\x00-\x1F\x7F]+$/;

// 目录展开时内联的 doc 行:既是 source(可拖出)也是 target(drop 落到所在 folder)。
function DocRow({
  depth,
  doc,
  isSelected,
  onOpen,
  onSetDropTarget,
  parentFolderId,
  onDropDoc,
  onMoveFolder,
}: {
  depth: number;
  doc: Document;
  isSelected: boolean;
  onOpen?: (doc: Document) => void;
  onSetDropTarget?: (id: number | null) => void;
  parentFolderId: number | null;
  onDropDoc?: (docId: number, targetFolderId: number | null) => void;
  onMoveFolder?: (
    sourceFolderId: number,
    targetFolderId: number | null
  ) => void;
}) {
  return (
    <button
      type="button"
      draggable
      onClick={() => onOpen?.(doc)}
      onDragStart={(e) => {
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', String(doc.id));
        // 别把高亮也清掉 —— 会和 dragover 设的互相打架。
      }}
      onDragOver={(e) => {
        // 接 doc drop 与 folder drop 两种;target 始终是所在 folder。
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        onSetDropTarget?.(parentFolderId);
      }}
      onDragEnd={() => onSetDropTarget?.(null)}
      onDrop={(e) => {
        e.preventDefault();
        e.stopPropagation(); // 不冒泡到外层 folder — 父 folder 已经在被我们代表。
        onSetDropTarget?.(null);
        const folderSrc = e.dataTransfer.getData('application/x-folder-path');
        if (folderSrc) {
          const srcId = Number(folderSrc);
          if (!Number.isFinite(srcId) || srcId <= 0) return;
          if (srcId === parentFolderId) return;
          // 深度预估:DocRow parent folder 实际 level = depth - 1(visual 渲染偏移);
          // 移过去后源 folder 落在 depth,其子树最深 = depth + srcSubtree。
          const sd = Number(
            e.dataTransfer.getData('application/x-folder-subtree-depth')
          );
          const srcSubtree = Number.isFinite(sd) ? sd : 0;
          if (depth + srcSubtree > MAX_FOLDER_DEPTH) {
            onSetDropTarget?.(null);
            window.alert('目录嵌套最多 3 层,无法移到此位置');
            return;
          }
          onMoveFolder?.(srcId, parentFolderId);
          return;
        }
        if (!onDropDoc) return;
        const raw = e.dataTransfer.getData('text/plain');
        const id = Number(raw);
        if (!Number.isFinite(id) || id <= 0) return;
        onDropDoc(id, parentFolderId);
      }}
      className={cn(
        'group/doc flex w-full items-center gap-1 px-2 py-1 text-xs rounded-md text-left transition-colors',
        isSelected
          ? 'bg-primary/10 text-foreground'
          : 'text-muted-foreground/80 hover:bg-muted hover:text-foreground'
      )}
      style={{ paddingLeft: 8 + depth * 12 }}
    >
      <span className="w-3 shrink-0" />
      <FileText className="h-3.5 w-3.5 shrink-0" />
      <span className="truncate flex-1">{doc.title}</span>
    </button>
  );
}

// VSCode 风格内联输入框:实时校验,Enter 提交,Esc 取消;无效命名不提交,保留输入+红框提示。
function DraftInputRow({
  depth,
  value,
  onChange,
  onCommit,
  onCancel,
}: {
  depth: number;
  value: string;
  onChange?: (v: string) => void;
  onCommit?: () => void;
  onCancel?: () => void;
}) {
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => {
    ref.current?.focus();
  }, []);

  const trimmed = value.trim();
  const hasContent = trimmed.length > 0;
  const tooLong = [...trimmed].length > SEGMENT_MAX;
  const isValid = hasContent && !tooLong && SEGMENT_RE.test(trimmed);

  const commit = () => {
    if (!isValid) return; // 无效就不提交,保留输入让你修
    onCommit?.();
  };

  return (
    <div className="flex flex-col gap-0.5" style={{ paddingLeft: 8 + depth * 12 }}>
      <div className="flex items-center gap-1">
        <span className="w-3 shrink-0" />
        <FolderPlus
          className={cn(
            'h-3.5 w-3.5 shrink-0',
            isValid || !hasContent ? 'text-muted-foreground' : 'text-destructive'
          )}
        />
        <input
          ref={ref}
          value={value}
          onChange={(e) => onChange?.(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              commit();
            } else if (e.key === 'Escape') {
              e.preventDefault();
              onCancel?.();
            }
          }}
          // 不在 blur 上无条件 commit:点 "+" 看 dropdown 时失焦会误触,Enter 显式提交更稳。
          placeholder="新文件夹名"
          className={cn(
            'flex-1 min-w-0 h-6 px-1.5 text-xs rounded-sm border bg-background focus:outline-none focus:ring-1',
            isValid || !hasContent
              ? 'border-primary/40 focus:ring-primary'
              : 'border-destructive focus:ring-destructive'
          )}
        />
      </div>
      {hasContent && !isValid && (
        <div className="text-[10px] text-destructive pl-4">
          {tooLong ? `名字过长(最多 ${SEGMENT_MAX} 个字符)` : '名字不能含控制字符或换行'}
        </div>
      )}
    </div>
  );
}