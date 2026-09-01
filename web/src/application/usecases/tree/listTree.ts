import type { TreeRepository } from '@/domain/repositories/treeRepository';
import type { TreeListResult } from '@/domain/entities/tree';

export class ListTreeUseCase {
  constructor(private readonly treeRepo: TreeRepository) {}
  execute(): Promise<TreeListResult> {
    return this.treeRepo.list();
  }
}