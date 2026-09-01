import type { AuthRepository } from '@/domain/repositories/authRepository';

export class LogoutUseCase {
  constructor(private readonly authRepo: AuthRepository) {}

  async execute(): Promise<void> {
    await this.authRepo.logout();
  }
}