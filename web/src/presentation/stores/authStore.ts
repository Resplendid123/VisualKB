'use client';

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { Session } from '@/domain/entities/session';

interface AuthState {
  session: Session | null;
  setSession: (session: Session | null) => void;
}

// 仅持久化 session.user,刷新后保持登录态;access_token 由 HttpOnly cookie 持有。
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      session: null,
      setSession: (session) => set({ session }),
    }),
    {
      name: 'learn.auth',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ session: state.session }),
    }
  )
);