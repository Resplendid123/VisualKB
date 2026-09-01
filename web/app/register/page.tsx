import AuthCard from '@/presentation/components/auth-card';
import RedirectIfAuthenticated from '@/presentation/components/redirect-if-authenticated';

export default function RegisterPage() {
  return (
    <RedirectIfAuthenticated>
      <main className="flex min-h-svh flex-1 items-center justify-center p-6">
        <AuthCard mode="register" />
      </main>
    </RedirectIfAuthenticated>
  );
}