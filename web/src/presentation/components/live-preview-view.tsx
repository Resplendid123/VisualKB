'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { AlertCircle, ExternalLink, Loader2, Radio, RefreshCcw } from 'lucide-react';
import { chatUseCases } from '@/application/chatContainer';
import type { Artifact } from '@/domain/entities/artifact';
import { useArtifactStreamStore } from '@/presentation/stores/artifactStreamStore';

// LivePreviewView — 嵌 iframe 显示 Agent 跑出来的 site。
//
// 数据流:
//   1) 优先 sandbox publicURL(controller stamp 的 CDN URL,pre/post build 都固定)
//   2) 没 sandbox URL 但有 build_artifact 记录 → 用 artifact.url(老路径,保留兼容)
//   3) 都没有 → 空态
//
// artifact 不主动轮询 — 等 SSE 流里 build_artifact 成功后由 store 推过来;
// 避免 mount 时打 /artifacts/latest 死路由(后端未注册)。
export function LivePreviewView({
  conversationId,
  // Sandbox-stamped CDN URL; "" pre-reconcile / 没有 active project。
  activeProjectPreviewUrl,
}: {
  conversationId: string;
  activeProjectPreviewUrl?: string;
}) {
  const [artifact, setArtifact] = useState<Artifact | null | undefined>(undefined);
  // iframe 重载触发器:build 出来的 url 通常带 ts,浏览器同源缓存仍可能命中
  // 旧的 index.html(虽然 Nginx Cache-Control: immutable),手动 reload 更稳。
  const [reloadKey, setReloadKey] = useState(0);

  const notifySeq = useArtifactStreamStore((s) => s.seq);
  const latestEvent = useArtifactStreamStore((s) => s.latest);

  const reload = useCallback(() => setReloadKey((k) => k + 1), []);

  // 没 sandbox URL + 有 convo 才回头走 artifact 链路;
  // SSE 推过来的 build_artifact event 直接覆盖 store → 这里 listener 触发刷新。
  useEffect(() => {
    if (!conversationId || activeProjectPreviewUrl) return;
    let cancelled = false;
    (async () => {
      try {
        const a = await chatUseCases.getLatestArtifact.execute(conversationId);
        if (cancelled) return;
        setArtifact(a ?? null);
      } catch (e) {
        if (cancelled) return;
        console.error('get latest artifact failed', e);
        setArtifact(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [conversationId, activeProjectPreviewUrl, notifySeq]);

  // 只在事件属于当前 conversation 时响应 — 切对话不串。
  const isCurrentConvo = latestEvent?.conversationId === conversationId;
  const sourceLabel = useMemo(() => {
    if (!artifact) return '';
    return artifact.framework ? artifact.framework : 'static';
  }, [artifact]);

  if (artifact === undefined) {
    return (
      <StatusHint>
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        加载最新 build artifact ...
      </StatusHint>
    );
  }

  if (artifact === null) {
    // 没 artifact 但 sandbox 已 stamp URL → 直接嵌 sandbox 预览路由。
    // URL 固定(非 build-gated);build 前是空/404,build 后渲站点。
    if (activeProjectPreviewUrl) {
      return (
        <div className="flex flex-col h-full">
          <HeaderBar conversationId={conversationId}>
            <span className="text-[10px] uppercase tracking-wider text-muted-foreground">
              sandbox
            </span>
            <a
              href={activeProjectPreviewUrl}
              target="_blank"
              rel="noreferrer"
              aria-label="open in new tab"
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              <ExternalLink className="h-3.5 w-3.5" />
            </a>
          </HeaderBar>
          <iframe
            key={`sandbox-${reloadKey}`}
            src={activeProjectPreviewUrl}
            title={`sandbox preview ${conversationId}`}
            sandbox="allow-scripts allow-same-origin allow-forms"
            className="flex-1 w-full border-0 bg-background"
          />
        </div>
      );
    }
    return (
      <div className="flex flex-col h-full">
        <HeaderBar conversationId={conversationId} />
        <StatusHint>
          {isCurrentConvo && latestEvent ? (
            'Agent 刚完成构建,等待产物写入 JuiceFS ...'
          ) : (
            <span>
              还没有 build artifact。让 agent 在对话里执行{' '}
              <code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded">
                build_artifact
              </code>
              ,静态构建产物 URL 就会出现在这里。
            </span>
          )}
        </StatusHint>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <HeaderBar conversationId={conversationId}>
        <span className="text-[10px] uppercase tracking-wider text-muted-foreground">
          {sourceLabel}
        </span>
        <span className="text-[10px] text-muted-foreground font-mono truncate max-w-[180px]">
          {artifact.path}
        </span>
        <button
          type="button"
          aria-label="reload preview"
          onClick={reload}
          className="text-muted-foreground hover:text-foreground transition-colors"
        >
          <RefreshCcw className="h-3.5 w-3.5" />
        </button>
        <a
          href={artifact.url}
          target="_blank"
          rel="noreferrer"
          aria-label="open in new tab"
          className="text-muted-foreground hover:text-foreground transition-colors"
        >
          <ExternalLink className="h-3.5 w-3.5" />
        </a>
      </HeaderBar>

      {artifact.status === 'failed' ? (
        <FailedBlock errorMsg={artifact.errorMsg} url={artifact.url} />
      ) : (
        <iframe
          key={`${artifact.id}-${reloadKey}`}
          src={artifact.url}
          title={`preview ${artifact.id}`}
          // iframe sandbox:不让 build 出来的页面跑顶层脚本(同源但我们不信任用户
          // 自己跑出的代码);allow-same-origin 是相对 JuiceFS / Nginx 协议需要的。
          sandbox="allow-scripts allow-same-origin allow-forms"
          className="flex-1 w-full border-0 bg-background"
        />
      )}
    </div>
  );
}

function HeaderBar({
  conversationId,
  children,
}: {
  conversationId: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-1.5 px-3 py-2 border-b">
      <Radio className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
      <span className="flex-1 text-xs font-mono truncate">Live Preview · {conversationId.slice(0, 8)}</span>
      {children}
    </div>
  );
}

function StatusHint({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-4 py-6 text-xs text-muted-foreground space-y-1 flex items-center gap-2">
      {children}
    </div>
  );
}

function FailedBlock({ errorMsg, url }: { errorMsg: string; url: string }) {
  return (
    <div className="flex-1 px-4 py-6 text-xs text-muted-foreground space-y-2">
      <div className="flex items-center gap-2 text-destructive">
        <AlertCircle className="h-3.5 w-3.5" />
        <span className="font-medium">构建失败</span>
      </div>
      {errorMsg ? (
        <pre className="whitespace-pre-wrap break-words text-[11px] leading-relaxed bg-muted/40 p-2 rounded">
          {errorMsg}
        </pre>
      ) : null}
      <a
        href={url}
        target="_blank"
        rel="noreferrer"
        className="inline-flex items-center gap-1 underline decoration-dotted"
      >
        打开 build_artifact 输出路径 <ExternalLink className="h-3 w-3" />
      </a>
    </div>
  );
}