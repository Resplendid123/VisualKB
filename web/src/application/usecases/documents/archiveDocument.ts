import type { DocumentRepository } from '@/domain/repositories/documentRepository';

export class ArchiveDocumentUseCase {
  constructor(private readonly docRepo: DocumentRepository) {}

  execute(id: number): Promise<void> {
    return this.docRepo.archive(id);
  }
}