import type { ArtifactRepository } from '@/domain/repositories/artifactRepository';
import type { Artifact } from '@/domain/entities/artifact';

// 拿当前对话 active project 的最新 build artifact;Live Preview iframe 用。
export class GetLatestArtifactUseCase {
  constructor(private readonly artifactRepo: ArtifactRepository) {}

  async execute(conversationId: string): Promise<Artifact | null> {
    return this.artifactRepo.latest(conversationId);
  }
}