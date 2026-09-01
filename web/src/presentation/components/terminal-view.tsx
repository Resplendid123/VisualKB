'use client';

import { useMemo } from 'react';
import { ArrowDown, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  useBashStreamStore,
  type BashEntry,
} from '@/presentation/stores/bashStreamStore';
import { useStickyScroll } from '@/presentation/hooks/useStickyScroll';

const PROMPT_HOST = 'learn@host';

// 一次 turn 的所有 bash 拼成一块;多 turn 按时间倒序堆叠。
export function TerminalView({ conversationId }: { conversationId: string }) {
  const allEntries = useBashStreamStore((s) => s.entries);
  const entries = useMemo(
    () => allEntries.filter((e) => e.call.conversationId === conversationId),
    [allEntries, conversationId]
  );

  const { ref: scrollRef, stickToBottom, jumpToBottom, onScroll } = useStickyScroll<HTMLDivElement>(
    [entries.length]
  );

  if (entries.length === 0) {
    return (
      <div className="px-4 py-6 text-xs text-muted-foreground">
        暂无 bash 调用。
        <br />
        当 LLM 跑 bash 时,命令和输出会自动出现在这里。
      </div>
    );
  }

  const groups = groupByTurn(entries);

  return (
    <div className="relative h-full">
      <div
        ref={scrollRef}
        onScroll={onScroll}
        className="h-full overflow-y-auto divide-y"
      >
        {groups.map((g) => (
          <TurnTerminal key={g.turnId} turnId={g.turnId} entries={g.entries} />
        ))}
      </div>

      {!stickToBottom && (
        <button
          type="button"
          onClick={jumpToBottom}
          aria-label="跳到底部"
          title="跳到底部"
          className="absolute left-1/2 -translate-x-1/2 bottom-4 z-10 h-9 w-9 rounded-full bg-background border shadow-md flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
        >
          <ArrowDown className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}

function TurnTerminal({
  turnId,
  entries,
}: {
  turnId: string;
  entries: BashEntry[];
}) {
  const startedAt = entries[0].call.startedAt;
  const allFinished = entries.every((e) => e.result);
  const anyFailed = entries.some((e) => e.result && e.result.error);

  const startTime = new Date(startedAt).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });

  return (
    <div className="px-4 py-3 space-y-1">
      <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground font-mono">
        <span>turn {turnId.slice(0, 6)}</span>
        <span>·</span>
        <span>{startTime}</span>
        <span>·</span>
        {allFinished ? (
          anyFailed ? (
            <span className="text-red-500">失败</span>
          ) : (
            <span className="text-green-600 dark:text-green-400">完成</span>
          )
        ) : (
          <span className="text-muted-foreground">运行中</span>
        )}
      </div>

      <div className="rounded-md bg-white dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 text-xs font-mono leading-relaxed overflow-hidden border border-zinc-200 dark:border-zinc-800">
        {entries.map((e) => (
          <CommandBlock key={e.call.toolCallId} entry={e} />
        ))}
        <div className="px-3 py-1.5 text-emerald-600 dark:text-emerald-400 select-none">
          <span>{PROMPT_HOST}</span>
          <span className="text-zinc-500 dark:text-zinc-400">$</span>
          <span className="ml-2 text-zinc-400 dark:text-zinc-500">▍</span>
        </div>
      </div>
    </div>
  );
}

function CommandBlock({ entry }: { entry: BashEntry }) {
  const { call, result } = entry;
  const running = !result;

  return (
    <div className="px-3 py-1">
      <div className="flex items-start gap-1.5">
        <span className="text-emerald-600 dark:text-emerald-400 select-none shrink-0">{PROMPT_HOST}</span>
        <span className="text-zinc-500 dark:text-zinc-400 select-none shrink-0">$</span>
        <span className="whitespace-pre-wrap break-all flex-1 text-zinc-900 dark:text-zinc-100">{call.command}</span>
        {running && (
          <Loader2 className="inline h-3 w-3 animate-spin align-text-bottom ml-1 text-zinc-400 dark:text-zinc-500 shrink-0" />
        )}
      </div>
      {(result?.content || result?.error) && (
        <pre
          className={cn(
            'whitespace-pre-wrap break-all text-[11px] mt-0.5',
            result.error
              ? 'text-red-600 dark:text-red-400'
              : 'text-zinc-700 dark:text-zinc-300'
          )}
        >
          {result.content}
          {result.error}
        </pre>
      )}
    </div>
  );
}

function groupByTurn(
  entries: BashEntry[]
): { turnId: string; entries: BashEntry[] }[] {
  const map = new Map<string, BashEntry[]>();
  for (const e of entries) {
    const arr = map.get(e.call.turnId);
    if (arr) arr.push(e);
    else map.set(e.call.turnId, [e]);
  }
  const groups = [...map.entries()].map(([turnId, list]) => {
    list.sort((a, b) => a.call.startedAt - b.call.startedAt);
    return { turnId, entries: list };
  });
  // turn 内升序;turn 间倒序(最新在上)
  groups.sort((a, b) => b.entries[0].call.startedAt - a.entries[0].call.startedAt);
  return groups;
}
