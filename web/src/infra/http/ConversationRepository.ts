import { expectNoContent, parseEnvelope } from './AuthClient';
import type { AuthedFetch } from './AuthClient';
import type { ConversationRepository } from '@/domain/repositories/conversationRepository';
import type { Conversation } from '@/domain/entities/conversation';
import type { Message } from '@/domain/entities/message';
import type { ToolCallData } from '@/domain/entities/streamEvent';

interface ConversationPayload {
  id: string;
  title: string;
  active_project_id?: string | null;
}

interface ListResponse {
  items: ConversationPayload[];
  total: number;
}

interface GetMessagesResponse {
  items: MessagePayload[];
  last_turn_at_load: number;
  last_seq_id_at_load: number;
  in_flight: boolean;
}

interface MessagePayload {
  id: string;
  role: 'user' | 'assistant' | 'tool';
  content: string;
  turn_id: number;
  seq_id: number;
  created_at: string;
  tool_calls?: ToolCallData[];
  tool_call_id?: string;
}

export class HttpConversationRepository implements ConversationRepository {
  constructor(public readonly authedFetch: AuthedFetch) {}

  async list(
    input: { limit: number; offset: number }
  ): Promise<{ items: Conversation[]; total: number }> {
    const res = await this.authedFetch(
      `/api/v1/conversations?limit=${input.limit}&offset=${input.offset}`
    );
    const data = await parseEnvelope<ListResponse>(res, 'list conversations');
    return {
      items: data.items.map(toConversation),
      total: data.total,
    };
  }

  async create(input: { title: string }): Promise<Conversation> {
    const res = await this.authedFetch('/api/v1/conversations', {
      method: 'POST',
      body: JSON.stringify({ title: input.title }),
    });
    const data = await parseEnvelope<ConversationPayload>(res, 'create conversation');
    return toConversation(data);
  }

  async getMessages(
    conversationId: string
  ): Promise<{
    items: Message[];
    lastTurnAtLoad: number;
    lastSeqIDAtLoad: number;
    inFlight: boolean;
  }> {
    const res = await this.authedFetch(
      `/api/v1/conversations/${conversationId}/messages`
    );
    const data = await parseEnvelope<GetMessagesResponse>(res, 'get messages');
    return {
      items: data.items.map(toMessage),
      lastTurnAtLoad: data.last_turn_at_load ?? 0,
      lastSeqIDAtLoad: data.last_seq_id_at_load ?? 0,
      inFlight: data.in_flight === true,
    };
  }

  // 软删:后端标 archived_at,List 不再返回;成功 204。
  async archive(conversationId: string): Promise<void> {
    const res = await this.authedFetch(
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/archive`,
      { method: 'POST' }
    );
    await expectNoContent(res, 'archive conversation');
  }
}

function toConversation(p: ConversationPayload): Conversation {
  return {
    id: p.id,
    title: p.title,
    activeProjectId: p.active_project_id ?? undefined,
  };
}

function toMessage(p: MessagePayload): Message {
  return {
    id: p.id,
    role: p.role,
    content: p.content,
    turn_id: p.turn_id ?? 0,
    seq_id: p.seq_id ?? 0,
    created_at: Date.parse(p.created_at) || 0,
    tool_calls: p.tool_calls,
    tool_call_id: p.tool_call_id,
  };
}