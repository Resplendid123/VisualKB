import type { Conversation } from '@/domain/entities/conversation';
import type { Message } from '@/domain/entities/message';

// 会话 CRUD 端口:list/create/getMessages/archive;后续加 update 在这里扩展。
export interface ConversationRepository {
  list(input: { limit: number; offset: number }): Promise<{
    items: Conversation[];
    total: number;
  }>;
  create(input: { title: string }): Promise<Conversation>;
  getMessages(conversationId: string): Promise<{
    items: Message[];
    // list 末尾 msg 的 (turn_id, seq_id) 元组,作 Replay 续传锚点。
    lastTurnAtLoad: number;
    lastSeqIDAtLoad: number;
    inFlight: boolean;
  }>;
  archive(conversationId: string): Promise<void>;
}