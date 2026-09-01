// 前端读写 project 的端口。
import type { Project, ActiveProject } from '@/domain/entities/project';

export interface ProjectRepository {
  list(): Promise<Project[]>;
  create(title: string): Promise<Project>;
  rename(id: string, title: string): Promise<Project>;
  getActive(conversationId: string): Promise<ActiveProject | null>;
  setActive(conversationId: string, projectId: string): Promise<ActiveProject>;
  archive(projectId: string): Promise<void>;
}