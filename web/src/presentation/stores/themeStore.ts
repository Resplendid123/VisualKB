'use client';

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

export type ThemeMode = 'light' | 'dark' | 'system';

// 主题偏好:light / dark / system;持久化到 localStorage;首屏 .dark 由 THEME_BOOT_SCRIPT 同步写。
interface ThemeState {
  mode: ThemeMode;
  setMode: (m: ThemeMode) => void;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      mode: 'system',
      setMode: (mode) => {
        set({ mode });
        applyResolved(mode);
      },
    }),
    {
      name: 'learn-theme',
      storage: createJSONStorage(() => localStorage),
      partialize: (s) => ({ mode: s.mode }),
      onRehydrateStorage: () => (state) => {
        if (state) applyResolved(state.mode);
      },
    },
  ),
);

// 把 mode 解析成实际是不是深色,并写到 <html> 的 .dark class。跟 THEME_BOOT_SCRIPT 同步。
export function applyResolved(mode: ThemeMode): void {
  if (typeof document === 'undefined') return;
  const dark =
    mode === 'dark' ||
    (mode === 'system' &&
      typeof window !== 'undefined' &&
      window.matchMedia?.('(prefers-color-scheme: dark)').matches === true);
  document.documentElement.classList.toggle('dark', dark);
}

// 跟 applyResolved 同语义,但只返回布尔值 — 给 matchMedia 监听器调用。
export function resolveDark(mode: ThemeMode): boolean {
  if (typeof window === 'undefined') return false;
  return (
    mode === 'dark' ||
    (mode === 'system' && window.matchMedia?.('(prefers-color-scheme: dark)').matches === true)
  );
}

// 在 <head> 里同步执行的引导脚本:根据 localStorage + 系统偏好给 <html> 加 .dark。必须跟 applyResolved / resolveDark 同步。
export const THEME_BOOT_SCRIPT = `(function(){try{var r=localStorage.getItem('learn-theme');var m=r?JSON.parse(r).state.mode:'system';var d=m==='dark'||(m==='system'&&matchMedia('(prefers-color-scheme: dark)').matches);if(d)document.documentElement.classList.add('dark');}catch(e){}})();`;