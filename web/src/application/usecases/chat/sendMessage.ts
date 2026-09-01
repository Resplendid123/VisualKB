import type { ChatRepository, SSERecord } from '@/domain/repositories/chatRepository';
import { ErrEmptyMessage } from '@/domain/errors';

// 发送一条消息流式获取回复,返回 AsyncIterable<SSERecord>;signal abort 后实现应尽快停止产出。
export class SendMessageUseCase {
  constructor(private readonly chatRepo: ChatRepository) {}

  execute(
    input: { content: string; conversationId: string; documentIds?: number[]; edit?: boolean },
    signal?: AbortSignal
  ): AsyncIterable<SSERecord> {
    const text = input.content.trim();
    if (!text) throw ErrEmptyMessage();
    return this.chatRepo.streamMessage(
      {
        content: text,
        conversationId: input.conversationId,
        documentIds: input.documentIds,
        edit: input.edit,
      },
      signal
    );
  }
}