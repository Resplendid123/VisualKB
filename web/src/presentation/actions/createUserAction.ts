'use server';

import { revalidatePath } from 'next/cache';
import { HttpUserRepository } from '@/infra/http/UserRepository';
import { API_BASE } from '@/infra/http/AuthClient';
import { CreateUserUseCase } from '@/application/usecases/user/createUser';
import { DomainError } from '@/domain/errors';

export type CreateUserState = {
  ok: boolean;
  errorMessage: string | null;
};

const INITIAL_STATE: CreateUserState = { ok: false, errorMessage: null };

// server action 没有 window,直接 raw fetch 把 cookie 透传过去。
const serverFetch: typeof fetch = (path, init) =>
  fetch(`${API_BASE}${path}`, { ...(init ?? {}), credentials: 'include' });

const createUser = new CreateUserUseCase(new HttpUserRepository(serverFetch));

export async function createUserAction(
  _prev: CreateUserState,
  formData: FormData
): Promise<CreateUserState> {
  const name = formData.get('name');
  const email = formData.get('email');

  if (typeof name !== 'string' || typeof email !== 'string') {
    return { ok: false, errorMessage: '表单数据格式错误' };
  }

  try {
    await createUser.execute({ name, email });
    revalidatePath('/admin');
    return { ok: true, errorMessage: null };
  } catch (e) {
    const message =
      e instanceof DomainError ? e.message : (e as Error).message;
    return { ok: false, errorMessage: message };
  }
}

export { INITIAL_STATE as CREATE_USER_INITIAL_STATE };
