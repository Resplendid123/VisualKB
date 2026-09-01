// 流式接口(AsyncIterable<SSERecord>);EventSource 自动维护 Last-Event-ID + auto-reconnect。
import type { StreamEvent } from '@/domain/entities/streamEvent';

// 一条 SSE 记录:typed 事件 + 后端给的 stream id(EventSource 自动透传给 Last-Event-ID)。
export interface SSERecord {
  ev: StreamEvent | null;
  id: string;
}

export interface ChatRepository {
  streamMessage(
    input: { content: string; conversationId: string; documentIds?: number[]; edit?: boolean },
    signal?: AbortSignal,
  ): AsyncIterable<SSERecord>;
  // 续传/重连:Last-Event-ID 由 EventSource 维护;lastTurn/lastSeqID 是 list 末尾 msg 元组。
  replayEvents(
    input: { conversationId: string; lastTurn?: number; lastSeqID?: number },
    signal?: AbortSignal,
  ): AsyncIterable<SSERecord>;
}