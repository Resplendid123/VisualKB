import type { DocumentRepository } from '@/domain/repositories/documentRepository';

export class UpdateDocumentUseCase {
  constructor(private readonly docRepo: DocumentRepository) {}

  // 整段覆盖内容;后端 whole_replace 拒绝空字符串。
  execute(id: number, content: string, title?: string) {
    return this.docRepo.patch(
      id,
      [{ type: 'whole_replace', args: { content } }],
      title
    );
  }
}