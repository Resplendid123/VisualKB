'use client';

import { MoreHorizontal } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu';

interface ChatHeaderProps {
  title: string;
  // 是否展示操作菜单:新对话(空)不展示,无内容可操作。
  hasActions?: boolean;
  // 归档当前对话;父级 ChatLayout 拥有会话状态。
  onDeleteConversation: () => void;
}

export function ChatHeader({
  title,
  hasActions = false,
  onDeleteConversation,
}: ChatHeaderProps) {
  return (
    <div className="flex items-center gap-1 px-6 py-3">
      <h2 className="text-base font-semibold truncate min-w-0">{title}</h2>

      {hasActions && (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 text-muted-foreground shrink-0"
                aria-label="对话操作"
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            }
          />
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              variant="destructive"
              onClick={() => {
                if (confirm('确定归档当前对话？')) {
                  onDeleteConversation();
                }
              }}
            >
              归档对话
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}