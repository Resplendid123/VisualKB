import UserForm from '@/presentation/components/user-form';
import AuthGuard from '@/presentation/components/auth-guard';

export default function UsersPage() {
    return (
        <AuthGuard>
            <main className="p-8">
                <h1 className="text-2xl font-bold mb-4">用户管理</h1>
                <UserForm />
            </main>
        </AuthGuard>
    );
}