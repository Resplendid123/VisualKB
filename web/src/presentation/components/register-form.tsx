'use client';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useAuthAction } from '@/presentation/hooks/useAuthAction';

export default function RegisterForm() {
  const [state, formAction, isPending] = useAuthAction('register');

  return (
    <form action={formAction} className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="register-name">昵称</Label>
        <Input
          id="register-name"
          name="name"
          type="text"
          placeholder="想一个昵称"
          autoComplete="nickname"
          maxLength={64}
          required
          disabled={isPending}
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="register-email">邮箱</Label>
        <Input
          id="register-email"
          name="email"
          type="email"
          placeholder="you@example.com"
          autoComplete="email"
          maxLength={128}
          required
          disabled={isPending}
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="register-password">密码</Label>
        <Input
          id="register-password"
          name="password"
          type="password"
          placeholder="至少 6 位"
          autoComplete="new-password"
          minLength={6}
          maxLength={72}
          required
          disabled={isPending}
        />
      </div>

      {state.errorMessage && (
        <div
          role="alert"
          className="text-sm text-destructive bg-destructive/10 px-3 py-2 rounded-md"
        >
          {state.errorMessage}
        </div>
      )}

      <Button type="submit" disabled={isPending} className="mt-1">
        {isPending ? '注册中…' : '注册'}
      </Button>
    </form>
  );
}