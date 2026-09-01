'use client';

import { useActionState } from 'react';
import { useRouter } from 'next/navigation';
import { authUseCases } from '@/application/authContainer';
import { DomainError } from '@/domain/errors/auth';
import { useAuthStore } from '@/presentation/stores/authStore';

// 登录/注册共用 React 19 useActionState;业务调用在 use case,UI 只读 state / 派发 action。
export function useAuthAction(kind: 'login' | 'register') {
  const router = useRouter();
  const setSession = useAuthStore((s) => s.setSession);

  const action = async (_prev: AuthActionState, formData: FormData): Promise<AuthActionState> => {
    try {
      const session =
        kind === 'login'
          ? await authUseCases.login.execute({
              email: String(formData.get('email') ?? ''),
              password: String(formData.get('password') ?? ''),
            })
          : await authUseCases.register.execute({
              name: String(formData.get('name') ?? ''),
              email: String(formData.get('email') ?? ''),
              password: String(formData.get('password') ?? ''),
            });
      setSession(session);
      router.push('/');
      router.refresh();
      return INITIAL_STATE;
    } catch (err) {
      const message = err instanceof DomainError ? err.message : (err as Error).message;
      return { errorMessage: message };
    }
  };

  return useActionState(action, INITIAL_STATE);
}

export interface AuthActionState {
  errorMessage: string | null;
}

const INITIAL_STATE: AuthActionState = { errorMessage: null };