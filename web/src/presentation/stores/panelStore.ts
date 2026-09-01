'use client';

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { Document } from '@/domain/entities/document';

export type PanelMode = 'default' | 'file' | 'terminal' | 'live';

interface PanelState {
  width: number;
  mode: PanelMode;
  open: boolean;
  // 当前预览面板要渲染的文档 id;不持久化(会话级)。
  previewDocId: number | null;
  // 用户通过 @ 选中的文档清单 — 同时是预览面板可切换的列表;send 时清空,不持久化。
  attachedDocs: Document[];
}

interface PanelActions {
  setWidth: (w: number) => void;
  setMode: (mode: PanelMode) => void;
  toggle: () => void;
  setOpen: (open: boolean) => void;
  setPreviewDocId: (id: number | null) => void;
  addAttachedDoc: (doc: Document) => void;
  removeAttachedDoc: (id: number) => void;
  clearAttachedDocs: () => void;
}

const PANEL_MIN_WIDTH = 360;
const PANEL_MAX_WIDTH = 720;
const PANEL_DEFAULT_WIDTH = 480;

const clampWidth = (w: number) =>
  Math.min(PANEL_MAX_WIDTH, Math.max(PANEL_MIN_WIDTH, w));

// 右栏(对话 / 终端 / 预览)状态;open 控制是否显示。
export const usePanelStore = create<PanelState & PanelActions>()(
  persist(
    (set) => ({
      width: PANEL_DEFAULT_WIDTH,
      mode: 'default',
      open: false,
      previewDocId: null,
      attachedDocs: [],

      setWidth: (width) => set({ width: clampWidth(width) }),
      setMode: (mode) => set({ mode }),
      toggle: () => set((s) => ({ open: !s.open })),
      setOpen: (open) => set({ open }),
      setPreviewDocId: (id) => set({ previewDocId: id }),
      addAttachedDoc: (doc) =>
        set((s) =>
          s.attachedDocs.some((d) => d.id === doc.id)
            ? s
            : { attachedDocs: [...s.attachedDocs, doc] }
        ),
      removeAttachedDoc: (id) =>
        set((s) => {
          const next = s.attachedDocs.filter((d) => d.id !== id);
          // 删除的是当前预览项 → 自动跳到下一项;清空则收起预览。
          let previewDocId = s.previewDocId;
          if (previewDocId === id) {
            previewDocId = next.length > 0 ? next[0].id : null;
          }
          return { attachedDocs: next, previewDocId };
        }),
      clearAttachedDocs: () => set({ attachedDocs: [] }),
    }),
    {
      name: 'learn.panel',
      storage: createJSONStorage(() => window.sessionStorage),
      partialize: (state) => ({ mode: state.mode, open: state.open }),
    }
  )
);