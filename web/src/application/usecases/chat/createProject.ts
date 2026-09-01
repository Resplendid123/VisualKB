import type { ProjectRepository } from '@/domain/repositories/projectRepository';
import type { Project } from '@/domain/entities/project';

// 用户主动建项目(sidebar + 按钮);后端默认 title="未命名"。
export class CreateProjectUseCase {
  constructor(private readonly projectRepo: ProjectRepository) {}

  async execute(title: string): Promise<Project> {
    return this.projectRepo.create(title);
  }
}