import type { TreeRepository } from '@/domain/repositories/treeRepository';
import type { TreeNodeIdResult } from '@/domain/entities/tree';

export class CreateFolderUseCase {
  constructor(private readonly treeRepo: TreeRepository) {}
  execute(parentId: number | null, name: string): Promise<TreeNodeIdResult> {
    return this.treeRepo.createFolder(parentId, name);
  }
}