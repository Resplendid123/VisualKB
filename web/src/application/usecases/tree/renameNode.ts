import type { TreeRepository } from '@/domain/repositories/treeRepository';

export class RenameNodeUseCase {
  constructor(private readonly treeRepo: TreeRepository) {}
  execute(id: number, name: string): Promise<void> {
    return this.treeRepo.renameNode(id, name);
  }
}