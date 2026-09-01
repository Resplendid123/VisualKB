'use client';

import { Database, Users, AppWindow, Sparkles, type LucideIcon } from 'lucide-react';

interface Prompt {
  icon: LucideIcon;
  title: string;
  desc: string;
  text?: string;
  // true = 该卡片不进入对话,改走 onPick()(例如切到知识库视图)。
  pick?: boolean;
}

interface SuggestedPromptsProps {
  onSelect: (text: string) => void;
  // "切到别处"分支;pick=true 的卡片点击时调。
  onPick?: () => void;
}

const PROMPTS: Prompt[] = [
  // 知识库 ingest 直接跳视图 — 让空对话用户立刻看到自己的文档,而不是被拉进一条 LLM 长对话。
  {
    icon: Database,
    title: '知识库 ingest',
    desc: '把这个知识库 ingest 到 AI',
    pick: true,
  },
  {
    icon: Users,
    title: '笔记协作',
    desc: '一起协作编辑一篇笔记',
    text: '和我一起协作编辑一篇笔记',
  },
  {
    icon: AppWindow,
    title: '交互式项目',
    desc: '生成一个可交互的 demo 项目',
    text: '帮我生成一个可交互的 demo 项目',
  },
  {
    icon: Sparkles,
    title: '头脑风暴',
    desc: '给我 5 个学习建议',
    text: '给我 5 个学习建议',
  },
];

export function SuggestedPrompts({ onSelect, onPick }: SuggestedPromptsProps) {
  return (
    <div className="flex flex-col items-center justify-center h-full px-4">
      <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-4">
        <Sparkles className="h-6 w-6 text-muted-foreground" />
      </div>
      <h3 className="text-base font-medium mb-1">开始对话</h3>
      <p className="text-sm text-muted-foreground mb-6">试试下面的 prompt，或输入自己的问题</p>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 w-full max-w-md">
        {PROMPTS.map((p) => {
          const Icon = p.icon;
          return (
            <button
              key={p.title}
              onClick={() => {
                if (p.pick) {
                  onPick?.();
                  return;
                }
                if (p.text !== undefined) onSelect(p.text);
              }}
              className="text-left p-3 rounded-lg border bg-background hover:bg-muted transition-colors group"
            >
              <div className="flex items-center gap-2 mb-1">
                <Icon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
                <span className="text-sm font-medium">{p.title}</span>
              </div>
              <p className="text-xs text-muted-foreground">{p.desc}</p>
            </button>
          );
        })}
      </div>
    </div>
  );
}