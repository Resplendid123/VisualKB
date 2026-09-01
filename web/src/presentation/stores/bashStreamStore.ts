import { create } from 'zustand';

// 跨会话累积 bash 调用,给右侧终端面板消费(不持久化,刷新即清)。turnId 把同 turn 多次串成一组。
export interface BashCall {
  toolCallId: string;
  turnId: string;
  // 所属 conversation;TerminalView 拿它过滤,流切走/切回不会让别会话的 bash 串进来。
  conversationId: string;
  command: string;
  description: string;
  startedAt: number;
}

export interface BashResult {
  toolCallId: string;
  // bash 的 stdout / stderr 由后端合并为一段 content;失败时附加 "error: ..." 行。
  content: string;
  // tool_result.error 非空时说明 tool 本身失败,UI 标红。
  error: string;
  finishedAt: number;
}

export interface BashEntry {
  call: BashCall;
  result?: BashResult;
}

interface BashStreamState {
  entries: BashEntry[];
}

interface BashStreamActions {
  addCall: (call: BashCall) => void;
  addResult: (result: BashResult) => void;
  clear: () => void;
}

export const useBashStreamStore = create<BashStreamState & BashStreamActions>(
  (set) => ({
    entries: [],

    addCall: (call) =>
      set((s) => ({ entries: [...s.entries, { call }] })),

    addResult: (result) =>
      set((s) => ({
        entries: s.entries.map((e) =>
          e.call.toolCallId === result.toolCallId
            ? { ...e, result }
            : e
        ),
      })),

    clear: () => set({ entries: [] }),
  })
);