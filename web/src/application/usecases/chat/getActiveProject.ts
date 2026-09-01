import type { ProjectRepository } from '@/domain/repositories/projectRepository';
import type { ActiveProject } from '@/domain/entities/project';

export class GetActiveProjectUseCase {
  constructor(private readonly projectRepo: ProjectRepository) {}

  async execute(conversationId: string): Promise<ActiveProject | null> {
    return this.projectRepo.getActive(conversationId);
  }
}