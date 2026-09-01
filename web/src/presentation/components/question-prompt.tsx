'use client';

import { useState } from 'react';
import { Send } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import type { QuestionPrompt as QuestionPromptData } from '@/domain/entities/streamEvent';

interface QuestionPromptProps {
  prompt: QuestionPromptData;
  onAnswer: (value: string) => void;
}

// ask_user_tool 推到前端的待答题目卡片 — 替换掉 ChatInput,选项前 [1] [2] 编号,
// 最后一行留空 input 给用户绕开预设选项自由回答。
export function QuestionPrompt({ prompt, onAnswer }: QuestionPromptProps) {
  const [freeText, setFreeText] = useState('');

  function submitFreeText() {
    const text = freeText.trim();
    if (!text) return;
    setFreeText('');
    onAnswer(text);
  }

  return (
    <div className="max-w-2xl mx-auto px-4 pb-4">
      <Card className="rounded-2xl border bg-background shadow-sm gap-0 overflow-hidden">
        <div className="px-4 pt-3 pb-2">
          <p className="text-sm font-medium leading-relaxed">{prompt.question}</p>
        </div>

        <div className="flex flex-col p-1.5">
          {prompt.options.map((opt, i) => (
            <button
              key={opt.label}
              type="button"
              onClick={() => onAnswer(opt.label)}
              className="text-left rounded-lg px-3 py-2 hover:bg-muted transition-colors group"
            >
              <div className="flex items-baseline gap-2 min-w-0">
                <span className="text-xs font-mono text-muted-foreground group-hover:text-foreground shrink-0 tabular-nums w-5 inline-block">
                  [{i + 1}]
                </span>
                <span className="text-sm font-medium">{opt.label}</span>
              </div>
              <p className="text-xs text-muted-foreground ml-7 mt-0.5 leading-relaxed">
                {opt.desc}
              </p>
            </button>
          ))}
        </div>

        <form
          className="flex items-center gap-2 px-3 py-2"
          onSubmit={(e) => {
            e.preventDefault();
            submitFreeText();
          }}
        >
          <span className="text-xs font-mono text-muted-foreground shrink-0 tabular-nums w-5 inline-block">
            [✎]
          </span>
          <Input
            value={freeText}
            onChange={(e) => setFreeText(e.target.value)}
            placeholder="或者直接输入回答…"
            className="h-8 border-0 shadow-none focus-visible:ring-0 bg-transparent px-0"
          />
          <Button
            type="submit"
            size="icon-sm"
            variant={freeText.trim() ? 'default' : 'ghost'}
            disabled={freeText.trim().length === 0}
            aria-label="发送"
          >
            <Send className="h-3.5 w-3.5" />
          </Button>
        </form>
      </Card>
    </div>
  );
}