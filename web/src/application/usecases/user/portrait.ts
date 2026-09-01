import type { UserPortrait } from '@/domain/entities/user';
import type { UserRepository } from '@/domain/repositories/userRepository';
import { ErrPortraitTooLong } from '@/domain/errors';

export class GetMyPortraitUseCase {
  constructor(private readonly userRepo: UserRepository) {}

  execute(): Promise<UserPortrait> {
    return this.userRepo.getMyPortrait();
  }
}

export class UpdateMyImmutableUseCase {
  constructor(private readonly userRepo: UserRepository) {}

  async execute(text: string): Promise<UserPortrait> {
    // 与后端 portraitFieldMaxLen 对齐,前端先拒掉让用户立即看到反馈。
    if (text.length > 4000) throw ErrPortraitTooLong();
    return this.userRepo.updateMyImmutable(text);
  }
}

export class UpdateMyMutableUseCase {
  constructor(private readonly userRepo: UserRepository) {}

  async execute(text: string): Promise<UserPortrait> {
    if (text.length > 4000) throw ErrPortraitTooLong();
    return this.userRepo.updateMyMutable(text);
  }
}