'use client';

import { useCallback, useEffect, useState } from 'react';

// 跑一个 async 函数,管 loading/error/data + 重跑;deps 变化自动重跑。
export function useAsyncState<T>(
  fn: () => Promise<T>,
  deps: ReadonlyArray<unknown>
): { data: T | null; loading: boolean; error: string | null; reload: () => void } {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // tick +1 触发重跑 — 复用同一份 effect,不重写一遍。
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      if (cancelled) return;
      setLoading(true);
      setError(null);
      try {
        const v = await fn();
        if (cancelled) return;
        setData(v);
      } catch (e) {
        if (cancelled) return;
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick]);

  return { data, loading, error, reload: useCallback(() => setTick((t) => t + 1), []) };
}
