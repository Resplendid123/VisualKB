import type { TreeRepository } from '@/domain/repositories/treeRepository';
import type { TreeNodeIdResult } from '@/domain/entities/tree';

export class MoveNodeUseCase {
  constructor(private readonly treeRepo: TreeRepository) {}
  execute(id: number, parentId: number | null): Promise<TreeNodeIdResult> {
    return this.treeRepo.moveNode(id, parentId);
  }
}