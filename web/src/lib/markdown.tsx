import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import { cn } from './utils';

// 共享 Markdown prose 类:note-editor / document-preview-view 共用,只覆写外观不一致处。
export const MARKDOWN_PROSE_CLASS =
  'prose prose-sm dark:prose-invert max-w-none min-w-0 ' +
  // inline code 显式 text-foreground,避免 light/dark 都被 prose 默认色拖累。
  '[&_code]:bg-zinc-200/80 dark:[&_code]:bg-zinc-800/80 [&_code]:text-foreground [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:rounded [&_code]:font-normal ' +
  '[&_code]:before:content-none [&_code]:after:content-none ' +
  // code block 底色再深一档,跟 inline code 拉开。
  '[&_pre]:bg-zinc-100 dark:[&_pre]:bg-zinc-900 [&_pre]:text-foreground [&_pre]:rounded-md [&_pre]:overflow-x-auto ' +
  '[&_pre]:max-w-full [&_pre]:min-w-0 ' +
  '[&_blockquote]:not-italic [&_blockquote]:border-muted-foreground/30 [&_blockquote]:text-muted-foreground ' +
  '[&_a]:text-blue-500 [&_a]:underline [&&_a]:underline-offset-2 ' +
  '[&_img]:rounded-md [&_img]:max-w-full';

// remark-math v6 只认 $...$ / $$...$$;先换 LaTeX 原生分隔符,否则 \[ 会被 remark-parse 当转义吃。
function normalizeMathDelimiters(src: string): string {
  return src
    .replace(/\\\[([\s\S]*?)\\]/g, (_, inner) => `$$${inner}$$`)
    .replace(/\\\(([\s\S]*?)\\\)/g, (_, inner) => `$${inner}$`);
}

// katex CSS 在 app/layout.tsx 引入。
const REMARK_PLUGINS = [remarkGfm, remarkMath];
const REHYPE_PLUGINS = [rehypeKatex];

export function MarkdownView({
  content,
  className,
}: {
  content: string;
  className?: string;
}) {
  const normalized = normalizeMathDelimiters(content);
  return (
    <div className={cn(MARKDOWN_PROSE_CLASS, className)}>
      <ReactMarkdown
        remarkPlugins={REMARK_PLUGINS}
        rehypePlugins={REHYPE_PLUGINS}
      >
        {normalized}
      </ReactMarkdown>
    </div>
  );
}