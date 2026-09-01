import { useAuthStore } from '@/presentation/stores/authStore';
import { HttpAuthRepository } from '@/infra/http/AuthRepository';
import { createAuthClient } from '@/infra/http/AuthClient';
import { LoginUseCase } from '@/application/usecases/auth/login';
import { RegisterUseCase } from '@/application/usecases/auth/register';
import { LogoutUseCase } from '@/application/usecases/auth/logout';

import type { AuthStatePort } from '@/domain/ports/authStatePort';

// 注入 zustand store,让 infra/http 不反向依赖 presentation。
const authStatePort: AuthStatePort = {
  setSession: (session) => useAuthStore.getState().setSession(session),
  clear: () => useAuthStore.getState().setSession(null),
};

const authedFetch = createAuthClient(authStatePort);
const authRepo = new HttpAuthRepository();

export const authUseCases = {
  login: new LoginUseCase(authRepo),
  register: new RegisterUseCase(authRepo),
  logout: new LogoutUseCase(authRepo),
};

export { authedFetch };