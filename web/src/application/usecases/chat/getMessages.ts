import type { ConversationRepository } from '@/domain/repositories/conversationRepository';
import type { Message } from '@/domain/entities/message';

export class GetMessagesUseCase {
  constructor(private readonly convoRepo: ConversationRepository) {}

  async execute(
    conversationId: string
  ): Promise<{
    items: Message[];
    lastTurnAtLoad: number;
    lastSeqIDAtLoad: number;
    inFlight: boolean;
  }> {
    return this.convoRepo.getMessages(conversationId);
  }
}