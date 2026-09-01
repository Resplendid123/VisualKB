import type { AssistantEvent, ToolCallData } from './streamEvent';

export type MessageRole = 'user' | 'assistant' | 'tool' | 'system';

export interface Message {
  id: string;
  role: MessageRole;
  content: string;
  // 后端 turn_id(per-conv INCR,user msg 起一轮);Replay 续传锚点之一。
  turn_id: number;
  // 后端 seq_id(per-turn 顺序,user=1);与 turn_id 组成元组续传锚点。
  seq_id: number;
  created_at: number;
  // 流式累积的渲染事件;历史不带,useConversations 拉 DB 后用 rebuildMessageEvents 重算。
  events?: AssistantEvent[];
  // assistant 消息携带的 tool_calls(JSONB 反序列化);有就意味着后续接了若干 tool 消息。
  tool_calls?: ToolCallData[];
  // tool 消息挂回 assistant tool_call.id 的指针。
  tool_call_id?: string;
}