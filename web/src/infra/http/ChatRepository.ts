import type { ChatRepository, SSERecord } from '@/domain/repositories/chatRepository';
import type { StreamEvent } from '@/domain/entities/streamEvent';
import { API_BASE } from './AuthClient';

// 新 turn (?content=...) 和续传共用 GET /events,靠 EventSource 做 auto-reconnect + auto-Last-Event-ID。
export class HttpChatRepository implements ChatRepository {
  async *streamMessage(
    input: { content: string; conversationId: string; documentIds?: number[]; edit?: boolean },
    signal?: AbortSignal
  ): AsyncIterable<SSERecord> {
    const params = new URLSearchParams();
    params.set('content', input.content);
    if (input.documentIds && input.documentIds.length > 0) {
      params.set('document_ids', input.documentIds.join(','));
    }
    params.set('edit', input.edit ? 'true' : 'false');
    yield* openEventSource(input.conversationId, params, signal);
  }

  // 续传:Last-Event-ID 由 EventSource 浏览器自动维护。
  async *replayEvents(
    input: { conversationId: string; lastTurn?: number; lastSeqID?: number },
    signal?: AbortSignal
  ): AsyncIterable<SSERecord> {
    const params = new URLSearchParams();
    if (input.lastTurn && input.lastTurn > 0) params.set('last_turn', String(input.lastTurn));
    if (input.lastSeqID && input.lastSeqID > 0)
      params.set('last_seq_id', String(input.lastSeqID));
    yield* openEventSource(input.conversationId, params, signal);
  }
}

// 一条 SSE 记录在 domain/repositories/chatRepository.ts 定义。

// 与后端 conversation_handler.Events 的 SSE write 一一对应。
const EVENT_TYPES = [
  'text',
  'tool_call',
  'tool_result',
  'question',
  'error',
  'done',
] as const;

// openEventSource 把 EventSource 包成 AsyncIterable:signal.abort/done/4xx → close。
async function* openEventSource(
  conversationId: string,
  params: URLSearchParams,
  signal: AbortSignal | undefined
): AsyncIterable<SSERecord> {
  const url = `${API_BASE}/api/v1/conversations/${conversationId}/events?${params.toString()}`;
  const es = new EventSource(url, { withCredentials: true });

  const queue: SSERecord[] = [];
  const waiters: Array<(v: IteratorResult<SSERecord>) => void> = [];
  let closed = false;

  const dispatch = (rec: SSERecord) => {
    const w = waiters.shift();
    if (w) w({ value: rec, done: false });
    else queue.push(rec);
  };

  const finish = () => {
    if (closed) return;
    closed = true;
    while (waiters.length) {
      waiters.shift()!({ value: undefined as unknown as SSERecord, done: true });
    }
  };

  const onAbort = () => {
    es.close();
    finish();
  };
  if (signal) {
    if (signal.aborted) {
      es.close();
      finish();
    } else {
      signal.addEventListener('abort', onAbort);
    }
  }

  for (const t of EVENT_TYPES) {
    es.addEventListener(t, (e: Event) => {
      const me = e as MessageEvent;
      const ev = mapEvent(t, me.data);
      if (!ev) return;
      dispatch({ ev, id: me.lastEventId ?? '' });
    });
  }
  // heartbeat 是 SSE 注释行,EventSource 不触发 message — 不用挂。
  es.onerror = () => {
    // CONNECTING 时浏览器自动重连,只有 CLOSED 才彻底结束。
    if (es.readyState === EventSource.CLOSED) finish();
  };

  try {
    while (true) {
      if (queue.length) {
        const rec = queue.shift()!;
        yield rec;
        if (rec.ev?.type === 'done') {
          es.close();
          return;
        }
        continue;
      }
      if (closed) return;
      const next = await new Promise<IteratorResult<SSERecord>>((resolve) =>
        waiters.push(resolve)
      );
      if (next.done) return;
      yield next.value;
      if (next.value.ev?.type === 'done') {
        es.close();
        return;
      }
    }
  } finally {
    if (signal) signal.removeEventListener('abort', onAbort);
    es.close();
  }
}

// payload 字段缺失或类型不对时静默丢弃 — 流不该因为单条事件挂掉。
function mapEvent(event: string, data: string): StreamEvent | null {
  let payload: Record<string, unknown>;
  try {
    const parsed: unknown = JSON.parse(data);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null;
    payload = parsed as Record<string, unknown>;
  } catch {
    return null;
  }

  switch (event) {
    case 'text': {
      if (typeof payload.delta !== 'string') return null;
      return { type: 'text', delta: payload.delta };
    }
    case 'tool_call': {
      if (typeof payload.tool_call_id !== 'string' || typeof payload.name !== 'string')
        return null;
      return {
        type: 'tool_call',
        toolCallId: payload.tool_call_id,
        name: payload.name,
        args: (payload.args as Record<string, unknown>) ?? {},
        description: typeof payload.description === 'string' ? payload.description : '',
      };
    }
    case 'tool_result': {
      if (typeof payload.tool_call_id !== 'string') return null;
      return {
        type: 'tool_result',
        toolCallId: payload.tool_call_id,
        name: typeof payload.name === 'string' ? payload.name : '',
        result: payload.result,
        error: typeof payload.error === 'string' ? payload.error : undefined,
      };
    }
    case 'question':
      if (
        typeof payload.tool_call_id === 'string' &&
        typeof payload.question === 'string' &&
        Array.isArray(payload.options)
      ) {
        return {
          type: 'question',
          toolCallId: payload.tool_call_id,
          question: payload.question,
          options: payload.options,
        };
      }
      return null;
    case 'error':
      if (typeof payload.message === 'string')
        return {
          type: 'error',
          code: typeof payload.code === 'string' ? payload.code : 'unknown',
          message: payload.message,
        };
      return null;
    case 'done':
      return {
        type: 'done',
        messageId:
          typeof payload.message_id === 'string' ? payload.message_id : undefined,
      };
    default:
      return null;
  }
}