'use client';

import { useEffect } from 'react';
import { useThemeStore, applyResolved } from '@/presentation/stores/themeStore';

// 全局跟 OS 的 prefers-color-scheme 联动 — 必须在顶层 client 组件挂,否则切走 settings 后失效。
export function useSystemThemeSync() {
  const mode = useThemeStore((s) => s.mode);

  useEffect(() => {
    if (typeof window === 'undefined' || mode !== 'system') return;
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const onChange = () => applyResolved('system');
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, [mode]);
}
