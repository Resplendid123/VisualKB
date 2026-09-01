import type { DocumentRepository } from '@/domain/repositories/documentRepository';
import type {
  CreateDocumentInput,
  CreateDocumentResult,
} from '@/domain/entities/document';

export class CreateDocumentUseCase {
  constructor(private readonly docRepo: DocumentRepository) {}

  execute(input: CreateDocumentInput): Promise<CreateDocumentResult> {
    return this.docRepo.create(input);
  }
}