// 后端 SSE typed union(text/tool_call/tool_result/question/error/done);旧独立类型已折叠。
export interface AskOption {
  // 按钮文案,同时原样成为下一轮 user message。
  label: string;
  // 渲染在 label 下方帮用户理解选项(不参与 LLM context)。
  desc: string;
}

// ask_user_tool 推上来的待答题;点选后被清掉并触发新一轮 sendMessage。
export interface QuestionPrompt {
  toolCallId: string;
  question: string;
  options: AskOption[];
}

export type StreamEvent =
  | { type: 'text'; delta: string }
  | {
      type: 'tool_call';
      toolCallId: string;
      name: string;
      args: Record<string, unknown>;
      description: string;
    }
  | {
      type: 'tool_result';
      toolCallId: string;
      name: string;
      result?: unknown;
      error?: string;
    }
  | {
      type: 'question';
      toolCallId: string;
      question: string;
      options: AskOption[];
    }
  | { type: 'error'; code: string; message: string }
  | { type: 'done'; messageId?: string };

// useConversations 持有的"已合并状态"卡片;tool_call 到达时 pending,tool_result 反查合并。
export type ToolCallStatus = 'pending' | 'done' | 'error';

// 后端 ToolCalls JSONB 反序列化形态;历史回放时 assistant 消息带这条数据生成 tool 卡片。
export interface ToolCallData {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: string;
  };
}

// 通用 tool 结果的渲染形态(bash 也用);error 非空时标红。
export interface ToolResultView {
  result?: unknown;
  error?: string;
}

export interface ToolCallView {
  toolCallId: string;
  name: string;
  description: string;
  args: Record<string, unknown>;
  status: ToolCallStatus;
  // 所有 tool(含 bash)统一挂在这里;bash 时 result 是 content string。
  toolResult?: ToolResultView;
}

export type AssistantEvent =
  | { kind: 'text'; content: string }
  | { kind: 'reason'; content: string }
  | { kind: 'tool'; tool: ToolCallView };