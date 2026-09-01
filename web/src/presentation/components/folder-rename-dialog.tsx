'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';

export interface FolderRenameDialogProps {
  // 当前目录 tree node id。null 时对话框隐藏。
  folderId: number | null;
  onClose: () => void;
  // 用户提交的新单段名(不含路径);后端只改 knowledge_tree.name 一行,后代不动,S3 不动。
  onSubmit: (newName: string) => void | Promise<void>;
}

const NAME_MAX = 64;
const NAME_RE = /^[^\x00-\x1F\x7F]+$/;

// 目录重命名对话框:后端仅改 knowledge_tree.name 一行,后代 docs/子文件夹不受影响。
export function FolderRenameDialog({
  folderId,
  onClose,
  onSubmit,
}: FolderRenameDialogProps) {
  return (
    <Dialog open={folderId !== null} onOpenChange={(o) => !o && onClose()}>
      {folderId !== null && (
        <DialogContent>
          {/* key={folderId} 强制 remount — 每次切目录表单状态自动归零,避开 set-state-in-effect 警告。 */}
          <FolderRenameForm
            key={folderId}
            onClose={onClose}
            onSubmit={onSubmit}
          />
        </DialogContent>
      )}
    </Dialog>
  );
}

function FolderRenameForm({
  onClose,
  onSubmit,
}: {
  onClose: () => void;
  onSubmit: (newName: string) => void | Promise<void>;
}) {
  const [value, setValue] = useState('');
  const [err, setErr] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    const trimmed = value.trim();
    if (trimmed === '') {
      setErr('目录名不能为空');
      return;
    }
    if (!NAME_RE.test(trimmed)) {
      setErr('目录名不能含控制字符或换行');
      return;
    }
    if ([...trimmed].length > NAME_MAX) {
      setErr(`目录名过长(最多 ${NAME_MAX} 个字符)`);
      return;
    }
    setSaving(true);
    setErr(null);
    try {
      await onSubmit(trimmed);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setSaving(false);
      return;
    }
    setSaving(false);
    onClose();
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>重命名目录</DialogTitle>
        <DialogDescription>
          改目录名(单段);后代文档 / 子目录不受影响,S3 对象不动。
        </DialogDescription>
      </DialogHeader>
      <div className="space-y-2">
        <Input
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="例:transformers"
          className="h-9"
          autoFocus
        />
        {err && (
          <div className="text-xs text-destructive px-2 py-1 rounded bg-destructive/10">
            {err}
          </div>
        )}
        <div className="text-[10px] text-muted-foreground/80 leading-relaxed">
          1-64 字符,允许中文等;不能含控制字符或换行。
        </div>
      </div>
      <DialogFooter>
        <Button variant="ghost" size="sm" onClick={onClose} disabled={saving}>
          取消
        </Button>
        <Button size="sm" onClick={() => void submit()} disabled={saving}>
          {saving ? '重命名中…' : '重命名'}
        </Button>
      </DialogFooter>
    </>
  );
}