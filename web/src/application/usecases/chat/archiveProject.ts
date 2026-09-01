import type { ProjectRepository } from '@/domain/repositories/projectRepository';

// 软删 project;调方负责清掉引用(conversation.active_project_id 可能仍指向已归档 project)。
export class ArchiveProjectUseCase {
  constructor(private readonly projectRepo: ProjectRepository) {}

  async execute(projectId: string): Promise<void> {
    return this.projectRepo.archive(projectId);
  }
}