// 列表/卡片复用,统一"刚刚 / X 分钟前 / X 小时前 / 昨天 / X 月 X 日"格式。
export function formatRelativeDate(input: string | Date): string {
  const d = typeof input === 'string' ? new Date(input) : input;
  if (Number.isNaN(d.getTime())) return '';

  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  const diffHour = Math.floor(diffMs / 3600000);
  const sameDay =
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate();
  const yesterday =
    new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1).getTime() ===
    new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();

  if (diffMin < 1) return '刚刚';
  if (diffMin < 60) return `${diffMin} 分钟前`;
  if (sameDay) return `${diffHour} 小时前`;
  if (yesterday) return '昨天';
  if (d.getFullYear() === now.getFullYear()) {
    return `${d.getMonth() + 1} 月 ${d.getDate()} 日`;
  }
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`;
}