import Link from 'next/link';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import LoginForm from '@/presentation/components/login-form';
import RegisterForm from '@/presentation/components/register-form';

type AuthMode = 'login' | 'register';

interface AuthCardProps {
  mode: AuthMode;
}

// 上方不做 Tab 切换,改用底部链接在 /login、/register 之间互跳,业务边界更清晰。
export default function AuthCard({ mode }: AuthCardProps) {
  const isLogin = mode === 'login';

  return (
    <Card className="w-full max-w-sm">
      <CardHeader>
        <CardTitle className="text-2xl">
          {isLogin ? '欢迎回来' : '创建账号'}
        </CardTitle>
        <CardDescription>
          {isLogin ? '使用邮箱登录你的账号' : '填写信息开始使用'}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col gap-6">
        {isLogin ? <LoginForm /> : <RegisterForm />}

        <p className="text-center text-sm text-muted-foreground">
          {isLogin ? (
            <>
              还没有账号？
              <Link
                href="/register"
                className="ml-1 text-foreground font-medium underline-offset-4 hover:underline"
              >
                去注册
              </Link>
            </>
          ) : (
            <>
              已有账号？
              <Link
                href="/login"
                className="ml-1 text-foreground font-medium underline-offset-4 hover:underline"
              >
                去登录
              </Link>
            </>
          )}
        </p>
      </CardContent>
    </Card>
  );
}