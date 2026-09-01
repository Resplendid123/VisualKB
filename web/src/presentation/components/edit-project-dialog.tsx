'use client';

import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import type { Project } from '@/domain/entities/project';

// 编辑项目名 dialog;父级控制 project state。
export function EditProjectDialog({
  project,
  onClose,
  onSave,
}: {
  project: Project | null;
  onClose: () => void;
  onSave: (id: string, title: string) => Promise<unknown> | void;
}) {
  const [draft, setDraft] = useState(() => project?.title ?? '');
  const [prevId, setPrevId] = useState(project?.id);
  const [saving, setSaving] = useState(false);

  // 切换编辑对象时同步草稿(避免 useEffect 同步 setState 的级联渲染)。
  if (project?.id !== prevId) {
    setPrevId(project?.id);
    setDraft(project?.title ?? '');
  }

  if (!project) return null;

  const trimmed = draft.trim();
  const unchanged = trimmed === project.title;
  const canSave = trimmed.length > 0 && !unchanged && !saving;

  async function commit() {
    if (!canSave || !project) return;
    setSaving(true);
    try {
      await onSave(project.id, trimmed);
      onClose();
    } catch {
      setSaving(false);
    }
  }

  return (
    <Dialog
      open={!!project}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>编辑项目</DialogTitle>
          <DialogDescription>修改项目显示名称;slug 不变,不影响 sandbox 路径。</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void commit();
          }}
        >
          <Input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="项目名"
            autoFocus
            onKeyDown={(e) => {
              if (e.key === 'Escape') {
                e.preventDefault();
                onClose();
              }
            }}
          />
          <DialogFooter className="mt-4">
            <DialogClose
              render={
                <Button type="button" variant="ghost" disabled={saving}>
                  取消
                </Button>
              }
            />
            <Button type="submit" disabled={!canSave}>
              {saving ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}