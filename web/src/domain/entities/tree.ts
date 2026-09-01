// 邻接表知识库树节点:isFolder=true 目录(name 必填);false 文档指针(docId → documents.id)。
export interface TreeNode {
  id: number;
  userId: number;
  parentId: number | null;
  name: string;
  isFolder: boolean;
  docId: number | null;
  createdAt: string;
}

export interface TreeListResult {
  nodes: TreeNode[];
}

export interface TreeNodeIdResult {
  id: number;
  parentId: number | null;
}