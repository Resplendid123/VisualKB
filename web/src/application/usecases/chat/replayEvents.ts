import type { ChatRepository, SSERecord } from '@/domain/repositories/chatRepository';

// EventSource 自动维护 Last-Event-ID + auto-reconnect,断网浏览器自动重连,无需前端退避。
export class ReplayEventsUseCase {
  constructor(private readonly chatRepo: ChatRepository) {}

  execute(
    input: { conversationId: string; lastTurn?: number; lastSeqID?: number },
    signal?: AbortSignal
  ): AsyncIterable<SSERecord> {
    return this.chatRepo.replayEvents(input, signal);
  }
}
