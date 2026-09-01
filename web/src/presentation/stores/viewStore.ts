'use client';

import { create } from 'zustand';
import type { Document } from '@/domain/entities/document';

export type AppView = 'chat' | 'notes' | 'knowledge' | 'settings';

// 触发 NotesPanel 直接以这个 docId 打开编辑器;消费后清空。from 标记来源(chat/notes),决定按钮文案。
export type NoteEditingPayload =
  | { id: number; from: 'chat' }
  | { id: number; from: 'notes' };

// 笔记编辑器请求 chat 侧的 one-shot 操作;ChatLayout 在切到 chat 视图时一次性消费并清空。
export type ChatIntent = { kind: 'new-conversation'; doc: Document } | null;

interface ViewState {
  view: AppView;
  setView: (v: AppView) => void;
  noteEditingDocId: NoteEditingPayload | null;
  setNoteEditingDocId: (p: NoteEditingPayload | null) => void;
  pendingChatIntent: ChatIntent;
  setPendingChatIntent: (i: ChatIntent) => void;
  // 知识库视图当前选中的文档 id;选中后 tree 里这一行高亮。仅 knowledge view 关心。
  knowledgeSelectedDocId: number | null;
  setKnowledgeSelectedDocId: (id: number | null) => void;
  // 知识库视图已展开的目录 id;true=展开。跨视图切换保留。
  knowledgeExpandedFolderIds: Record<number, boolean>;
  setKnowledgeFolderExpanded: (id: number, open: boolean) => void;
  // 删除/重建后清空整个展开记录。
  resetKnowledgeExpanded: () => void;
}

// 顶层视图:左栏入口切换;不持久化,刷新回默认 chat。
export const useViewStore = create<ViewState>((set) => ({
  view: 'chat',
  noteEditingDocId: null,
  pendingChatIntent: null,
  knowledgeSelectedDocId: null,
  knowledgeExpandedFolderIds: {},
  setView: (view) => set({ view }),
  setNoteEditingDocId: (noteEditingDocId) => set({ noteEditingDocId }),
  setPendingChatIntent: (pendingChatIntent) => set({ pendingChatIntent }),
  setKnowledgeSelectedDocId: (knowledgeSelectedDocId) =>
    set({ knowledgeSelectedDocId }),
  setKnowledgeFolderExpanded: (id, open) =>
    set((s) => ({
      knowledgeExpandedFolderIds: { ...s.knowledgeExpandedFolderIds, [id]: open },
    })),
  resetKnowledgeExpanded: () => set({ knowledgeExpandedFolderIds: {} }),
}));