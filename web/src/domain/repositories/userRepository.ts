import type { User, UserPortrait } from '@/domain/entities/user';

export interface UserRepository {
  create(input: { name: string; email: string }): Promise<User>;
  // GET /users/me/portrait — 读自己的 immutable + mutable;用户身份从 JWT 取。
  getMyPortrait(): Promise<UserPortrait>;
  // PUT /users/me/portrait/immutable — 改 immutable;后端回读再返回完整 portrait。
  updateMyImmutable(text: string): Promise<UserPortrait>;
  // PUT /users/me/portrait/mutable — 改 mutable;用户也能编辑 AI 记忆。
  updateMyMutable(text: string): Promise<UserPortrait>;
}