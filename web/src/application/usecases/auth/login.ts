import type { AuthRepository } from '@/domain/repositories/authRepository';
import type { Session } from '@/domain/entities/session';
import { ErrInvalidCredentials, ErrInvalidEmail } from '@/domain/errors/auth';

export class LoginUseCase {
  constructor(private readonly authRepo: AuthRepository) {}

  async execute(input: { email: string; password: string }): Promise<Session> {
    const email = input.email.trim();
    if (!email.includes('@')) throw ErrInvalidEmail();
    if (!input.password) throw ErrInvalidCredentials();
    return this.authRepo.login({ email, password: input.password });
  }
}