'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { chatUseCases, documentUseCases } from '@/application/chatContainer';
import type { Conversation } from '@/domain/entities/conversation';
import type { Document } from '@/domain/entities/document';
import type { Message } from '@/domain/entities/message';
import type { Project, ActiveProject } from '@/domain/entities/project';
import type {
  AskOption,
  AssistantEvent,
  QuestionPrompt,
  ToolCallData,
  ToolCallView,
} from '@/domain/entities/streamEvent';
import type { StreamEvent } from '@/domain/entities/streamEvent';
import type { BashCall, BashResult } from '@/presentation/stores/bashStreamStore';
import { useBashStreamStore } from '@/presentation/stores/bashStreamStore';
import { usePanelStore } from '@/presentation/stores/panelStore';
import {
  parseArtifactToolResult,
  useArtifactStreamStore,
} from '@/presentation/stores/artifactStreamStore';
import { stripThinkTag } from '@/lib/stripThinkTag';

// 与后端 chat.BashName 对齐 — bash 是普通 tool,只是走终端面板分支。
const BASH_TOOL_NAME = 'bash';
// 与后端 ai/tools/project.ProjectToolName 对齐 — LLM 跑成功后
// 后端会同步把项目建好 + 把当前对话绑到该项目;前端这里负责把
// sidebar 项目列表 + 对话 activeProjectId + activeProject 缓存同步过去。
const CREATE_PROJECT_TOOL_NAME = 'create_project';
// 与后端 ai/tools/document.*ToolName 对齐 — LLM 调这两个工具时,
// 前端要把右侧文件预览面板打开并切到对应文档,让用户立刻看到结果。
const EDIT_DOCUMENT_TOOL_NAME = 'edit_document';
const CREATE_DOCUMENT_TOOL_NAME = 'create_document';
// 与后端 ai/tools/artifact.BuildArtifactToolName 对齐 — LLM 跑完 build_artifact
// 后,后端会把最新 artifact URL 通过 SSE 推回,前端自动把右栏切到 Live Preview。
const BUILD_ARTIFACT_TOOL_NAME = 'build_artifact';

// 推理模型(DeepSeek-R1 / QwQ 等)把思维链塞在 assistant.content 的
// `` / `` 标签里 — text.delta 不是纯文本,而是 text + reason 混在一起的
// 拼版。前端做流式切分,把 reason 抽出来当独立 AssistantEvent,
// 正文剥掉标签,渲染时折叠卡片区分显示。
// 标签用拼接写,避开编辑工具把成对尖括号当 HTML 吞掉。
const THINK_START = '<' + 'think' + '>';
const THINK_END = '<' + '/' + 'think' + '>';

interface HistoryBashEvent {
  call: BashCall;
  result: BashResult;
}

interface ConversationState extends Conversation {
  messages: Message[];
  isLoaded: boolean;
}

const ASK_USER_TOOL_NAME = 'ask_user_tool';

// 从 messages.events 反推当前 pending 的 ask_user_tool 题目;从后往前找(LLM 在 tool_result 后可能再追加 assistant 正文)。
function derivePendingQuestion(messages: Message[]): QuestionPrompt | null {
  // 从后往前找最后一个 events 含 ask_user_tool 的 assistant 消息。
  let targetIdx = -1;
  for (let i = messages.length - 1; i >= 0; i--) {
    const m = messages[i];
    if (m.role !== 'assistant') continue;
    const events = m.events ?? [];
    if (events.some((e) => e.kind === 'tool' && e.tool.name === ASK_USER_TOOL_NAME)) {
      targetIdx = i;
      break;
    }
  }
  if (targetIdx === -1) return null;

  // 该 assistant 之后出现 user 消息 → 已答,不再 pending。
  for (let i = targetIdx + 1; i < messages.length; i++) {
    if (messages[i].role === 'user') return null;
  }

  // 在该 assistant 的 events 里找最后那次 ask_user_tool 调用 — 一次 turn 可能问多次。
  const events = messages[targetIdx].events ?? [];
  for (let i = events.length - 1; i >= 0; i--) {
    const e = events[i];
    if (e.kind !== 'tool') continue;
    if (e.tool.name !== ASK_USER_TOOL_NAME) continue;

    const args = e.tool.args ?? {};
    const question = typeof args.question === 'string' ? args.question.trim() : '';
    if (!question) return null;
    const rawOpts = Array.isArray(args.options) ? args.options : [];
    const options: AskOption[] = [];
    for (const raw of rawOpts) {
      if (!raw || typeof raw !== 'object') continue;
      const obj = raw as Record<string, unknown>;
      if (typeof obj.label !== 'string' || typeof obj.desc !== 'string') continue;
      options.push({ label: obj.label, desc: obj.desc });
    }
    if (options.length === 0) return null;
    return { toolCallId: e.tool.toolCallId, question, options };
  }
  return null;
}

const DRAFT_ID = '__new__';

function draftConversation(): ConversationState {
  return { id: DRAFT_ID, title: '新对话', messages: [], isLoaded: true };
}

function newMessage(role: Message['role'], content: string): Message {
  // turn_id/seq_id 默认 0:本地占位 / 用户消息,直到 streamToAssistant 走完、
  // backend 把 list 重排时再拿到真实值。0 与 list 末尾比对时永远 ≤ 任意已落库 msg,
  // 不会误触发 Replay 元组过滤。
  return {
    id: crypto.randomUUID(),
    role,
    content,
    turn_id: 0,
    seq_id: 0,
    created_at: Date.now(),
  };
}

function nowMs(): number {
  return Date.now();
}

// 流式 text delta 的 think 切分状态机;残留"半个标签"放 pending 等下个 chunk 拼上。

interface ThinkSplitState {
  pending: string;
  isInThink: boolean;
}

interface ThinkSplitResult {
  text: string;
  reason: string;
  next: ThinkSplitState;
}

// 兜底:把漏过 splitter 的字面 think 标签擦掉,避免漂到正文。
const stripThinkTagLiteral = stripThinkTag;

function splitTextDelta(state: ThinkSplitState, delta: string): ThinkSplitResult {
  let s = state.pending + delta;
  let text = '';
  let reason = '';

  while (s.length > 0) {
    const tag = state.isInThink ? THINK_END : THINK_START;
    const idx = s.indexOf(tag);
    if (idx < 0) {
      // 没完整 tag — 先尝试"末尾是 tag 的前缀"模式(常规跨 chunk 切分);
      // 如果没匹配,再扫整段 s 找最右一个 '<' 看它到末尾是不是 tag 的前缀 —
      // 这是 < / <t / <th / <thi 等"刚开始打 tag"就被切到下个 chunk 的情况。
      let partial = '';
      const max = Math.min(tag.length - 1, s.length);
      for (let len = max; len > 0; len--) {
        if (tag.startsWith(s.slice(s.length - len))) {
          partial = s.slice(s.length - len);
          break;
        }
      }
      if (partial === '') {
        // 找最右一个 '<',看从它到末尾是否是 tag 的前缀
        const lt = s.lastIndexOf('<');
        if (lt >= 0) {
          const tail = s.slice(lt);
          if (tail.length < tag.length && tag.startsWith(tail)) {
            partial = tail;
          }
        }
      }
      const emit = s.slice(0, s.length - partial.length);
      if (state.isInThink) reason += emit;
      else text += emit;
      s = partial;
      break;
    }
    // tag 完整 — emit tag 之前的部分,翻状态。
    const emit = s.slice(0, idx);
    if (state.isInThink) reason += emit;
    else text += emit;
    s = s.slice(idx + tag.length);
    state = { pending: '', isInThink: !state.isInThink };
  }

  // 兜底:任何残留的 `` / `` 字面(例如模型裸打或 splitter 漏掉的边缘 case)
  // 都从 emit 出的 text / reason 里擦掉,绝不显示字面标签。
  return {
    text: stripThinkTagLiteral(text),
    reason: stripThinkTagLiteral(reason),
    next: { pending: s, isInThink: state.isInThink },
  };
}

// 流结束时把残留 pending 当作当前模式 emit;只在 chunk 边界切在 tag 中间才有内容。
function flushThinkSplit(state: ThinkSplitState): ThinkSplitResult {
  const text = state.isInThink ? '' : state.pending;
  const reason = state.isInThink ? state.pending : '';
  return { text, reason, next: { pending: '', isInThink: state.isInThink } };
}

// 把历史 assistant/tool 行反推出 AssistantEvent;bash call/result 也交给 bashStreamStore。
function rebuildEventsFromHistory(
  rawMsgs: Message[],
  conversationId: string
): {
  messages: Message[];
  bashEvents: HistoryBashEvent[];
} {
  // tool_call_id -> tool 消息,反查 result 用
  const toolByCallId = new Map<string, Message>();
  for (const m of rawMsgs) {
    if (m.role === 'tool' && m.tool_call_id) {
      toolByCallId.set(m.tool_call_id, m);
    }
  }

  const bashEvents: HistoryBashEvent[] = [];
  // 同轮的 bash 共享 roundId(=该轮 user.id),无 user 时统一兜底
  let roundId = 'no-user';
  for (const m of rawMsgs) {
    if (m.role === 'user') roundId = m.id;
  }

  // 顺序遍历:filter tool + 给 assistant 生成 events
  const filtered: Message[] = [];
  for (const m of rawMsgs) {
    if (m.role === 'tool') continue;
    if (m.role === 'user') {
      filtered.push(m);
      continue;
    }
    const events: AssistantEvent[] = [];
    if (m.content) {
      for (const seg of splitThinkContent(m.content)) {
        if (seg.kind === 'reason') {
          events.push({ kind: 'reason', content: seg.text });
        } else {
          events.push({ kind: 'text', content: seg.text });
        }
      }
    }
    const calls = m.tool_calls ?? [];
    for (const c of calls) {
      const view = toolViewFromHistory(
        m,
        c,
        toolByCallId.get(c.id),
        bashEvents,
        conversationId,
        roundId
      );
      if (c.function.name !== BASH_TOOL_NAME) {
        events.push({ kind: 'tool', tool: view });
      }
    }
    filtered.push({ ...m, events });
  }

  // 同轮相邻 assistant 行合并成一个 bubble
  return { messages: coalesceAssistantRounds(filtered), bashEvents };
}

// 把同轮相邻 assistant 行合并:取首条 id,content/events 串接。
function coalesceAssistantRounds(messages: Message[]): Message[] {
  const out: Message[] = [];
  let buf: Message[] = [];

  function flush() {
    if (buf.length === 0) return;
    if (buf.length === 1) {
      out.push(buf[0]);
      buf = [];
      return;
    }
    const head = buf[0];
    const content = buf
      .map(plainTextOf)
      .filter((s) => s.length > 0)
      .join('\n\n');
    const events = buf.flatMap((m) => m.events ?? []);
    out.push({ ...head, content, events });
    buf = [];
  }

  for (const m of messages) {
    if (m.role === 'user') {
      flush();
      out.push(m);
    } else {
      buf.push(m);
    }
  }
  flush();
  return out;
}

// 取 content 的纯文本段(去 think 段)。
function plainTextOf(m: Message): string {
  if (!m.content) return '';
  return splitThinkContent(m.content)
    .filter((s) => s.kind === 'text')
    .map((s) => s.text)
    .join('\n\n')
    .trim();
}

// 把 content 切成 reason/text 段。
function splitThinkContent(s: string): { kind: 'reason' | 'text'; text: string }[] {
  const out: { kind: 'reason' | 'text'; text: string }[] = [];
  let cursor = 0;
  while (cursor < s.length) {
    const startIdx = s.indexOf(THINK_START, cursor);
    if (startIdx < 0) {
      // 没 think 标签则全是 text
      out.push({ kind: 'text', text: s.slice(cursor) });
      break;
    }
    if (startIdx > cursor) {
      out.push({
        kind: 'text',
        text: stripThinkTagLiteral(s.slice(cursor, startIdx)),
      });
    }
    const afterOpen = startIdx + THINK_START.length;
    const endIdx = s.indexOf(THINK_END, afterOpen);
    if (endIdx < 0) {
      // 没配对到 </think> — 标签内的内容当 reason,避免丢数据;
      // 下游渲染 reason 卡片原样展示,不需要带 closing tag。
      out.push({
        kind: 'reason',
        text: stripThinkTagLiteral(s.slice(afterOpen)),
      });
      cursor = s.length;
      break;
    }
    out.push({
      kind: 'reason',
      text: stripThinkTagLiteral(s.slice(afterOpen, endIdx)),
    });
    cursor = endIdx + THINK_END.length;
  }
  return out.filter((seg) => seg.text.length > 0);
}

function toolViewFromHistory(
  assistant: Message,
  c: ToolCallData,
  result: Message | undefined,
  bashEvents: HistoryBashEvent[],
  conversationId: string,
  roundId: string
): ToolCallView {
  let args: Record<string, unknown> = {};
  try {
    const parsed = JSON.parse(c.function.arguments);
    if (parsed && typeof parsed === 'object') args = parsed as Record<string, unknown>;
  } catch {
    // 损坏的 arguments 当空 args
  }

  const isBash = c.function.name === BASH_TOOL_NAME;
  const finishedAt = assistant.created_at + 1;

  if (isBash) {
    const content = result?.content ?? '';
    const command = typeof args.command === 'string' ? args.command : '';
    bashEvents.push({
      call: {
        toolCallId: c.id,
        turnId: roundId,
        conversationId,
        command,
        description: '',
        startedAt: assistant.created_at,
      },
      result: {
        toolCallId: c.id,
        content,
        error: '',
        finishedAt,
      },
    });
  }

  return {
    toolCallId: c.id,
    name: c.function.name,
    description: '',
    args,
    status: result ? 'done' : 'pending',
    toolResult: result ? { result: result.content, error: undefined } : undefined,
  };
}

// 会话状态 + AI 流式编排:数据源全部走真实后端,AbortController 统一处理中断。
export function useConversations() {
  const [conversations, setConversations] = useState<ConversationState[]>([]);
  const [pending, setPending] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);

  // URL → 内部 state 反向同步:URL 是 single source of truth。
  // path /c/:id → activeId = id;?project=:pid → focusedProjectId = pid;
  // 其他路径(DRAFT / notes / knowledge / settings 等)→ activeId = DRAFT_ID。
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  // 从 URL 直接派生活跃会话 id 和聚焦项目 id — 避免 handler 里读 closure 拿到旧值。
  // handler 用 router.push 后,effect 异步触发 setState,中间那一帧的 closure 是旧 state;
  // 这里直接读 URL 的派生值始终反映「最新一次推送」,跟 effect 同步,但不进 React state。
  const urlActiveId = useMemo(() => {
    const m = pathname?.match(/^\/c\/([^/?]+)/);
    return m ? decodeURIComponent(m[1]) : DRAFT_ID;
  }, [pathname]);
  const urlFocusedProjectId = useMemo(() => searchParams?.get('project') ?? null, [searchParams]);
  const activeId = urlActiveId;
  const focusedProjectId = urlFocusedProjectId;
  // 用户当前"聚焦"的项目 — sidebar 点项目行设置;后续新建对话会绑到它(与后端 active project 解耦)。
  const [activeProject, setActiveProject] = useState<ActiveProject | null | undefined>(undefined);
  // ask_user_tool 的待答题目直接 derive 自 active.messages(ask_user_tool 事件的
  // args 里就有 question + options);切会话、刷新都跟着消息走,不需要独立 state。
  // 真正的 useMemo 见 active 定义之后。
  const abortRef = useRef<AbortController | null>(null);
  // 每条 assistant 消息一条 think-split 状态机;按 msg.id 索引。
  // 不放在 message 里 — applyEvent 是纯函数,状态机副作用由 hook 持有。
  const thinkStateRef = useRef<Map<string, ThinkSplitState>>(new Map());
  // 每个对话 list 末尾 msg 的 (turn_id, seq_id) 元组(后端 GetMessages 给),
  // 作为 Replay 续传锚点 — 刷新 / 切回对话时若 inFlight=true 转走 subscribeReplay,
  // 把 (lastTurn, lastSeqID) 通过 last_turn / last_seq_id query 给后端,
  // 后端按 (turn, seq) 字典序 > 元组过滤 list 已渲染部分。
  // 键是 conversationId — 同一对话多轮对话共用同一个锚点。
  const baselineAnchorRef = useRef<Map<string, { turn: number; seqID: number }>>(new Map());

  // mount:拉一次会话列表 + 项目列表
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const { items } = await chatUseCases.listConversations.execute({
          limit: 50,
          offset: 0,
        });
        if (cancelled) return;
        setConversations(items.map((c) => ({ ...c, messages: [], isLoaded: false })));
        // 进入页面默认停在草稿态,不自动选中第一条 — 否则侧边栏会无故高亮,
        // 但中间区域还没拉 messages,视觉/语义都不一致。让用户主动点选或新建。
      } catch (e) {
        // 列表拉失败不致命,UI 上展示空会话列表 + 草稿态
        console.error('list conversations failed', e);
      } finally {
        // no-op: list 拉取结束标志位曾用于驱动 Loading UI,现已删除(组件不再需要)。
      }
      // project 列表独立 try,不影响 convo 列表;失败就展示空项目区。
      try {
        const ps = await chatUseCases.listProjects.execute();
        if (!cancelled) setProjects(ps);
      } catch (e) {
        console.error('list projects failed', e);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // 切换 activeId 时拉 active project;previewUrl 为空则轮询,直到 controller stamp。
  useEffect(() => {
    if (activeId === DRAFT_ID) return;
    let cancelled = false;
    let pollTimer: ReturnType<typeof setTimeout> | null = null;
    const fetchOnce = async () => {
      try {
        const ap = await chatUseCases.getActiveProject.execute(activeId);
        if (cancelled) return;
        setActiveProject(ap);
        // previewUrl 还没就位 → 2s 后再拉一次,直到 controller 完成 reconcile。
        if (ap && !ap.previewUrl) {
          pollTimer = setTimeout(fetchOnce, 2000);
        }
      } catch (e) {
        console.error('get active project failed', e);
        if (!cancelled) setActiveProject(null);
      }
    };
    fetchOnce();
    return () => {
      cancelled = true;
      if (pollTimer) clearTimeout(pollTimer);
    };
  }, [activeId]);

  const visibleActiveProject = activeId === DRAFT_ID ? undefined : activeProject;

  const active: ConversationState =
    activeId === DRAFT_ID
      ? draftConversation()
      : (conversations.find((c) => c.id === activeId) ?? draftConversation());

  const pendingQuestion = useMemo(
    () => (activeId === DRAFT_ID ? null : derivePendingQuestion(active.messages)),
    [activeId, active.messages]
  );

  // 输入区 chip:已绑 active project 用 active,草稿态用 focused,都没有则 null。
  const focusedOrActiveProject: Project | ActiveProject | null = (() => {
    if (visibleActiveProject) return visibleActiveProject;
    if (focusedProjectId) {
      return projects.find((p) => p.id === focusedProjectId) ?? null;
    }
    return null;
  })();

  // 传给侧边栏的精简列表(不含 messages)— activeProjectId 用来把对话归到对应项目下。
  const sidebarConversations: Conversation[] = conversations.map((c) => ({
    id: c.id,
    title: c.title,
    activeProjectId: c.activeProjectId,
  }));
  const sidebarActiveId = activeId === DRAFT_ID ? '' : activeId;

  // 就地更新某条消息 — 收敛所有"改单条消息"的写法。
  function patchMessage(targetId: string, msgId: string, patch: (m: Message) => Message) {
    setConversations((prev) =>
      prev.map((c) =>
        c.id !== targetId
          ? c
          : { ...c, messages: c.messages.map((m) => (m.id === msgId ? patch(m) : m)) }
      )
    );
  }

  // 选中会话时按需拉 messages(去重 + 标记 isLoaded);inFlight 时自动续传尾巴。
  async function ensureLoaded(targetId: string) {
    const target = conversations.find((c) => c.id === targetId);
    if (!target || target.isLoaded) return;
    try {
      const {
        items: rawMsgs,
        lastTurnAtLoad,
        lastSeqIDAtLoad,
        inFlight,
      } = await chatUseCases.getMessages.execute(targetId);
      // 历史消息没有 events 字段 — 把 assistant.tool_calls 与后续 tool 消息配对,
      // 还原出跟流式一致的 events,MessageBubble / EventStreamView 不必分支处理。
      const { messages, bashEvents } = rebuildEventsFromHistory(rawMsgs, targetId);
      // 把加载瞬间 list 末尾 msg 的 (turn_id, seq_id) 记为续传锚点;后续
      // maybeAutoResumeReplay 会把它写进 last_turn / last_seq_id query 给后端。
      baselineAnchorRef.current.set(targetId, { turn: lastTurnAtLoad, seqID: lastSeqIDAtLoad });
      // 顺手把历史 bash 调用塞到右侧终端面板,切回对话也能看到之前的命令 + 输出。
      // 只更新数据,不强制打开面板 — 面板的 open / mode 跟着用户上次的选择走。
      if (bashEvents.length > 0) {
        const store = useBashStreamStore.getState();
        store.clear();
        for (const ev of bashEvents) {
          store.addCall(ev.call);
          store.addResult(ev.result);
        }
      }
      setConversations((prev) =>
        prev.map((c) => (c.id !== targetId ? c : { ...c, messages, isLoaded: true }))
      );
      // 刷新 / 切回自动接流:messages 已经在闭包里同步可见,直接判。
      // 不用第二轮 setConversations 读 state,避免异步窗口期误判。
      maybeAutoResumeReplay(targetId, messages, lastTurnAtLoad, lastSeqIDAtLoad, inFlight);
    } catch (e) {
      console.error('get messages failed', e);
    }
  }

  // 刷新 / 切回对话时按 inFlight 自动续传,补一个本地 assistant 占位承接续传事件。
  function maybeAutoResumeReplay(
    targetId: string,
    msgs: Message[],
    lastTurn: number,
    lastSeqID: number,
    inFlight: boolean
  ) {
    // 只有会话本就在飞 turn 才续;已完成 turn 的 list 已 done,空 Replay 浪费 SSE 槽位。
    if (!inFlight) return;

    const placeholder = newMessage('assistant', '');
    setConversations((prev) =>
      prev.map((c) => (c.id === targetId ? { ...c, messages: [...c.messages, placeholder] } : c))
    );

    const assistantId = placeholder.id;
    const controller = new AbortController();
    abortRef.current = controller;
    setPending(true);

    // roundId:本轮触发的 user 消息 id — 从 msgs 末尾倒序找最近一条 user,
    // bashStreamStore 用它作 turnId。
    let roundId = '';
    for (let i = msgs.length - 1; i >= 0; i--) {
      if (msgs[i].role === 'user') {
        roundId = msgs[i].id;
        break;
      }
    }

    void subscribeReplay(targetId, roundId, assistantId, controller, lastTurn, lastSeqID).then(() => {
      // 只有还是当前 controller 才收尾,否则已被 streamToAssistant 接管。
      if (abortRef.current === controller) {
        setPending(false);
        abortRef.current = null;
        thinkStateRef.current.delete(assistantId);
      }
    });
  }

  // URL 切到已有会话时拉消息。urlActiveId 改变触发初次检查;
  // conversations 进依赖是为了处理"URL 已切但 list fetch 还没到"的 race —
  // list 加载完成后 effect 重跑,ensureLoaded 才找到 target。
  useEffect(() => {
    if (urlActiveId === DRAFT_ID) return;
    const target = conversations.find((c) => c.id === urlActiveId);
    if (!target) return;
    void ensureLoaded(urlActiveId);
  }, [urlActiveId, conversations]); // eslint-disable-line react-hooks/exhaustive-deps

  // POST 流:把 chunk 推到目标 assistant 消息。EventSource 自动重连 + Last-Event-ID,不用前端手动 dedup。
  async function streamToAssistant(
    targetId: string,
    roundId: string,
    assistantId: string,
    text: string,
    documentIds?: number[],
    edit?: boolean
  ) {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setPending(true);

    try {
      const iter = chatUseCases.sendMessage.execute(
        { content: text, conversationId: targetId, documentIds, edit },
        controller.signal
      );
      for await (const rec of iter) {
        if (controller.signal.aborted) return;
        const ev = rec.ev;
        if (!ev) continue;
        dispatchStreamSideEffects(ev, roundId, targetId);
        patchMessage(targetId, assistantId, (m) => applyEvent(m, ev, assistantId));
        if (ev.type === 'done') break;
      }
      flushThinkSplitForMessage(targetId, assistantId);
    } catch (e) {
      if (controller.signal.aborted) return;
      flushThinkSplitForMessage(targetId, assistantId);
      console.warn('chat stream failed', { err: (e as Error).message });
    } finally {
      // 用户主动 stop / 流异常断开:把还在 pending 的 tool 标 error,不让 spinner
      // 永远转。done 已发过的话工具早就 reconcile 过了,这里是 belt-and-suspenders。
      patchMessage(targetId, assistantId, (m) => reconcilePendingTools(m));
      if (abortRef.current === controller) {
        setPending(false);
        abortRef.current = null;
      }
      thinkStateRef.current.delete(assistantId);
    }
  }

  // 续传订阅:浏览器 EventSource auto-reconnect + auto-Last-Event-ID,前端只管消费。返 true 表示见到 done 收尾。
  async function subscribeReplay(
    targetId: string,
    roundId: string,
    assistantId: string,
    controller: AbortController,
    lastTurn = 0,
    lastSeqID = 0
  ): Promise<boolean> {
    try {
      const iter = chatUseCases.replayEvents.execute(
        {
          conversationId: targetId,
          lastTurn: lastTurn || undefined,
          lastSeqID: lastSeqID || undefined,
        },
        controller.signal
      );
      for await (const rec of iter) {
        if (controller.signal.aborted) return false;
        const ev = rec.ev;
        if (!ev) continue;
        dispatchStreamSideEffects(ev, roundId, targetId);
        patchMessage(targetId, assistantId, (m) => applyEvent(m, ev, assistantId));
        if (ev.type === 'done') {
          flushThinkSplitForMessage(targetId, assistantId);
          return true;
        }
      }
      return false;
    } catch {
      if (controller.signal.aborted) return false;
      return false;
    } finally {
      patchMessage(targetId, assistantId, (m) => reconcilePendingTools(m));
    }
  }

  // 把还没收到 tool_result 的 pending tool 标 error;流已收尾(无论 done 还是 abort)时调用,避免 spinner 永远转。
  function reconcilePendingTools(m: Message): Message {
    const events = m.events ?? [];
    let touched = false;
    const next = events.map((e) => {
      if (e.kind !== 'tool' || e.tool.status !== 'pending') return e;
      touched = true;
      return {
        kind: 'tool' as const,
        tool: {
          ...e.tool,
          status: 'error' as const,
          toolResult: { error: 'turn ended before tool result arrived' },
        },
      };
    });
    return touched ? { ...m, events: next } : m;
  }

  function flushThinkSplitForMessage(targetId: string, assistantId: string) {
    const state = thinkStateRef.current.get(assistantId);
    if (!state) return;
    const out = flushThinkSplit(state);
    thinkStateRef.current.set(assistantId, out.next);
    if (!out.text && !out.reason) {
      if (out.next.pending === '') thinkStateRef.current.delete(assistantId);
      return;
    }
    patchMessage(targetId, assistantId, (m) => {
      const events = m.events ?? [];
      const next = out.text ? appendTextEvent(events, out.text) : events;
      return {
        ...m,
        content: m.content + out.text,
        events: out.reason ? appendReasonEvent(next, out.reason) : next,
      };
    });
  }

  // 从 result payload 里抽出嵌套的 { document_id };字符串/对象都防御式 parse,缺失返回 null。
  function parseDocumentId(result: unknown): number | null {
    let r: Record<string, unknown> | null = null;
    if (typeof result === 'string') {
      try {
        const parsed = JSON.parse(result);
        if (parsed && typeof parsed === 'object') r = parsed as Record<string, unknown>;
      } catch {
        // 非 JSON 字符串 — 当作没解析价值。
      }
    } else if (result && typeof result === 'object') {
      r = result as Record<string, unknown>;
    }
    if (!r) return null;
    const id = r.document_id;
    return typeof id === 'number' && id > 0 ? id : null;
  }

  // 拉文档元数据加进 attachedDocs + 切到 file 模式打开;失败只 console.error,不抛。
  async function openDocumentInPanel(docId: number) {
    try {
      const detail = await documentUseCases.get.execute(docId);
      // panelStore 里的 Document 不含 content(只放元数据,正文由 preview 视图自己拉);
      // DocumentDetail 多一个 content 字段,这里剥掉即可。
      const docMeta: Document = {
        id: detail.id,
        title: detail.title,
        source: detail.source,
        lang: detail.lang,
        content_type: detail.content_type,
        chunk_status: detail.chunk_status,
        current_version_id: detail.current_version_id,
        archived_at: detail.archived_at,
        created_at: detail.created_at,
        updated_at: detail.updated_at,
      };
      const panel = usePanelStore.getState();
      panel.addAttachedDoc(docMeta);
      panel.setPreviewDocId(docId);
      panel.setMode('file');
      panel.setOpen(true);
    } catch (e) {
      console.error(`open document ${docId} in panel failed`, e);
    }
  }

  // store 副作用必须在 applyEvent 之前单独调,渲染阶段 set() 会触发"Cannot update while rendering"。
  function dispatchStreamSideEffects(ev: StreamEvent, roundId: string, targetId: string) {
    switch (ev.type) {
      case 'tool_call':
        if (ev.name === BASH_TOOL_NAME) {
          // bash 走终端面板 — command 从 args 提(description 由 tool_call 自带)。
          // 续传可能重放同 toolCallId,addCall 在 store 里是 push 语义(无去重),
          // 这里手动挡一下,避免终端出现重复条目。
          const bashStore = useBashStreamStore.getState();
          if (!bashStore.entries.some((e) => e.call.toolCallId === ev.toolCallId)) {
            bashStore.addCall({
              toolCallId: ev.toolCallId,
              turnId: roundId,
              conversationId: targetId,
              command: typeof ev.args.command === 'string' ? ev.args.command : '',
              description: ev.description,
              startedAt: nowMs(),
            });
          }
          usePanelStore.getState().setOpen(true);
          // 同时切到 terminal 视图 —— 否则用户上次选了默认页/文件预览,
          // 打开面板只能看到旧内容,得手动再切一次。
          usePanelStore.getState().setMode('terminal');
        } else if (ev.name === EDIT_DOCUMENT_TOOL_NAME) {
          // edit_document:document_id 在 args 里,直接拉元数据 → 打开预览。
          const docId = typeof ev.args.document_id === 'number' ? ev.args.document_id : NaN;
          if (docId > 0) void openDocumentInPanel(docId);
        }
        // create_document 的 document_id 在 tool_result 才出现,见下面。
        break;
      case 'tool_result':
        if (ev.name === BASH_TOOL_NAME) {
          // result 是后端给的 content 段(可能含 "error: ..." 行);
          // tool_result.error 是后端的独立 error 字段,失败时非空。
          const content =
            typeof ev.result === 'string'
              ? ev.result
              : ev.result == null
                ? ''
                : JSON.stringify(ev.result);
          useBashStreamStore.getState().addResult({
            toolCallId: ev.toolCallId,
            content,
            error: ev.error ?? '',
            finishedAt: nowMs(),
          });
        } else if (ev.name === CREATE_PROJECT_TOOL_NAME) {
          // LLM 在草稿/无 active project 的对话里跑 create_project,
          // 后端会同步把项目建好 + 把当前对话绑到该项目(activeProjectId 被更新)。
          // 前端这里专门同步状态:sidebar 项目列表 + 当前对话 activeProjectId
          // + activeProject 缓存,这样三处立刻反映出来,不用等下次刷新。
          applyCreateProjectToolResult(ev.result, targetId);
        } else if (ev.name === CREATE_DOCUMENT_TOOL_NAME) {
          // create_document 的结果就是 { document_id };拿到 id 拉元数据 → 打开预览。
          const docId = parseDocumentId(ev.result);
          if (docId != null) void openDocumentInPanel(docId);
        } else if (ev.name === BUILD_ARTIFACT_TOOL_NAME) {
          // build_artifact 跑完一轮:
          //   1) 自动把右栏打开并切到 Live Preview(setOpen 在 setMode 前,避免展开/切换分两次渲染)。
          //   2) 解析 result 拿到 url/id,推进 artifactStreamStore,LivePreviewView
          //      订阅后立即重新拉最新 artifact + 切 iframe src。
          const panel = usePanelStore.getState();
          panel.setOpen(true);
          panel.setMode('live');
          const parsed = parseArtifactToolResult(ev.result);
          if (parsed) {
            useArtifactStreamStore.getState().notifyBuilt({
              conversationId: targetId,
              artifactId: parsed.id,
              url: parsed.url,
            });
          }
        }
        break;
      case 'question':
        // 题目数据已经在 tool_call SSE 的 args 里,derivePendingQuestion
        // 从 message.events 反推 — 这里不需要额外动作。
        break;
    }
  }

  // 把 LLM create_project 结果同步到 sidebar / chip / 缓存;字符串/对象防御式 parse。
  function applyCreateProjectToolResult(result: unknown, targetId: string) {
    let r: Record<string, unknown> | null = null;
    if (typeof result === 'string') {
      try {
        const parsed = JSON.parse(result);
        if (parsed && typeof parsed === 'object') r = parsed as Record<string, unknown>;
      } catch {
        // 非 JSON 字符串 — 当作无解析价值,直接忽略。
      }
    } else if (result && typeof result === 'object') {
      r = result as Record<string, unknown>;
    }
    if (!r) return;
    const id = typeof r.id === 'string' ? r.id : null;
    if (!id) return;

    const title =
      typeof r.title === 'string' ? r.title : typeof r.name === 'string' ? r.name : '未命名';
    const project: Project = {
      id,
      name: typeof r.name === 'string' ? r.name : '',
      title,
      cwd: typeof r.cwd === 'string' ? r.cwd : '',
      status: typeof r.status === 'string' ? r.status : 'ready',
      updatedAt: typeof r.updated_at === 'string' ? r.updated_at : new Date().toISOString(),
    };

    // 1) projects 列表 — 去重头插(sidebar 立刻看到)
    setProjects((prev) => {
      if (prev.some((p) => p.id === id)) return prev;
      return [project, ...prev];
    });

    // 2) 当前对话的 activeProjectId — sidebar 把它归到新项目下
    setConversations((prev) =>
      prev.map((c) => (c.id === targetId ? { ...c, activeProjectId: id } : c))
    );

    // 3) activeProject 缓存 — 输入框 chip 和上层 fallback 都靠它
    const activeProj: ActiveProject = {
      id: project.id,
      name: project.name,
      title: project.title,
      cwd: project.cwd,
      updatedAt: project.updatedAt,
    };
    setActiveProject(activeProj);
  }

  // 纯函数,把单个 StreamEvent 折叠进 message(events + content);不可在此调 zustand(走 dispatchStreamSideEffects)。
  function applyEvent(msg: Message, ev: StreamEvent, assistantId: string): Message {
    const events = msg.events ?? [];
    switch (ev.type) {
      case 'text': {
        // 后端把 <think>...</think> 标签原样塞在 delta 里,这里按 chunk 切。
        const cur = thinkStateRef.current.get(assistantId) ?? {
          pending: '',
          isInThink: false,
        };
        const split = splitTextDelta(cur, ev.delta);
        thinkStateRef.current.set(assistantId, split.next);
        const next = { ...msg, content: msg.content + split.text };
        let nextEvents: AssistantEvent[] = events;
        if (split.text) nextEvents = appendTextEvent(nextEvents, split.text);
        if (split.reason) nextEvents = appendReasonEvent(nextEvents, split.reason);
        return { ...next, events: nextEvents };
      }
      case 'tool_call': {
        const tool: ToolCallView = {
          toolCallId: ev.toolCallId,
          name: ev.name,
          description: ev.description,
          args: ev.args,
          status: 'pending',
        };
        // bash 工具不进 events(由 dispatchStreamSideEffects 推到终端面板)。
        if (ev.name === BASH_TOOL_NAME) return msg;
        return { ...msg, events: [...events, { kind: 'tool', tool }] };
      }
      case 'tool_result': {
        const matchIdx = events.findIndex(
          (e) => e.kind === 'tool' && e.tool.toolCallId === ev.toolCallId
        );
        const status = ev.error ? 'error' : 'done';
        const toolResult = { result: ev.result, error: ev.error };
        // 没找到配对的 tool_call(并发工具的 tool_call 被过滤/replay 丢/早于订阅到达):
        // 补一条 stub 占位,避免 status=pending 永远转不出 spinner。
        if (matchIdx === -1) {
          return {
            ...msg,
            events: [
              ...events,
              {
                kind: 'tool',
                tool: {
                  toolCallId: ev.toolCallId,
                  name: ev.name,
                  description: '',
                  args: {},
                  status,
                  toolResult,
                },
              },
            ],
          };
        }
        return {
          ...msg,
          events: events.map((e) =>
            e.kind === 'tool' && e.tool.toolCallId === ev.toolCallId
              ? { kind: 'tool', tool: { ...e.tool, status, toolResult } }
              : e
          ),
        };
      }
      // error 事件:后端把 LLM/编排错误作为 SSE event 推出,直接写到 content 让用户看见,
      // 不再让 default 静默吞掉(否则用户看到空气泡 = "没回应")。
      case 'error': {
        return { ...msg, content: `出错了：${ev.message}` };
      }
      // question:由 dispatchStreamSideEffects 推到 pendingQuestion state,
      // 聊天气泡里不渲染(免得和输入框上方的题目卡片重复);events 数组保持原样。
      case 'question':
        return msg;
      case 'done':
        // turn 已收尾:任何还在 pending 的 tool 都没等到 tool_result(网络丢/服务端
        // 异常),不让 spinner 永远转。best-effort 标 error + 兜底文案。
        return {
          ...msg,
          events: events.map((e) =>
            e.kind === 'tool' && e.tool.status === 'pending'
              ? {
                  kind: 'tool',
                  tool: {
                    ...e.tool,
                    status: 'error',
                    toolResult: {
                      error: 'turn ended before tool result arrived',
                    },
                  },
                }
              : e
          ),
        };
      default:
        return { ...msg, events };
    }
  }

  // 把新 delta 拼到末尾的 text 块;末尾不是 text 则新建 — 保证连续 text 只渲染一段 markdown。
  function appendTextEvent(events: AssistantEvent[], delta: string): AssistantEvent[] {
    if (events.length === 0) return [{ kind: 'text', content: delta }];
    const last = events[events.length - 1];
    if (last.kind === 'text') {
      return [...events.slice(0, -1), { kind: 'text', content: last.content + delta }];
    }
    return [...events, { kind: 'text', content: delta }];
  }

  // 与 appendTextEvent 同形,把 reason delta 拼到末尾的 reason 段,渲染时只一个折叠卡片。
  function appendReasonEvent(events: AssistantEvent[], delta: string): AssistantEvent[] {
    if (events.length === 0) return [{ kind: 'reason', content: delta }];
    const last = events[events.length - 1];
    if (last.kind === 'reason') {
      return [...events.slice(0, -1), { kind: 'reason', content: last.content + delta }];
    }
    return [...events, { kind: 'reason', content: delta }];
  }

  async function sendMessage(text: string, documentIds?: number[]): Promise<boolean> {
    const userMsg = newMessage('user', text);
    // 读 URL 派生值,不读 activeId / focusedProjectId 的 closure — 后者在 router.push 后
    // 会有一帧旧值,导致草稿态切了项目再发消息绑错或漏绑。
    const currentActiveId = urlActiveId;
    const currentFocusedProjectId = urlFocusedProjectId;

    // 草稿态:先调后端创建会话,拿到真实 id 再走 SSE
    if (currentActiveId === DRAFT_ID) {
      const title = text.length > 20 ? text.slice(0, 20) + '…' : text;
      let realId: string;
      try {
        const created = await chatUseCases.createConversation.execute({
          title,
        });
        realId = created.id;
      } catch (e) {
        // 失败时不动前端 input/attachedDocs — chat-interface 拿 false 不会清,文本和 @ 还在。
        console.error('create conversation failed', e);
        return false;
      }
      // 如果有 active project(sidebar 当前选中某个项目),
      // 把新对话也绑到同一个项目 — 后端不强制,前端按 UX 决定。
      if (currentFocusedProjectId) {
        try {
          await chatUseCases.switchActiveProject.execute(realId, currentFocusedProjectId);
        } catch (e) {
          console.error('bind new conversation to project failed', e);
        }
      }
      setConversations((prev) => [
        {
          id: realId,
          title,
          messages: [userMsg],
          isLoaded: true,
          ...(currentFocusedProjectId ? { activeProjectId: currentFocusedProjectId } : {}),
        },
        ...prev,
      ]);
      router.push(`/c/${encodeURIComponent(realId)}`);

      const assistantMsg = newMessage('assistant', '');
      setConversations((prev) =>
        prev.map((c) => (c.id === realId ? { ...c, messages: [...c.messages, assistantMsg] } : c))
      );
      void streamToAssistant(realId, userMsg.id, assistantMsg.id, text, documentIds);
      return true;
    }

    // 现有会话:本地先插一条 user + 占位 assistant,再起流。
    // 新插入的 user 消息让 derivePendingQuestion 自动判定为"已答",卡片消失。
    const targetId = currentActiveId;
    setConversations((prev) =>
      prev.map((c) => (c.id === targetId ? { ...c, messages: [...c.messages, userMsg] } : c))
    );

    const assistantMsg = newMessage('assistant', '');
    setConversations((prev) =>
      prev.map((c) => (c.id === targetId ? { ...c, messages: [...c.messages, assistantMsg] } : c))
    );

    void streamToAssistant(targetId, userMsg.id, assistantMsg.id, text, documentIds);
    return true;
  }

  // 编辑最近一条 user 消息重发;后端标旧分支 is_modified,本地丢弃旧尾巴 + 补新 user + 占位 assistant。
  function editMessage(msgId: string, content: string) {
    const targetId = urlActiveId;
    if (targetId === DRAFT_ID) return;

    const userMsg = newMessage('user', content);
    const assistantMsg = newMessage('assistant', '');
    setConversations((prev) =>
      prev.map((c) => {
        if (c.id !== targetId) return c;
        const idx = c.messages.findIndex((m) => m.id === msgId);
        if (idx === -1) return c;
        return { ...c, messages: [...c.messages.slice(0, idx), userMsg, assistantMsg] };
      })
    );

    void streamToAssistant(targetId, userMsg.id, assistantMsg.id, content, undefined, true);
  }

  // 用户主动停止当前流。
  function stop() {
    abortRef.current?.abort();
    abortRef.current = null;
    setPending(false);
  }

  function newConversation() {
    // 顶部"新建会话"= 干净草稿;同时清掉上次聚焦的项目,避免残留 focusedProjectId
    // 让第一条消息绑到一个用户没显式选过的项目。
    router.push('/');
  }

  // 归档当前对话 → 后端标 archived_at,本地剔除并回到草稿态。归档失败则保留列表不动。
  async function deleteActive() {
    if (urlActiveId === DRAFT_ID) return;
    const target = urlActiveId;
    try {
      await chatUseCases.archiveConversation.execute(target);
    } catch (e) {
      // 抛给 caller 做 toast;不动本地列表,避免"本地没了但后端还在"。
      throw e instanceof Error ? e : new Error(String(e));
    }
    setConversations(conversations.filter((c) => c.id !== target));
    router.push('/');
    baselineAnchorRef.current.delete(target);
  }

  // 选择已有会话时清掉项目聚焦,避免项目高亮和 chip 残留。
  function selectConversation(id: string) {
    router.push(`/c/${encodeURIComponent(id)}`);
  }

  // 创建项目并切到草稿态 + 聚焦该项目,让用户直接进入"在新项目里发起对话"。
  const createProject = useCallback(
    async (title: string): Promise<Project> => {
      const p = await chatUseCases.createProject.execute(title);
      setProjects((prev) => [p, ...prev]);
      router.push(`/?project=${encodeURIComponent(p.id)}`);
      // 旧对话挂着的待答题目保留 — 切回去时照样能看见。
      return p;
    },
    [router]
  );

  // 改 Title;失败由调用方处理,UI 这里不弹错误,只把成功结果回写列表。
  const renameProject = useCallback(async (id: string, title: string): Promise<Project> => {
    const p = await chatUseCases.renameProject.execute(id, title);
    setProjects((prev) => prev.map((x) => (x.id === id ? p : x)));
    // 如果该 project 是当前对话的 active,刷新 activeProject 缓存里的 title。
    setActiveProject((cur) => (cur && cur.id === id ? { ...cur, title: p.title } : cur));
    return p;
  }, []);

  // 软删项目 + 联动归档子会话;聚焦项目/子会话切到草稿。
  const archiveProject = useCallback(
    async (id: string): Promise<void> => {
      const childIds = conversations.filter((c) => c.activeProjectId === id).map((c) => c.id);
      await chatUseCases.archiveProject.execute(id);
      // child 归档失败不重试:项目已归档,child activeProjectId 指向已删项目,
      // 下次刷新由后端权威状态覆盖。
      await Promise.all(
        childIds.map((cid) => chatUseCases.archiveConversation.execute(cid).catch(() => undefined))
      );
      setProjects((prev) => prev.filter((p) => p.id !== id));
      // URL 是 single source of truth,只推 URL;命中聚焦项目/子会话 → 退回草稿。
      if (urlFocusedProjectId === id || childIds.includes(urlActiveId)) {
        router.push('/');
      }
      setConversations((prev) => prev.filter((c) => !childIds.includes(c.id)));
    },
    [conversations, urlFocusedProjectId, urlActiveId, router]
  );

  // 进入"以该项目为目标的草稿态",不动当前对话的 active project;后续 sendMessage 会把新对话绑到 pickedId。
  const pickProject = useCallback(
    async (id: string): Promise<void> => {
      router.push(`/?project=${encodeURIComponent(id)}`);
    },
    [router]
  );

  // 题目卡片点了 option → 当下一轮 user message 发出。ref 包一层确保每次都拿最新 sendMessage。
  const sendMessageRef = useRef(sendMessage);
  useEffect(() => {
    sendMessageRef.current = sendMessage;
  });
  const answerQuestion = useCallback((value: string) => {
    void sendMessageRef.current(value);
  }, []);

  return {
    active,
    pending,
    sidebarConversations,
    sidebarActiveId,
    selectConversation,
    sendMessage,
    editMessage,
    stop,
    newConversation,
    deleteActive,
    projects,
    focusedProjectId,
    activeProject: visibleActiveProject,
    focusedOrActiveProject,
    createProject,
    renameProject,
    archiveProject,
    pickProject,
    pendingQuestion,
    answerQuestion,
  };
}
