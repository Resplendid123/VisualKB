// 全局 replace 去掉 LLM 输出里的 <think> 标签 — 兜底 splitter 漏掉的中段残留。
export function stripThinkTag(s: string): string {
  return s.replace(/<\/?think>/g, '');
}