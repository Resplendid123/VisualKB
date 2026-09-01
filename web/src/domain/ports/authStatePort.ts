// session 读写端口;由 zustand store 实现,在容器里注入,让 infra/http 不依赖 presentation。
import type { Session } from '@/domain/entities/session';

export interface AuthStatePort {
  setSession(session: Session): void;
  clear(): void;
}