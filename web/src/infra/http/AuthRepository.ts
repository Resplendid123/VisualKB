import type { AuthRepository } from '@/domain/repositories/authRepository';
import type { Session } from '@/domain/entities/session';
import {
  DomainError,
  ErrAuthLoginFailed,
  ErrAuthRegisterFailed,
  mapAuthBackendError,
} from '@/domain/errors/auth';
import { API_BASE, parseEnvelope } from './AuthClient';

// access_token 由 HttpOnly cookie 持有,这里只是回传不存。
interface AuthPayload {
  access_token: string;
  user: { id: number; name: string; email: string };
}

export class HttpAuthRepository implements AuthRepository {
  async register(input: { name: string; email: string; password: string }): Promise<Session> {
    try {
      const data = await postAuth<AuthPayload>('/api/v1/auth/register', input);
      return toSession(data);
    } catch (e) {
      throw e instanceof DomainError ? e : ErrAuthRegisterFailed();
    }
  }

  async login(input: { email: string; password: string }): Promise<Session> {
    try {
      const data = await postAuth<AuthPayload>('/api/v1/auth/login', input);
      return toSession(data);
    } catch (e) {
      throw e instanceof DomainError ? e : ErrAuthLoginFailed();
    }
  }

  // refresh 由 authClient 内部走 cookie 发起,此处不实现。
  async refresh(): Promise<Session> {
    throw new Error('refresh is handled by authClient');
  }

  async logout(): Promise<void> {
    // 后端失败也要清本地 session,清理逻辑交给 use case。
    try {
      await postAuth<unknown>('/api/v1/auth/logout', {});
    } catch {}
  }
}

async function postAuth<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  return parseEnvelope<T>(res, path, mapAuthBackendError);
}

function toSession(data: AuthPayload): Session {
  return {
    user: {
      id: data.user.id,
      name: data.user.name,
      email: data.user.email,
    },
  };
}