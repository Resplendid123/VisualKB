'use client';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useAuthAction } from '@/presentation/hooks/useAuthAction';

export default function LoginForm() {
  const [state, formAction, isPending] = useAuthAction('login');

  return (
    <form action={formAction} className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="login-email">邮箱</Label>
        <Input
          id="login-email"
          name="email"
          type="email"
          placeholder="you@example.com"
          autoComplete="email"
          required
          disabled={isPending}
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="login-password">密码</Label>
        <Input
          id="login-password"
          name="password"
          type="password"
          placeholder="••••••"
          autoComplete="current-password"
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
        {isPending ? '登录中…' : '登录'}
      </Button>
    </form>
  );
}