'use client';

import { useEffect, useState } from 'react';
import { LogOut, User as UserIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { useAuthStore } from '@/presentation/stores/authStore';
import { authUseCases } from '@/application/authContainer';
import { userUseCases } from '@/application/userContainer';
import { useThemeStore, type ThemeMode } from '@/presentation/stores/themeStore';
import { useAsyncState } from '@/presentation/hooks/useAsyncState';
import { cn } from '@/lib/utils';

// 设置视图:账号 + 用户画像 + 会话登出;左栏不动,右侧整片是本组件。
export function SettingsPanel() {
  const session = useAuthStore((s) => s.session);
  const setSession = useAuthStore((s) => s.setSession);
  const [loggingOut, setLoggingOut] = useState(false);

  async function logout() {
    if (loggingOut) return;
    setLoggingOut(true);
    try {
      // 先调后端清 cookie(失败也不阻断本地清理,见 HttpAuthRepository.logout)。
      await authUseCases.logout.execute();
      setSession(null);
      window.location.href = '/login';
    } finally {
      setLoggingOut(false);
    }
  }

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="max-w-xl mx-auto px-6 py-10 space-y-6">
        <header className="space-y-1">
          <h2 className="text-lg font-semibold">设置</h2>
          <p className="text-xs text-muted-foreground">外观、账号与会话信息。</p>
        </header>

        <ThemeSection />

        <section className="rounded-md border bg-muted/30 px-4 py-3 flex items-center gap-3">
          <div className="h-10 w-10 rounded-full bg-background border flex items-center justify-center text-muted-foreground shrink-0">
            <UserIcon className="h-4 w-4" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="text-sm font-medium truncate">
              {session?.user.name ?? '未登录'}
            </div>
            <div className="text-xs text-muted-foreground truncate">
              {session?.user.email ?? ''}
            </div>
          </div>
        </section>

        <PortraitSection />

        <section className="space-y-3">
          <div className="flex justify-end">
            <Button
              type="button"
              variant="destructive"
              onClick={() => void logout()}
              disabled={loggingOut}
            >
              <LogOut className="h-4 w-4" />
              {loggingOut ? '登出中…' : '登出'}
            </Button>
          </div>
        </section>
      </div>
    </div>
  );
}

const PORTRAIT_MAX = 4000;

const THEME_OPTIONS: { value: ThemeMode; label: string }[] = [
  { value: 'light', label: '浅色' },
  { value: 'dark', label: '深色' },
  { value: 'system', label: '跟随系统' },
];

// 主题切换:浅色/深色/跟随系统;system 模式的 OS 监听在 useSystemThemeSync 全局挂。
function ThemeSection() {
  const mode = useThemeStore((s) => s.mode);
  const setMode = useThemeStore((s) => s.setMode);

  return (
    <section className="space-y-3">
      <div>
        <div className="text-sm font-medium">主题</div>
        <p className="text-xs text-muted-foreground mt-1 leading-relaxed">
          切换浅色 / 深色,或跟随系统。立刻生效。
        </p>
      </div>
      <div
        role="radiogroup"
        aria-label="主题"
        className="inline-flex rounded-md border bg-muted/40 p-0.5"
      >
        {THEME_OPTIONS.map((opt) => {
          const active = mode === opt.value;
          return (
            <button
              key={opt.value}
              type="button"
              role="radio"
              aria-checked={active}
              onClick={() => setMode(opt.value)}
              className={cn(
                'px-3 h-8 text-xs rounded-[5px] transition-colors',
                active
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {opt.label}
            </button>
          );
        })}
      </div>
    </section>
  );
}

// 画像编辑:immutable 用户主写,mutable agent 用 write_memory 追加且用户可覆盖;两段独立 dirty/saving。
function PortraitSection() {
  const { data: portrait, loading } = useAsyncState(
    () => userUseCases.getMyPortrait.execute(),
    []
  );
  const [immutable, setImmutable] = useState('');
  const [savedImmutable, setSavedImmutable] = useState('');
  const [mutable, setMutable] = useState('');
  const [savedMutable, setSavedMutable] = useState('');
  const [savingImmutable, setSavingImmutable] = useState(false);
  const [savingMutable, setSavingMutable] = useState(false);
  const [errorImmutable, setErrorImmutable] = useState<string | null>(null);
  const [errorMutable, setErrorMutable] = useState<string | null>(null);

  // 画像拉回来时一次性搬到本地可编辑态 + 同步基线。
  useEffect(() => {
    if (!portrait) return;
    setImmutable(portrait.immutable);
    setSavedImmutable(portrait.immutable);
    setMutable(portrait.mutable);
    setSavedMutable(portrait.mutable);
  }, [portrait]);

  const dirtyImmutable = immutable !== savedImmutable;
  const overImmutable = immutable.length > PORTRAIT_MAX;
  const dirtyMutable = mutable !== savedMutable;
  const overMutable = mutable.length > PORTRAIT_MAX;

  async function saveImmutable() {
    if (!dirtyImmutable || overImmutable || savingImmutable) return;
    setSavingImmutable(true);
    setErrorImmutable(null);
    try {
      const p = await userUseCases.updateMyImmutable.execute(immutable);
      setSavedImmutable(p.immutable);
      setMutable(p.mutable);
      setSavedMutable(p.mutable);
    } catch (e) {
      setErrorImmutable((e as Error).message);
    } finally {
      setSavingImmutable(false);
    }
  }

  async function saveMutable() {
    if (!dirtyMutable || overMutable || savingMutable) return;
    setSavingMutable(true);
    setErrorMutable(null);
    try {
      const p = await userUseCases.updateMyMutable.execute(mutable);
      setSavedMutable(p.mutable);
      setImmutable(p.immutable);
      setSavedImmutable(p.immutable);
    } catch (e) {
      setErrorMutable((e as Error).message);
    } finally {
      setSavingMutable(false);
    }
  }

  return (
    <section className="space-y-5">
      <div>
        <div className="text-sm font-medium">用户画像</div>
        <p className="text-xs text-muted-foreground mt-1 leading-relaxed">
          只对你可见。每次新会话开始,这两段会拼到系统提示词里 ——
          "我的偏好"是稳定身份/指令,"AI 记忆"是 agent 自己观察到的内容。
        </p>
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <label className="text-xs text-muted-foreground">我的偏好</label>
        </div>
        <Textarea
          value={immutable}
          onChange={(e) => setImmutable(e.target.value)}
          rows={10}
          placeholder="比如:只用中文回复;我是后端开发;回答时尽量给代码示例…"
          disabled={loading || savingImmutable}
          className="resize-y min-h-40"
        />
        <div className="flex items-center justify-between text-[11px]">
          <span className={overImmutable ? 'text-destructive' : 'text-muted-foreground'}>
            {immutable.length} / {PORTRAIT_MAX}
          </span>
          {errorImmutable && (
            <span className="text-destructive truncate">{errorImmutable}</span>
          )}
        </div>
        <div className="flex justify-end">
          <Button
            type="button"
            size="sm"
            onClick={() => void saveImmutable()}
            disabled={!dirtyImmutable || overImmutable || savingImmutable}
          >
            {savingImmutable ? '保存中…' : '保存'}
          </Button>
        </div>
      </div>

      <div className="space-y-1.5">
        <label className="text-xs text-muted-foreground">AI 记忆</label>
        <Textarea
          value={mutable}
          onChange={(e) => setMutable(e.target.value)}
          rows={10}
          placeholder={loading ? '加载中…' : 'agent 会用 write_memory 工具在这里追加观察,你也可以直接覆盖编辑。'}
          disabled={loading || savingMutable}
          className="resize-y min-h-40"
        />
        <div className="flex items-center justify-between text-[11px]">
          <span className={overMutable ? 'text-destructive' : 'text-muted-foreground'}>
            {mutable.length} / {PORTRAIT_MAX}
          </span>
          {errorMutable && (
            <span className="text-destructive truncate">{errorMutable}</span>
          )}
        </div>
        <div className="flex justify-end">
          <Button
            type="button"
            size="sm"
            onClick={() => void saveMutable()}
            disabled={!dirtyMutable || overMutable || savingMutable}
          >
            {savingMutable ? '保存中…' : '保存'}
          </Button>
        </div>
      </div>
    </section>
  );
}