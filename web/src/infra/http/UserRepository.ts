import type { User, UserPortrait } from '@/domain/entities/user';
import type { UserRepository } from '@/domain/repositories/userRepository';
import { parseEnvelope, type AuthedFetch } from './AuthClient';

interface UserPayload {
  id: number;
  name: string;
  email: string;
}

interface PortraitPayload {
  immutable: string;
  mutable: string;
}

export class HttpUserRepository implements UserRepository {
  constructor(public readonly authedFetch: AuthedFetch) {}

  async create(input: { name: string; email: string }): Promise<User> {
    const res = await this.authedFetch('/api/v1/users', {
      method: 'POST',
      body: JSON.stringify(input),
    });
    const data = await parseEnvelope<UserPayload>(res, 'create user');
    return toUser(data);
  }

  async getMyPortrait(): Promise<UserPortrait> {
    const res = await this.authedFetch('/api/v1/users/me/portrait');
    const data = await parseEnvelope<PortraitPayload>(res, 'get my portrait');
    return toPortrait(data);
  }

  async updateMyImmutable(text: string): Promise<UserPortrait> {
    const res = await this.authedFetch('/api/v1/users/me/portrait/immutable', {
      method: 'PUT',
      body: JSON.stringify({ immutable: text }),
    });
    const data = await parseEnvelope<PortraitPayload>(res, 'update immutable portrait');
    return toPortrait(data);
  }

  async updateMyMutable(text: string): Promise<UserPortrait> {
    const res = await this.authedFetch('/api/v1/users/me/portrait/mutable', {
      method: 'PUT',
      body: JSON.stringify({ mutable: text }),
    });
    const data = await parseEnvelope<PortraitPayload>(res, 'update mutable portrait');
    return toPortrait(data);
  }
}

function toUser(p: UserPayload): User {
  return { id: p.id, name: p.name, email: p.email };
}

function toPortrait(p: PortraitPayload): UserPortrait {
  return { immutable: p?.immutable ?? '', mutable: p?.mutable ?? '' };
}
