import type { ConversationRepository } from '@/domain/repositories/conversationRepository';
import type { Conversation } from '@/domain/entities/conversation';
import { ErrInvalidName } from '@/domain/errors';

export class CreateConversationUseCase {
  constructor(private readonly convoRepo: ConversationRepository) {}

  async execute(input: { title: string }): Promise<Conversation> {
    const title = input.title.trim();
    if (!title) throw ErrInvalidName();
    return this.convoRepo.create({ title });
  }
}