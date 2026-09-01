import type { DocumentRepository, UploadDocumentInput } from '@/domain/repositories/documentRepository';
import type { CreateDocumentResult } from '@/domain/entities/document';

export class UploadDocumentUseCase {
  constructor(private readonly docRepo: DocumentRepository) {}

  execute(input: UploadDocumentInput): Promise<CreateDocumentResult> {
    return this.docRepo.upload(input);
  }
}