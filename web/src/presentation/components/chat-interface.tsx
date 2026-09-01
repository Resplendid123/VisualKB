'use client';

import { useState, useRef } from 'react';
import { ArrowDown } from 'lucide-react';
import { ChatHeader } from './chat-header';
import { MessageBubble } from './message-bubble';
import { SuggestedPrompts } from './suggested-prompts';
import { ChatInput } from './chat-input';
import { DocumentPicker } from './document-picker';
import { QuestionPrompt } from './question-prompt';
import type { Message } from '@/domain/entities/message';
import type { Project, ActiveProject } from '@/domain/entities/project';
import type { Document } from '@/domain/entities/document';
import type { QuestionPrompt as QuestionPromptData } from '@/domain/entities/streamEvent';
import { usePanelStore } from '@/presentation/stores/panelStore';
import { useViewStore } from '@/presentation/stores/viewStore';
import { useStickyScroll } from '@/presentation/hooks/useStickyScroll';
import { showToast } from './toast';

interface ChatInterfaceProps {
  title: string;
  messages: Message[];
  pending: boolean;
  onSend: (text: string, documentIds?: number[]) => Promise<boolean>;
  onStop: () => void;
  onEditMessage: (msgId: string, content: string) => void;
  onDeleteConversation: () => void;
  currentProject?: Project | ActiveProject | null;
  projects?: Project[];
  onPickProject?: (id: string) => void;
  onClearProject?: () => void;
  pendingQuestion?: QuestionPromptData | null;
  onAnswerQuestion?: (value: string) => void;
}

export function ChatInterface({
  title,
  messages,
  pending,
  onSend,
  onStop,
  onEditMessage,
  onDeleteConversation,
  currentProject,
  projects,
  onPickProject,
  onClearProject,
  pendingQuestion,
  onAnswerQuestion,
}: ChatInterfaceProps) {
  const [input, setInput] = useState('');
  const [pickerOpen, setPickerOpen] = useState(false);
  // 新消息 / 流式 chunk 到达:仅在粘附底部时自动滚下去,流式期间 'auto' 避免每个 chunk 都触发 smooth 卡顿。
  const { ref: scrollRef, stickToBottom, jumpToBottom, onScroll } =
    useStickyScroll<HTMLDivElement>(
      [messages, pending],
      { behavior: () => (pending ? 'auto' : 'smooth') }
    );
  const atButtonRef = useRef<HTMLButtonElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // 已 @ 选中的文档走 panelStore — preview 面板也要读这份数据做 prev/next 切换,
  // 所以提到 store 而不是本地 state。
  const selectedDocs = usePanelStore((s) => s.attachedDocs);
  const addAttachedDoc = usePanelStore((s) => s.addAttachedDoc);
  const removeAttachedDoc = usePanelStore((s) => s.removeAttachedDoc);
  const clearAttachedDocs = usePanelStore((s) => s.clearAttachedDocs);

  // 仅对最近一条用户消息展示编辑入口。
  const lastUserIndex = (() => {
    for (let i = messages.length - 1; i >= 0; i--) {
      if (messages[i].role === 'user') return i;
    }
    return -1;
  })();

  async function handleSend() {
    const text = input.trim();
    if (!text || pending) return;
    const docIds = selectedDocs.map((d) => d.id);
    // 成功才清 input / 附件 — 失败时文本和 @ 留在原位,用户重试不用重输。
    const ok = await onSend(text, docIds);
    if (!ok) {
      showToast('发送失败,会话创建未成功', 'error');
      return;
    }
    setInput('');
    clearAttachedDocs();
  }

  function handlePromptSelect(text: string) {
    setInput(text);
  }

  function handlePickDocument(doc: Document) {
    addAttachedDoc(doc);
    setPickerOpen(false);
    // 复刻 useConversations.dispatchStreamSideEffects 的 panel 联动模式:
    // 打开面板 + 切到文件预览 + 设预览 id,跟 bash 触发终端面板是一组操作。
    const panel = usePanelStore.getState();
    panel.setOpen(true);
    panel.setMode('file');
    panel.setPreviewDocId(doc.id);
  }

  function handleRemoveDocument(id: number) {
    removeAttachedDoc(id);
  }

  return (
    <div className="flex flex-col h-full">
      <ChatHeader
        title={title}
        hasActions={messages.length > 0}
        onDeleteConversation={onDeleteConversation}
      />

      <div
        ref={scrollRef}
        onScroll={onScroll}
        className="flex-1 overflow-y-auto px-6 py-4 space-y-4"
      >
        {messages.length === 0 ? (
          <SuggestedPrompts
            onSelect={handlePromptSelect}
            onPick={() => window.location.assign('/knowledge')}
          />
        ) : (
          messages.map((m, i) => (
            <MessageBubble
              key={m.id}
              message={m}
              editable={i === lastUserIndex}
              onEdit={
                i === lastUserIndex
                  ? (content) => onEditMessage(m.id, content)
                  : undefined
              }
              streaming={pending && i === messages.length - 1 && m.role !== 'system'}
            />
          ))
        )}
      </div>

      {!stickToBottom && (
        <div className="relative">
          <button
            type="button"
            onClick={jumpToBottom}
            aria-label="跳到底部"
            title="跳到底部"
            className="absolute -top-12 left-1/2 -translate-x-1/2 z-10 h-9 w-9 rounded-full bg-background border shadow-md flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          >
            <ArrowDown className="h-4 w-4" />
          </button>
        </div>
      )}

      <div className="relative">
        {pendingQuestion && onAnswerQuestion ? (
          <QuestionPrompt prompt={pendingQuestion} onAnswer={onAnswerQuestion} />
        ) : (
          <ChatInput
            value={input}
            onChange={setInput}
            onSend={handleSend}
            onStop={onStop}
            pending={pending}
            currentProject={currentProject}
            projects={projects}
            onPickProject={onPickProject}
            onClearProject={onClearProject}
            selectedDocuments={selectedDocs}
            atButtonRef={atButtonRef}
            containerRef={containerRef}
            onTogglePicker={() => setPickerOpen((v) => !v)}
            onRemoveDocument={handleRemoveDocument}
          />
        )}
        <DocumentPicker
          open={pickerOpen}
          onOpenChange={setPickerOpen}
          anchorRef={containerRef}
          onPick={handlePickDocument}
        />
      </div>
    </div>
  );
}