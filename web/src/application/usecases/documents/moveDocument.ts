import type { DocumentRepository } from '@/domain/repositories/documentRepository';

export class MoveDocumentUseCase {
  constructor(private readonly docRepo: DocumentRepository) {}
  execute(id: number, parentTreeId: number | null): Promise<void> {
    return this.docRepo.move(id, { parent_tree_id: parentTreeId });
  }
}