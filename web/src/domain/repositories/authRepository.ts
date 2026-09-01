// 认证仓储端口(由基础设施层实现);业务层不直接依赖 fetch/axios 等具体实现。
import type { Session } from '@/domain/entities/session';

export interface AuthRepository {
  register(input: { name: string; email: string; password: string }): Promise<Session>;
  login(input: { email: string; password: string }): Promise<Session>;
  // 保留签名让 mock 对齐;Http 实现由 authClient 内部走 cookie,此处不实现。
  refresh(rawToken: string): Promise<Session>;
  logout(): Promise<void>;
}