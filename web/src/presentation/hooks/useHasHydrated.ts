// 订阅 Zustand persist 的水合状态;SSR 渲染时返回 false,客户端水合完成后翻为 true。
import { useSyncExternalStore } from 'react';
import { useAuthStore } from '@/presentation/stores/authStore';

export function useHasHydrated() {
  return useSyncExternalStore(
    (notify) => useAuthStore.persist.onFinishHydration(notify),
    () => useAuthStore.persist.hasHydrated(),
    () => false
  );
}