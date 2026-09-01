import { expectNoContent, parseEnvelope } from './AuthClient';
import type { AuthedFetch } from './AuthClient';
import type { TreeRepository } from '@/domain/repositories/treeRepository';
import type {
  TreeListResult,
  TreeNode,
  TreeNodeIdResult,
} from '@/domain/entities/tree';

interface RawTreeNode {
  id: number;
  user_id: number;
  parent_id: number | null;
  name: string;
  is_folder: boolean;
  doc_id: number | null;
  created_at: string;
}

interface RawTreeListResponse {
  nodes: RawTreeNode[];
}

interface TreeNodeIdResponse {
  id: number;
  parent_id: number | null;
}

export class HttpTreeRepository implements TreeRepository {
  constructor(public readonly authedFetch: AuthedFetch) {}

  async list(): Promise<TreeListResult> {
    const res = await this.authedFetch('/api/v1/tree');
    const data = await parseEnvelope<RawTreeListResponse>(res, 'list tree');
    const nodes: TreeNode[] = (data.nodes ?? []).map((n) => ({
      id: n.id,
      userId: n.user_id,
      parentId: n.parent_id,
      name: n.name,
      isFolder: n.is_folder,
      docId: n.doc_id,
      createdAt: n.created_at,
    }));
    return { nodes };
  }

  async createFolder(parentId: number | null, name: string): Promise<TreeNodeIdResult> {
    const res = await this.authedFetch('/api/v1/tree/folder', {
      method: 'POST',
      body: JSON.stringify({ parent_id: parentId, name }),
    });
    const data = await parseEnvelope<TreeNodeIdResponse>(res, 'create folder');
    return { id: data.id, parentId: data.parent_id };
  }

  async renameNode(id: number, name: string): Promise<void> {
    const res = await this.authedFetch(`/api/v1/tree/folder/${id}/rename`, {
      method: 'POST',
      body: JSON.stringify({ name }),
    });
    await expectNoContent(res, 'rename node');
  }

  async moveNode(id: number, parentId: number | null): Promise<TreeNodeIdResult> {
    const res = await this.authedFetch(`/api/v1/tree/folder/${id}/move`, {
      method: 'POST',
      body: JSON.stringify({ parent_id: parentId }),
    });
    const data = await parseEnvelope<TreeNodeIdResponse>(res, 'move node');
    return { id: data.id, parentId: data.parent_id };
  }

  async deleteFolder(id: number): Promise<void> {
    const res = await this.authedFetch(`/api/v1/tree/folder/${id}/delete`, {
      method: 'POST',
    });
    await expectNoContent(res, 'delete folder');
  }
}