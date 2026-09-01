'use client';

import { Check, ChevronDown, Folder, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import type { Project, ActiveProject } from '@/domain/entities/project';

// 静态显示当前项目;有 projects + onPick 就成为 dropdown;X 只在 onClear 存在时显示。
export function ProjectPicker({
  project,
  projects,
  onPick,
  onClear,
}: {
  project: Project | ActiveProject;
  projects: Project[];
  onPick?: (id: string) => void;
  onClear?: () => void;
}) {
  const canPick = projects.length > 0 && !!onPick;
  const currentId = project.id;

  const trigger = (
    <button
      type="button"
      className={cn(
        'group/picker inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors min-w-0 max-w-[20rem] h-7 pl-2 pr-1.5 rounded-md bg-muted/60 hover:bg-muted',
        canPick && 'cursor-pointer'
      )}
      title={project.title}
    >
      <Folder className="h-3.5 w-3.5 shrink-0" />
      <span className="truncate">{project.title}</span>
      {canPick && (
        <ChevronDown className="h-3 w-3 shrink-0 opacity-60 group-hover/picker:opacity-100" />
      )}
    </button>
  );

  return (
    <div className="flex items-center gap-1">
      {canPick ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={trigger} />
          <DropdownMenuContent align="start">
            {projects.map((p) => (
              <DropdownMenuItem key={p.id} onClick={() => onPick!(p.id)}>
                <Folder className="h-3.5 w-3.5" />
                <span className="flex-1 truncate">{p.title}</span>
                {p.id === currentId && <Check className="h-3.5 w-3.5" />}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      ) : (
        trigger
      )}
      {onClear && (
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label="取消项目"
          onClick={onClear}
        >
          <X className="h-3.5 w-3.5" />
        </Button>
      )}
    </div>
  );
}