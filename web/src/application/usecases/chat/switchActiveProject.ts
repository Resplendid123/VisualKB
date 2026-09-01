import type { ProjectRepository } from '@/domain/repositories/projectRepository';
import type { ActiveProject } from '@/domain/entities/project';

// 把对话 active project 切到 user 拥有的另一个 project;后端确保目标可用后回填 cwd。
export class SwitchActiveProjectUseCase {
  constructor(private readonly projectRepo: ProjectRepository) {}

  async execute(
    conversationId: string,
    projectId: string,
  ): Promise<ActiveProject> {
    return this.projectRepo.setActive(conversationId, projectId);
  }
}