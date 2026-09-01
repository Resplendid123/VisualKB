'use client';

import { useRef } from 'react';
import {
  FileText,
  Layers,
  Radio,
  Terminal as TerminalIcon,
  X,
  type LucideIcon,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import { usePanelStore, type PanelMode } from '@/presentation/stores/panelStore';
import { LivePreviewView } from './live-preview-view';
import { TerminalView } from './terminal-view';
import { DocumentPreviewView } from './document-preview-view';

interface ChatSidePanelProps {
  onClose: () => void;
  // 当前 conversation 的 id;live 视图用它取 active project。
  conversationId: string;
  // Sandbox-stamped CDN URL; "" until controller stamps it.
  activeProjectPreviewUrl?: string;
}

const MODE_TITLES: Record<PanelMode, string> = {
  default: '预览面板',
  file: '文件预览',
  terminal: '虚拟终端',
  live: 'Live Preview',
};

const MODES: { key: PanelMode; icon: LucideIcon; label: string }[] = [
  { key: 'default', icon: Layers, label: '默认页' },
  { key: 'file', icon: FileText, label: '文件预览' },
  { key: 'terminal', icon: TerminalIcon, label: '虚拟终端' },
  { key: 'live', icon: Radio, label: 'Live Preview' },
];

export function ChatSidePanel({ onClose, conversationId, activeProjectPreviewUrl }: ChatSidePanelProps) {
  const width = usePanelStore((s) => s.width);
  const mode = usePanelStore((s) => s.mode);
  const setMode = usePanelStore((s) => s.setMode);

  return (
    <aside
      style={{ width }}
      className="shrink-0 border-l flex flex-col h-full bg-background relative"
    >
      <ResizeHandle width={width} />

      <div className="flex items-center justify-between px-4 py-3 border-b">
        <h3 className="text-sm font-semibold">{MODE_TITLES[mode]}</h3>
        <div className="flex items-center gap-1">
          <ModeSwitcher current={mode} onChange={setMode} />
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={onClose}
            aria-label="关闭面板"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {mode === 'default' && <DefaultView onPick={setMode} />}
        {mode === 'terminal' && <TerminalView conversationId={conversationId} />}
        {mode === 'file' && <DocumentPreviewView />}
        {mode === 'live' && <LivePreviewView conversationId={conversationId} activeProjectPreviewUrl={activeProjectPreviewUrl} />}
      </div>
    </aside>
  );
}

// 拖拽手柄 — 抓住往左/右拖动改变面板宽度。
function ResizeHandle({ width }: { width: number }) {
  const setWidth = usePanelStore((s) => s.setWidth);
  const startXRef = useRef(0);
  const startWidthRef = useRef(0);

  function onMouseDown(e: React.MouseEvent) {
    e.preventDefault();
    startXRef.current = e.clientX;
    startWidthRef.current = width;

    const onMouseMove = (ev: MouseEvent) => {
      // 往左拖 → clientX 变小 → 宽度变大(setWidth 内部 clamp)。
      const delta = startXRef.current - ev.clientX;
      setWidth(startWidthRef.current + delta);
    };

    const onMouseUp = () => {
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
    };

    document.addEventListener('mousemove', onMouseMove);
    document.addEventListener('mouseup', onMouseUp);
  }

  return (
    <div
      onMouseDown={onMouseDown}
      // 键盘可达:聚焦后 ←/→ 调宽度(每次 16px),Home/End 到 min/max。Shift 加步。
      tabIndex={0}
      role="separator"
      aria-orientation="vertical"
      aria-valuenow={width}
      aria-valuemin={360}
      aria-valuemax={720}
      aria-label="调整侧栏宽度"
      onKeyDown={(e) => {
        const step = e.shiftKey ? 64 : 16;
        if (e.key === 'ArrowLeft') {
          e.preventDefault();
          setWidth(width + step);
        } else if (e.key === 'ArrowRight') {
          e.preventDefault();
          setWidth(width - step);
        } else if (e.key === 'Home') {
          e.preventDefault();
          setWidth(360);
        } else if (e.key === 'End') {
          e.preventDefault();
          setWidth(720);
        }
      }}
      className="absolute left-0 top-0 bottom-0 w-1 cursor-col-resize z-10 group focus-visible:w-1.5 focus-visible:bg-primary/30"
    >
      <div className="h-full w-px bg-border group-hover:bg-primary/50 group-active:bg-primary transition-colors mx-auto" />
    </div>
  );
}

// 4 个图标按钮的 segmented control。
function ModeSwitcher({
  current,
  onChange,
}: {
  current: PanelMode;
  onChange: (m: PanelMode) => void;
}) {
  return (
    <div className="flex items-center rounded-md border bg-muted/30 p-0.5">
      {MODES.map(({ key, icon: Icon, label }) => {
        const active = key === current;
        return (
          <Tooltip key={key}>
            <TooltipTrigger
              render={
                <button
                  type="button"
                  onClick={() => onChange(key)}
                  aria-label={label}
                  aria-pressed={active}
                  className={cn(
                    'h-6 w-6 rounded inline-flex items-center justify-center transition-colors',
                    active
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  <Icon className="h-3.5 w-3.5" />
                </button>
              }
            />
            <TooltipContent>{label}</TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}

// 入口页:可点击卡片跳到对应单视图。
function DefaultView({ onPick }: { onPick: (m: PanelMode) => void }) {
  const items: { key: PanelMode; icon: LucideIcon; title: string; desc: string }[] = [
    {
      key: 'file',
      icon: FileText,
      title: '文件预览',
      desc: '查看项目里任意文件',
    },
    {
      key: 'terminal',
      icon: TerminalIcon,
      title: '虚拟终端',
      desc: 'AI 跑的 bash 命令与输出',
    },
    {
      key: 'live',
      icon: Radio,
      title: 'Live Preview',
      desc: 'agent 起的 dev server / watcher iframe 预览',
    },
  ];

  return (
    <div className="p-4 space-y-3">
      <div className="text-xs text-muted-foreground">选择一个视图:</div>
      <div className="space-y-2">
        {items.map((it) => {
          const Icon = it.icon;
          return (
            <button
              key={it.key}
              type="button"
              onClick={() => onPick(it.key)}
              className="w-full text-left border rounded-lg p-3 hover:bg-muted/50 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
            >
              <div className="flex items-center gap-2 mb-1">
                <Icon className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">{it.title}</span>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">{it.desc}</p>
            </button>
          );
        })}
      </div>
    </div>
  );
}