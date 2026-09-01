import type { ProjectRepository } from '@/domain/repositories/projectRepository';
import type { Project } from '@/domain/entities/project';

// 改 project 显示名(sidebar inline rename);仅更新 title,slug(name)不变,沙箱文件路径不受影响。
export class RenameProjectUseCase {
  constructor(private readonly projectRepo: ProjectRepository) {}

  async execute(id: string, title: string): Promise<Project> {
    return this.projectRepo.rename(id, title);
  }
}