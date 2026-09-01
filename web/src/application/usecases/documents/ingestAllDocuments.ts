import type { DocumentRepository } from '@/domain/repositories/documentRepository';

export class IngestAllDocumentsUseCase {
  constructor(private readonly docRepo: DocumentRepository) {}
  execute(source: 'note' | 'knowledge'): Promise<number> {
    return this.docRepo.ingestAll(source);
  }
}