import type { DocumentRepository } from '@/domain/repositories/documentRepository';
import type {
  ListDocumentsInput,
  ListDocumentsResult,
} from '@/domain/entities/document';

export class ListDocumentsUseCase {
  constructor(private readonly docRepo: DocumentRepository) {}

  execute(input: ListDocumentsInput): Promise<ListDocumentsResult> {
    return this.docRepo.list(input);
  }
}
