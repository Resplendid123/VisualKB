import type { User } from '@/domain/entities/user';
import type { UserRepository } from '@/domain/repositories/userRepository';
import { ErrInvalidEmail, ErrInvalidName } from '@/domain/errors';

export class CreateUserUseCase {
  constructor(private readonly userRepo: UserRepository) {}

  async execute(input: { name: string; email: string }): Promise<User> {
    if (!input.name.trim()) throw ErrInvalidName();
    if (!input.email.includes('@')) throw ErrInvalidEmail();
    return this.userRepo.create(input);
  }
}