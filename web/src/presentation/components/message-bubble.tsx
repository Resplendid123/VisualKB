'use client';

import { useState } from 'react';
import { Check, ChevronRight, Copy, Pencil } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip';
import type { Message } from '@/domain/entities/message';
import {
  buildBlocks,
  splitBlocksForFold,
  RenderBlock,
} from './event-stream-view';
import { MarkdownView } from '@/lib/markdown';

interface MessageBubbleProps {
  message: Message;
  // 父级 (ChatInterface) 决定:当前对话里"最近一条用户消息"才传 true。
  editable?: boolean;
  onEdit?: (newContent: string) => void;
  // 流式生成中让思考卡片默认展开,流结束收起。
  streaming?: boolean;
}

export function MessageBubble({
  message,
  editable,
  onEdit,
  streaming,
}: MessageBubbleProps) {
  const isUser = message.role === 'user';
  const [copied, setCopied] = useState(false);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(message.content);
  // 派生而非用 effect 重置:父级把 editable 翻 false(又发新消息)时会自动退出编辑。
  const isEditing = editing && editable;
  // 流式期间默认展开,流结束自动收起;用户在 summary 上点开后保留用户状态。
  const [isOpen, setIsOpen] = useState<boolean>(!!streaming);
  const [userTouched, setUserTouched] = useState(false);
  const effectiveOpen = userTouched ? isOpen : !!streaming;
  function handleToggle(open: boolean) {
    setUserTouched(true);
    setIsOpen(open);
  }

  async function handleCopy() {
    await navigator.clipboard.writeText(message.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  function startEdit() {
    setDraft(message.content);
    setEditing(true);
  }

  // 空内容(含纯空白)不让提交 — 避免把消息替换成空白触发后端空校验。
  const trimmed = draft.trim();
  const canSubmit = trimmed.length > 0 && trimmed !== message.content;

  function sendEdit() {
    if (!canSubmit) return;
    onEdit?.(trimmed);
    setEditing(false);
  }

  function cancelEdit() {
    setDraft(message.content);
    setEditing(false);
  }

  function handleEditKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendEdit();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      cancelEdit();
    }
  }

  return (
    <div className={`flex gap-3 items-start ${isUser ? 'justify-end' : 'justify-start'}`}>
      {!isUser && (
        <Avatar className="h-8 w-8 shrink-0">
          <AvatarFallback className="bg-primary text-primary-foreground text-xs">
            AI
          </AvatarFallback>
        </Avatar>
      )}

      <div className={`flex flex-col gap-1 w-full max-w-3xl min-w-0 ${isUser ? 'items-end' : 'items-start'}`}>
        {isEditing ? (
          <div className="w-full min-w-72 max-w-xl rounded-2xl border bg-background px-3 py-2 shadow-sm focus-within:ring-2 focus-within:ring-ring/30 focus-within:border-ring">
            <Textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={handleEditKeyDown}
              autoFocus
              rows={3}
              className="border-0 shadow-none focus-visible:ring-0 p-0 bg-transparent resize-none"
            />
            <div className="mt-2 flex justify-end gap-2">
              <Button size="sm" variant="ghost" onClick={cancelEdit}>
                取消
              </Button>
              <Button size="sm" onClick={sendEdit} disabled={!canSubmit}>
                发送
              </Button>
            </div>
          </div>
        ) : (
          <div
            className={`px-4 py-2.5 rounded-2xl text-sm break-words max-w-full min-w-[10rem] overflow-hidden ${
              isUser
                ? 'bg-primary text-primary-foreground rounded-tr-sm'
                : 'bg-muted text-foreground rounded-tl-sm'
            }`}
          >
            {renderBubbleBody(message, streaming, effectiveOpen, handleToggle)}
          </div>
        )}

        {/* 时间+操作行:用户消息 flex-row-reverse,源码 time 在前 icon 在后,视觉上 icon 在 time 左边。 */}
        <div className={`flex items-center gap-1 px-1 text-xs text-muted-foreground ${isUser ? 'flex-row-reverse' : ''}`}>
          {isUser ? (
            <span>
              {new Date(message.created_at).toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
              })}
            </span>
          ) : (
            <>
              <span>{formatTime(message.created_at)}</span>
              {message.role === 'system' && message.content === '用户中止' && (
                <span className="px-1.5 py-0.5 rounded bg-muted text-muted-foreground text-[10px]">
                  已停止
                </span>
              )}
            </>
          )}

          {isUser && !isEditing && editable && onEdit && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6"
                    onClick={startEdit}
                    aria-label="编辑"
                  >
                    <Pencil className="h-3 w-3" />
                  </Button>
                }
              />
              <TooltipContent>编辑</TooltipContent>
            </Tooltip>
          )}

          {!isUser && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6"
                    onClick={handleCopy}
                    aria-label={copied ? '已复制' : '复制'}
                  >
                    {copied ? (
                      <Check className="h-3 w-3 text-green-500" />
                    ) : (
                      <Copy className="h-3 w-3" />
                    )}
                  </Button>
                }
              />
              <TooltipContent>{copied ? '已复制' : '复制'}</TooltipContent>
            </Tooltip>
          )}
        </div>
      </div>

      {isUser && (
        <Avatar className="h-8 w-8 shrink-0">
          <AvatarFallback className="bg-secondary text-xs">你</AvatarFallback>
        </Avatar>
      )}
    </div>
  );
}

function formatTime(ms: number): string {
  return new Date(ms).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  });
}

function Dots() {
  // 整个 turn 内的"还在工作"提示;接 done 之前一直挂在卡底。
  return (
    <span className="inline-flex gap-1 text-muted-foreground">
      <span className="animate-bounce">·</span>
      <span className="animate-bounce" style={{ animationDelay: '0.15s' }}>·</span>
      <span className="animate-bounce" style={{ animationDelay: '0.3s' }}>·</span>
    </span>
  );
}

function renderBubbleBody(
  message: Message,
  streaming: boolean | undefined,
  isOpen: boolean,
  onToggle: (v: boolean) => void,
) {
  if (message.role === 'user') {
    return <div className="whitespace-pre-wrap">{message.content}</div>;
  }

  const events = message.events ?? [];
  const hasContent = !!message.content && message.content.trim() !== '';
  const hasEvents = events.length > 0;

  // 整轮没收到任何内容:turn 还在跑就一直 dots,流结束才落到"(无回应)"兜底。
  if (!hasContent && !hasEvents) {
    if (streaming) return <Dots />;
    return <span className="text-muted-foreground text-xs italic">（无回应）</span>;
  }
  // 老历史(没 events、只有 content)走纯 markdown 路径,无 fold。
  if (!hasEvents) {
    return (
      <MarkdownView
        content={message.content!}
        className="[&_p]:my-1 [&_ul]:my-1 [&_ol]:my-1 [&_li]:my-0 [&_h1]:text-base [&_h2]:text-sm [&_h3]:text-sm [&_code]:bg-background/50 [&_code]:px-1 [&_pre]:bg-background/50 [&_pre]:p-2 [&_pre]:rounded"
      />
    );
  }

  // 通用:events → blocks → fold + final reply + bash inline + Dots
  const blocks = buildBlocks(events);
  const { preFold, finalReply } = splitBlocksForFold(blocks);
  const lastBlock = blocks[blocks.length - 1];
  // 流式时最后一个块是 reason → 在 fold 里默认展开,操作中的思维链跟上去。
  const activeReasonKey =
    streaming && lastBlock?.kind === 'reason' ? lastBlock.key : null;

  return (
    <div className="space-y-2 text-sm">
      {preFold.length > 0 && (
        <FoldSummary
          isOpen={isOpen}
          onToggle={onToggle}
          count={preFold.length}
        >
          {preFold.map((b) => (
            <RenderBlock
              key={b.key}
              block={b}
              defaultOpen={b.key === activeReasonKey}
            />
          ))}
        </FoldSummary>
      )}
      {finalReply && <RenderBlock block={finalReply} />}
      {streaming && <Dots />}
    </div>
  );
}

// 一个回合的"思考+工具调用"折成可折叠;bash 走侧栏 TerminalView,主对话不再 inline。
function FoldSummary({
  isOpen,
  onToggle,
  count,
  children,
}: {
  isOpen: boolean;
  onToggle: (v: boolean) => void;
  count: number;
  children: React.ReactNode;
}) {
  return (
    <details
      open={isOpen}
      onToggle={(e) => onToggle((e.currentTarget as HTMLDetailsElement).open)}
      className="rounded-md border border-dashed border-muted-foreground/30 bg-muted/30 text-xs"
    >
      <summary className="flex items-center gap-1.5 px-2.5 py-1.5 cursor-pointer select-none text-muted-foreground hover:text-foreground list-none [&::-webkit-details-marker]:hidden">
        <ChevronRight className="h-3.5 w-3.5 transition-transform [[open]_&]:rotate-90" />
        <span>{count} 个步骤</span>
      </summary>
      <div className="px-3 pb-2.5 pt-1 space-y-2 text-sm">{children}</div>
    </details>
  );
}
