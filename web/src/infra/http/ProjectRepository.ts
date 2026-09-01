import { expectNoContent, parseEnvelope } from './AuthClient';
import type { AuthedFetch } from './AuthClient';
import type { ProjectRepository } from '@/domain/repositories/projectRepository';
import type { Project, ActiveProject } from '@/domain/entities/project';

interface ProjectPayload {
  id: string;
  name: string;
  title: string;
  cwd: string;
  status: string;
  updated_at: string;
}

interface ProjectListResponse {
  projects: ProjectPayload[];
}

function toProject(p: ProjectPayload): Project {
  return {
    id: p.id,
    name: p.name,
    title: p.title,
    cwd: p.cwd,
    status: p.status,
    updatedAt: p.updated_at,
  };
}

interface ActiveProjectPayload {
  id: string;
  name: string;
  title: string;
  cwd: string;
  preview_url?: string;
  updated_at: string;
}

function toActiveProject(p: ActiveProjectPayload): ActiveProject {
  return {
    id: p.id,
    name: p.name,
    title: p.title,
    cwd: p.cwd,
    previewUrl: p.preview_url ?? '',
    updatedAt: p.updated_at,
  };
}

export class HttpProjectRepository implements ProjectRepository {
  constructor(public readonly authedFetch: AuthedFetch) {}

  async list(): Promise<Project[]> {
    const res = await this.authedFetch('/api/v1/projects');
    const data = await parseEnvelope<ProjectListResponse>(res, 'list projects');
    return data.projects.map(toProject);
  }

  async create(title: string): Promise<Project> {
    // 不带 conversation_id → 后端走 user 模式,slug 自动生成。
    const res = await this.authedFetch('/api/v1/projects', {
      method: 'POST',
      body: JSON.stringify({ name: title }),
    });
    const data = await parseEnvelope<ProjectPayload>(res, 'create project');
    return toProject(data);
  }

  async rename(id: string, title: string): Promise<Project> {
    const res = await this.authedFetch(
      `/api/v1/projects/${encodeURIComponent(id)}`,
      { method: 'PATCH', body: JSON.stringify({ title }) }
    );
    const data = await parseEnvelope<ProjectPayload>(res, 'rename project');
    return toProject(data);
  }

  async getActive(conversationId: string): Promise<ActiveProject | null> {
    // 草稿态/空 id 早返 null — 否则 URL 变 /conversations//active-project,Gin 报错。
    if (!conversationId) return null;
    // 后端约定:无 active 时 envelope.data = null,parseEnvelope allowNullData 接受。
    const res = await this.authedFetch(
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/active-project`
    );
    const data = await parseEnvelope<ActiveProjectPayload>(
      res,
      'get active project',
      undefined,
      true
    );
    return data ? toActiveProject(data) : null;
  }

  async setActive(conversationId: string, projectId: string): Promise<ActiveProject> {
    const res = await this.authedFetch(
      `/api/v1/conversations/${encodeURIComponent(conversationId)}/active-project`,
      {
        method: 'PUT',
        body: JSON.stringify({ project_id: projectId }),
      }
    );
    const data = await parseEnvelope<ActiveProjectPayload>(
      res,
      'set active project'
    );
    return toActiveProject(data);
  }

  async archive(projectId: string): Promise<void> {
    const res = await this.authedFetch(
      `/api/v1/projects/${encodeURIComponent(projectId)}/archive`,
      { method: 'POST' }
    );
    await expectNoContent(res, 'archive project');
  }
}