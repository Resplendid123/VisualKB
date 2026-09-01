import { create } from 'zustand';

// build_artifact tool_result 落地后,dispatchStreamSideEffects 会调用
// notifyBuilt 进这里,LivePreviewView 订阅 store.seq 后立即重新拉一次
// 最新 artifact 然后切 iframe src。每个 conversation 各自一个"最新事件"位,
// 切对话不串。
export interface ArtifactBuiltEvent {
  // 该事件来自哪个 conversation — LivePreviewView 用 conversationId 过滤。
  conversationId: string;
  // artifact ID / url 都直接从 tool result 拿;保 ID 是为了 LivePreviewView
  // 拉最新时可以用 If-None-Match 之类的优化(目前没实现,但留口子)。
  artifactId: string;
  url: string;
}

interface ArtifactStreamState {
  latest: ArtifactBuiltEvent | null;
  seq: number;
}

interface ArtifactStreamActions {
  notifyBuilt: (e: ArtifactBuiltEvent) => void;
  reset: () => void;
}

export const useArtifactStreamStore = create<ArtifactStreamState & ArtifactStreamActions>(
  (set) => ({
    latest: null,
    seq: 0,

    notifyBuilt: (e) =>
      set((s) => ({
        latest: e,
        seq: s.seq + 1,
      })),

    reset: () => set({ latest: null, seq: 0 }),
  })
);

// 解析 build_artifact tool_result 的 content 字符串 — 后端 JSON.Marshal
// buildArtifactResult{ID, URL, Path, Framework, BuiltAt}。
//   容错:任何字段缺失 / 类型不对都返 null,不抛 — 让 UI 兜底空态即可。
export function parseArtifactToolResult(result: unknown): {
  id: string;
  url: string;
} | null {
  let r: Record<string, unknown> | null = null;
  if (typeof result === 'string') {
    try {
      const parsed = JSON.parse(result);
      if (parsed && typeof parsed === 'object') r = parsed as Record<string, unknown>;
    } catch {
      return null;
    }
  } else if (result && typeof result === 'object') {
    r = result as Record<string, unknown>;
  }
  if (!r) return null;
  const id = typeof r.id === 'string' ? r.id : '';
  const url = typeof r.url === 'string' ? r.url : '';
  if (!id || !url) return null;
  return { id, url };
}