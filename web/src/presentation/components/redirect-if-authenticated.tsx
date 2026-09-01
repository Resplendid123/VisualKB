'use client';

import * as React from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/presentation/stores/authStore';
import { useHasHydrated } from '@/presentation/hooks/useHasHydrated';

interface RedirectIfAuthenticatedProps {
  children: React.ReactNode;
  redirectTo?: string;
}

// 已登录用户访问 /login 或 /register 时送回首页,避免"登录态下还能看到登录表单"。
export default function RedirectIfAuthenticated({
  children,
  redirectTo = '/',
}: RedirectIfAuthenticatedProps) {
  const router = useRouter();
  const session = useAuthStore((s) => s.session);
  const hydrated = useHasHydrated();

  React.useEffect(() => {
    if (hydrated && session) router.replace(redirectTo);
  }, [hydrated, session, router, redirectTo]);

  if (!hydrated || session) {
    return (
      <div className="flex min-h-svh items-center justify-center text-sm text-muted-foreground">
        加载中…
      </div>
    );
  }

  return <>{children}</>;
}