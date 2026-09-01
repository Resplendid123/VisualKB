// 前端读写 build artifact 的端口 — 当前只有查询,build 走 LLM 工具发起。
import type { Artifact } from '@/domain/entities/artifact';

export interface ArtifactRepository {
  // 拿 active project 的最新 artifact。404 区分两个空态:
  //   - conversation 还没绑 active project
  //   - 绑了但还没 build 过
  // 调用方决定怎么显示。
  latest(conversationId: string): Promise<Artifact | null>;
  // 全部 build 历史,按 built_at desc。
  list(conversationId: string): Promise<Artifact[]>;
}