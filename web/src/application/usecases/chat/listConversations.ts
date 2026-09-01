import type { ConversationRepository } from '@/domain/repositories/conversationRepository';
import type { Conversation } from '@/domain/entities/conversation';

export class ListConversationsUseCase {
  constructor(private readonly convoRepo: ConversationRepository) {}

  async execute(input: { limit: number; offset: number }): Promise<{
    items: Conversation[];
    total: number;
  }> {
    return this.convoRepo.list(input);
  }
}