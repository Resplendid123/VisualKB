'use client';

import { Brain } from 'lucide-react';
import type { AssistantEvent, ToolCallView } from '@/domain/entities/streamEvent';
import { ToolCallBlock } from './tool-call-block';
import { stripThinkTag } from '@/lib/stripThinkTag';
import { MarkdownView } from '@/lib/markdown';

export type Block =
  | { kind: 'markdown'; text: string; key: string }
  | { kind: 'tool'; tool: ToolCallView; key: string }
  | { kind: 'reason'; content: string; key: string };

// 把 SSE events 聚成 markdown/tool/reason 三类 block,相邻同类型合并;切分逻辑给 bubble 复用。
export function buildBlocks(events: AssistantEvent[]): Block[] {
  const blocks: Block[] = [];
  let textBuf = '';
  let textKey = '';
  let reasonBuf = '';
  let reasonKey = '';
  let counter = 0;

  function flushText() {
    if (textBuf) {
      blocks.push({ kind: 'markdown', text: textBuf, key: textKey });
      textBuf = '';
      textKey = '';
    }
  }

  function flushReason() {
    if (reasonBuf) {
      blocks.push({ kind: 'reason', content: reasonBuf, key: reasonKey });
      reasonBuf = '';
      reasonKey = '';
    }
  }

  for (const ev of events) {
    if (ev.kind === 'text') {
      flushReason();
      if (!textBuf) textKey = `t-${counter++}`;
      textBuf += ev.content;
    } else if (ev.kind === 'reason') {
      flushText();
      if (!reasonBuf) reasonKey = `r-${counter++}`;
      reasonBuf += ev.content;
    } else {
      flushReason();
      flushText();
      blocks.push({ kind: 'tool', tool: ev.tool, key: `tl-${counter++}` });
    }
  }
  flushReason();
  flushText();
  return blocks;
}

// 切出"最终回复"=结尾那个 markdown block(若有)+ 其余全部可折叠;只认结尾一个,后面再接 tool/reason 说明本轮没结束。
export function splitBlocksForFold(blocks: Block[]): { preFold: Block[]; finalReply?: Block } {
  const last = blocks[blocks.length - 1];
  if (!last || last.kind !== 'markdown') {
    return { preFold: blocks };
  }
  return { preFold: blocks.slice(0, -1), finalReply: last };
}

// 渲染单个 block;fold / final reply 公用。defaultOpen 只对 reason 起作用(给流式最新思考卡片透传)。
export function RenderBlock({ block, defaultOpen }: { block: Block; defaultOpen?: boolean }) {
  if (block.kind === 'markdown') {
    return (
      <MarkdownView
        key={block.key}
        content={block.text}
        className="[&_p]:my-1 [&_ul]:my-1 [&_ol]:my-1 [&_li]:my-0 [&_h1]:text-base [&_h2]:text-sm [&_h3]:text-sm [&_code]:bg-background/50 [&_code]:px-1 [&_pre]:bg-background/50 [&_pre]:p-2 [&_pre]:rounded"
      />
    );
  }
  if (block.kind === 'reason') {
    return <ReasonCard key={block.key} content={block.content} defaultOpen={defaultOpen} />;
  }
  return <ToolCallBlock key={block.key} tool={block.tool} />;
}

// LLM 思考过程(思内容)折成可折叠卡片;Brain 图标和工具卡片视觉上区分开。
function ReasonCard({
  content,
  defaultOpen,
}: {
  content: string;
  defaultOpen?: boolean;
}) {
  return (
    <details
      className="rounded-md border border-dashed border-muted-foreground/30 bg-muted/30 text-xs"
      open={defaultOpen || undefined}
    >
      <summary className="flex items-center gap-1.5 px-2.5 py-1.5 cursor-pointer select-none text-muted-foreground hover:text-foreground list-none [&::-webkit-details-marker]:hidden">
        <Brain className="h-3.5 w-3.5" />
        <span>思考过程</span>
      </summary>
      <div className="px-3 pb-2.5 pt-1 text-muted-foreground whitespace-pre-wrap break-words leading-relaxed">
        {stripThinkTag(content)}
      </div>
    </details>
  );
}
