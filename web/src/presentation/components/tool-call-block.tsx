'use client';

import {
  Brain,
  CheckCircle2,
  FilePen,
  FilePlus,
  FileText,
  Files,
  FolderPlus,
  FolderTree,
  HelpCircle,
  Loader2,
  Search,
  Terminal,
  Wrench,
  XCircle,
  type LucideIcon,
} from 'lucide-react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import type { ToolCallView, ToolResultView } from '@/domain/entities/streamEvent';

// 工具名 → lucide icon;查不到用 Wrench 兜底。跟后端 internal/infra/ai/tools/*_tool.go 的 ToolName 常量对齐。
const ICONS: Record<string, LucideIcon> = {
  bash: Terminal,
  search_kb: Search,
  write_memory: Brain,
  ask_user_tool: HelpCircle,
  create_project: FolderPlus,
  list_projects: FolderTree,
  create_document: FilePlus,
  read_document: FileText,
  edit_document: FilePen,
  list_documents: Files,
};

interface ToolCallBlockProps {
  tool: ToolCallView;
}

// 渲染单个 tool_call 卡片;bash 走 name 路由到 Terminal icon 但展示形态与其他 tool 一致。
export function ToolCallBlock({ tool }: ToolCallBlockProps) {
  const Icon = ICONS[tool.name] ?? Wrench;
  const statusIcon =
    tool.status === 'pending' ? (
      <Loader2 className="h-4 w-4 animate-spin text-muted-foreground shrink-0" />
    ) : tool.status === 'error' ? (
      <XCircle className="h-4 w-4 text-red-500 shrink-0" />
    ) : (
      <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
    );

  const argsText = JSON.stringify(tool.args, null, 2);
  const hasOutput = tool.toolResult !== undefined;

  return (
    <Card
      size="sm"
      data-slot="tool-call-block"
      className="bg-muted/40 border-muted my-2 ring-foreground/5 not-prose"
    >
      <CardHeader className="grid-cols-[auto_1fr] gap-x-2 items-center">
        {statusIcon}
        <div className="flex flex-col gap-0.5 min-w-0">
          <CardTitle className="text-sm font-medium flex items-center gap-1.5">
            <Icon className="h-3.5 w-3.5 text-muted-foreground" />
            {tool.name}
          </CardTitle>
          {tool.description && (
            <CardDescription className="text-xs leading-snug">
              {tool.description}
            </CardDescription>
          )}
        </div>
      </CardHeader>

      {(argsText !== '{}' || hasOutput) && (
        <CardContent className="pt-0 space-y-2">
          {argsText !== '{}' && (
            <details className="text-xs">
              <summary className="cursor-pointer text-muted-foreground hover:text-foreground select-none">
                args
              </summary>
              <pre className="mt-1 rounded bg-background/60 p-2 overflow-x-auto text-[11px] leading-relaxed min-w-0">
                {argsText}
              </pre>
            </details>
          )}
          {hasOutput && <OutputSection result={tool.toolResult!} />}
        </CardContent>
      )}
    </Card>
  );
}

function OutputSection({ result }: { result: ToolResultView }) {
  // error 非空优先展示;否则把 result 当任意 JSON 序列化展示
  const failed = !!result.error;
  const text = failed
    ? result.error!
    : JSON.stringify(result.result ?? null, null, 2);
  return (
    <details className="text-xs">
      <summary className="cursor-pointer text-muted-foreground hover:text-foreground select-none">
        {failed ? 'error' : 'output'}
      </summary>
      <pre
        className={`mt-1 rounded p-2 overflow-x-auto text-[11px] leading-relaxed whitespace-pre-wrap min-w-0 ${
          failed
            ? 'bg-red-500/5 ring-1 ring-red-500/20 text-red-600 dark:text-red-400'
            : 'bg-background/60'
        }`}
      >
        {text}
      </pre>
    </details>
  );
}