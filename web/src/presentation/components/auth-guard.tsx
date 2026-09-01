'use client';

import * as React from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/presentation/stores/authStore';
import { useHasHydrated } from '@/presentation/hooks/useHasHydrated';

interface AuthGuardProps {
  children: React.ReactNode;
  redirectTo?: string;
}

// 订阅水合态避免 SSR 把 session 误判为 null → 刷新即跳登录。
export default function AuthGuard({ children, redirectTo = '/login' }: AuthGuardProps) {
  const router = useRouter();
  const session = useAuthStore((s) => s.session);
  const hydrated = useHasHydrated();

  React.useEffect(() => {
    if (hydrated && !session) router.replace(redirectTo);
  }, [hydrated, session, router, redirectTo]);

  if (!hydrated || !session) {
    return (
      <div className="flex min-h-svh items-center justify-center text-sm text-muted-foreground">
        加载中…
      </div>
    );
  }

  return <>{children}</>;
}