'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

// 滚动容器:贴底时跟随新内容 / 新依赖项自动滚底,用户主动上滚后松手不再强拉;
// 返 jumpToBottom 给「回到底部」按钮。
export function useStickyScroll<T extends HTMLElement>(
  // 内容变化依赖项(deps);任意一项变化都会在 stickToBottom=true 时滚底。
  deps: ReadonlyArray<unknown>,
  opts?: {
    threshold?: number;
    // 每次滚底前调一次:流式期间返 'auto'(无动画,逐帧不抖),否则 'smooth'。
    behavior?: () => ScrollBehavior;
  }
): {
  ref: React.RefObject<T | null>;
  stickToBottom: boolean;
  jumpToBottom: () => void;
  // 容器要绑的 onScroll — 滚动时重算 stickToBottom。
  onScroll: () => void;
} {
  const ref = useRef<T>(null);
  const [stickToBottom, setStickToBottom] = useState(true);
  const threshold = opts?.threshold ?? 64;
  const behaviorFn = opts?.behavior;

  const checkBottom = useCallback(() => {
    const el = ref.current;
    if (!el) return true;
    return el.scrollHeight - el.scrollTop - el.clientHeight <= threshold;
  }, [threshold]);

  const onScroll = useCallback(() => {
    setStickToBottom(checkBottom());
  }, [checkBottom]);

  useEffect(() => {
    if (!stickToBottom) return;
    ref.current?.scrollTo({
      top: ref.current.scrollHeight,
      behavior: behaviorFn ? behaviorFn() : 'auto',
    });
  }, deps); // eslint-disable-line react-hooks/exhaustive-deps

  const jumpToBottom = useCallback(() => {
    ref.current?.scrollTo({ top: ref.current.scrollHeight, behavior: 'smooth' });
    setStickToBottom(true);
  }, []);

  return { ref, stickToBottom, jumpToBottom: jumpToBottom as () => void, onScroll };
}
