'use client';

import { useActionState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  createUserAction,
  CREATE_USER_INITIAL_STATE,
} from '@/presentation/actions/createUserAction';

// 只负责 UI 渲染和事件分发;业务逻辑(createUserAction)和校验(在 use case 里)都不在这里。
export default function UserForm() {
  const [state, formAction, isPending] = useActionState(
    createUserAction,
    CREATE_USER_INITIAL_STATE
  );

  return (
    <form action={formAction} className="flex flex-col gap-3 max-w-sm">
      <Input
        name="name"
        required
        placeholder="姓名"
        disabled={isPending}
      />
      <Input
        name="email"
        type="email"
        required
        placeholder="邮箱"
        disabled={isPending}
      />

      {state.errorMessage && (
        <div
          role="alert"
          className="text-sm text-destructive bg-destructive/10 px-3 py-2 rounded"
        >
          {state.errorMessage}
        </div>
      )}

      <Button type="submit" disabled={isPending}>
        {isPending ? '创建中...' : '创建用户'}
      </Button>
    </form>
  );
}