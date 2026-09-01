import type { ConversationRepository } from '@/domain/repositories/conversationRepository';

// 软删会话,后端标 archived_at,List 不再返回;调方负责把本地列表的这条剔除。
export class ArchiveConversationUseCase {
  constructor(private readonly convoRepo: ConversationRepository) {}

  async execute(conversationId: string): Promise<void> {
    return this.convoRepo.archive(conversationId);
  }
}