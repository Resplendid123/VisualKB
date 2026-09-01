import type { AuthStatePort } from '@/domain/ports/authStatePort';
import type { DomainError } from '@/domain/errors';

function resolveApiBase(): string {
  const fromEnv = process.env.NEXT_PUBLIC_API_URL;
  if (fromEnv && fromEnv.length > 0) return fromEnv;
  // 生产缺失 URL 说明部署配错,抛错让 CI 立即失败,避免静默指向 localhost:8889。
  if (process.env.NODE_ENV === 'production') {
    throw new Error('NEXT_PUBLIC_API_URL is required in production');
  }
  return 'http://localhost:8889';
}

export const API_BASE = resolveApiBase();

export interface ApiEnvelope<T> {
  code: number;
  message: string;
  data?: T | null;
}

export interface AuthedFetchOptions {
  skipAuthRedirect?: boolean;
}

// 401 触发一次 refresh + 重试,失败由 port 清 session 并跳 /login。
export function createAuthClient(port: AuthStatePort) {
  let inflightRefresh: Promise<boolean> | null = null;

  async function refreshSession(): Promise<boolean> {
    if (inflightRefresh) return inflightRefresh;
    inflightRefresh = (async () => {
      try {
        const res = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
          method: 'POST',
          credentials: 'include',
        });
        if (!res.ok) {
          port.clear();
          return false;
        }
        const envelope = (await res.json()) as ApiEnvelope<{
          access_token: string;
          user: { id: number; name: string; email: string };
        }>;
        if (envelope.code !== 0 || !envelope.data) {
          port.clear();
          return false;
        }
        port.setSession({ user: envelope.data.user });
        return true;
      } catch {
        port.clear();
        return false;
      } finally {
        inflightRefresh = null;
      }
    })();
    return inflightRefresh;
  }

  return async function authedFetch(
    path: string,
    init: RequestInit = {},
    options: AuthedFetchOptions = {}
  ): Promise<Response> {
    const headers = new Headers(init.headers);
    // FormData 的 boundary 由浏览器生成,传 Content-Type 会带错的 boundary。
    if (init.body && !headers.has('Content-Type') && !(init.body instanceof FormData)) {
      headers.set('Content-Type', 'application/json');
    }

    const res = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers,
      credentials: 'include',
    });

    if (res.status !== 401) return res;

    const ok = await refreshSession();
    if (!ok) {
      if (!options.skipAuthRedirect) {
        const here = window.location.pathname;
        if (here !== '/login' && here !== '/register') {
          window.location.href = '/login';
        }
      }
      return res;
    }

    return fetch(`${API_BASE}${path}`, {
      ...init,
      headers,
      credentials: 'include',
    });
  };
}

// 显式类型便于测试注入 fake。
export type AuthedFetch = (
  path: string,
  init?: RequestInit,
  options?: AuthedFetchOptions
) => Promise<Response>;

// 非 2xx / 非 JSON body / envelope code !== 0 都抛错;data=null + allowNullData 时返 null。
export async function parseEnvelope<T>(
  res: Response,
  what: string,
  mapError?: (code: number, message: string) => DomainError,
  allowNullData: boolean = false
): Promise<T> {
  if (res.status === 204) {
    if (allowNullData) return null as T;
    throw new Error(`${what} failed: HTTP 204 (empty body)`);
  }

  let env: ApiEnvelope<T> | null = null;
  try {
    env = (await res.json()) as ApiEnvelope<T>;
  } catch {
    throw new Error(`${what} failed: HTTP ${res.status} (non-JSON body)`);
  }

  if (!res.ok) {
    throw mapError
      ? mapError(env?.code ?? -1, env?.message ?? `HTTP ${res.status}`)
      : new Error(`${what} failed: ${env.message ?? `HTTP ${res.status}`}`);
  }
  if (env.code !== 0) {
    throw mapError
      ? mapError(env.code, env.message ?? `code ${env.code}`)
      : new Error(`${what} failed: ${env.message ?? `code ${env.code}`}`);
  }
  if (env.data === undefined || env.data === null) {
    if (allowNullData) return null as T;
    throw new Error(`${what} failed: empty data`);
  }
  return env.data;
}

// 204 / 2xx 都视为成功,其它抛错。
export async function expectNoContent(res: Response, what: string): Promise<void> {
  if (res.ok) return;
  const env = (await res.json().catch(() => null)) as { message?: string } | null;
  throw new Error(`${what} failed: ${env?.message ?? `HTTP ${res.status}`}`);
}