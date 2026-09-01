import { parseEnvelope } from './AuthClient';
import type { AuthedFetch } from './AuthClient';
import type { ArtifactRepository } from '@/domain/repositories/artifactRepository';
import type { Artifact } from '@/domain/entities/artifact';

interface ArtifactPayload {
  id: string;
  project_id: string;
  framework: string;
  path: string;
  url: string;
  status: string;
  error_msg?: string;
  built_at: string;
}

interface ArtifactListPayload {
  artifacts: ArtifactPayload[];
}

function toArtifact(p: ArtifactPayload): Artifact {
  return {
    id: p.id,
    projectId: p.project_id,
    framework: p.framework,
    path: p.path,
    url: p.url,
    status: (p.status as Artifact['status']) ?? 'building',
    errorMsg: p.error_msg ?? '',
    builtAt: p.built_at,
  };
}

export class HttpArtifactRepository implements ArtifactRepository {
  constructor(public readonly authedFetch: AuthedFetch) {}

  async latest(conversationId: string): Promise<Artifact | null> {
    if (!conversationId) return null;
    let res: Response;
    try {
      res = await this.authedFetch(
        `/api/v1/conversations/${encodeURIComponent(conversationId)}/artifacts/latest`
      );
    } catch {
      // 404 / 5xx 都落到这里,统一当 "no artifact yet"。
      return null;
    }
    if (res.status === 404) return null;
    const data = await parseEnvelope<ArtifactPayload>(res, 'get latest artifact');
    return toArtifact(data);
  }

  async list(conversationId: string): Promise<Artifact[]> {
    if (!conversationId) return [];
    let res: Response;
    try {
      res = await this.authedFetch(
        `/api/v1/conversations/${encodeURIComponent(conversationId)}/artifacts`
      );
    } catch {
      return [];
    }
    if (res.status === 404) return [];
    const data = await parseEnvelope<ArtifactListPayload>(res, 'list artifacts');
    return data.artifacts.map(toArtifact);
  }
}