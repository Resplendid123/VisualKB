'use client';

import { useEffect, useState } from 'react';
import { CheckCircle2, XCircle } from 'lucide-react';
import { cn } from '@/lib/utils';

type ToastKind = 'success' | 'error';
type Toast = { id: number; kind: ToastKind; message: string };

const listeners = new Set<(toast: Toast) => void>();
let nextId = 1;

// 极简全局 toast — 不引第三方,够 push 一行提示即可。
export function showToast(message: string, kind: ToastKind = 'success') {
  const toast: Toast = { id: nextId++, kind, message };
  for (const fn of listeners) fn(toast);
}

// 单文件挂载点 — 在 root layout 渲染一次。
export function ToastHost() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  useEffect(() => {
    const onShow = (t: Toast) => {
      setToasts((prev) => [...prev, t]);
      window.setTimeout(() => {
        setToasts((prev) => prev.filter((x) => x.id !== t.id));
      }, 3500);
    };
    listeners.add(onShow);
    return () => {
      listeners.delete(onShow);
    };
  }, []);

  return (
    <div
      // role=status + aria-live=polite 让屏幕阅读器在变化时朗读;error 用 alert 抢优先级。
      className="fixed bottom-4 left-1/2 -translate-x-1/2 z-50 flex flex-col gap-2 pointer-events-none"
      role="status"
      aria-live="polite"
    >
      {toasts.map((t) => (
        <div
          key={t.id}
          role={t.kind === 'error' ? 'alert' : 'status'}
          className={cn(
            'pointer-events-auto rounded-md border px-3 py-2 text-xs shadow-sm flex items-center gap-2 bg-background animate-in fade-in-0 slide-in-from-bottom-2',
            t.kind === 'success'
              ? 'border-primary/30 text-foreground'
              : 'border-destructive/40 text-destructive'
          )}
        >
          {t.kind === 'success' ? (
            <CheckCircle2 className="h-3.5 w-3.5 text-primary" />
          ) : (
            <XCircle className="h-3.5 w-3.5" />
          )}
          <span>{t.message}</span>
        </div>
      ))}
    </div>
  );
}