import type { ProjectRepository } from '@/domain/repositories/projectRepository';
import type { Project } from '@/domain/entities/project';

// sidebar 拉所有未归档 project(按 updated_at 倒序)。
export class ListProjectsUseCase {
  constructor(private readonly projectRepo: ProjectRepository) {}

  async execute(): Promise<Project[]> {
    return this.projectRepo.list();
  }
}