import type {
  TreeListResult,
  TreeNode,
  TreeNodeIdResult,
} from '@/domain/entities/tree';

// 邻接表知识库树端口:与后端 /api/v1/tree 对齐。
export interface TreeRepository {
  list(): Promise<TreeListResult>;
  createFolder(parentId: number | null, name: string): Promise<TreeNodeIdResult>;
  renameNode(id: number, name: string): Promise<void>;
  moveNode(id: number, parentId: number | null): Promise<TreeNodeIdResult>;
  deleteFolder(id: number): Promise<void>;
}

// 从 (parentId, name) 找子节点。
export function findChildByName(
  nodes: TreeNode[],
  parentId: number | null,
  name: string
): TreeNode | undefined {
  return nodes.find((n) => n.parentId === parentId && n.name === name);
}

// 通过 docId 反查树节点。
export function findNodeByDocId(nodes: TreeNode[], docId: number): TreeNode | undefined {
  return nodes.find((n) => !n.isFolder && n.docId === docId);
}

// 从 docId 算出从根到直接父目录的祖先路径(包含「home」);doc 不存在返回空数组。
export function getAncestorNames(nodes: TreeNode[], docId: number): string[] {
  const docNode = nodes.find((n) => !n.isFolder && n.docId === docId);
  if (!docNode) return [];
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const path: string[] = ['home'];
  let cur = docNode.parentId;
  while (cur != null) {
    const node = byId.get(cur);
    if (!node) break;
    path.push(node.name);
    cur = node.parentId;
  }
  return path;
}