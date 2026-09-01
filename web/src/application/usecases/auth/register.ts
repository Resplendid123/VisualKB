import type { AuthRepository } from '@/domain/repositories/authRepository';
import type { Session } from '@/domain/entities/session';
import { ErrInvalidEmail, ErrInvalidName, ErrInvalidPassword } from '@/domain/errors/auth';

export class RegisterUseCase {
  constructor(private readonly authRepo: AuthRepository) {}

  async execute(input: { name: string; email: string; password: string }): Promise<Session> {
    const name = input.name.trim();
    const email = input.email.trim();
    if (!name) throw ErrInvalidName();
    if (!email.includes('@')) throw ErrInvalidEmail();
    if (input.password.length < 6) throw ErrInvalidPassword();
    return this.authRepo.register({ name, email, password: input.password });
  }
}