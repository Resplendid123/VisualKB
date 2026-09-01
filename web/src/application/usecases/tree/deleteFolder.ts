import type { TreeRepository } from '@/domain/repositories/treeRepository';

export class DeleteFolderUseCase {
  constructor(private readonly treeRepo: TreeRepository) {}
  execute(id: number): Promise<void> {
    return this.treeRepo.deleteFolder(id);
  }
}