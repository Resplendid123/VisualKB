import type { DocumentRepository } from '@/domain/repositories/documentRepository';
import type { DocumentDetail } from '@/domain/entities/document';

export class GetDocumentUseCase {
  constructor(private readonly docRepo: DocumentRepository) {}

  execute(id: number): Promise<DocumentDetail> {
    return this.docRepo.get(id);
  }
}