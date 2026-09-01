import type { ArtifactRepository } from '@/domain/repositories/artifactRepository';
import type { Artifact } from '@/domain/entities/artifact';

export class ListArtifactsUseCase {
  constructor(private readonly artifactRepo: ArtifactRepository) {}

  async execute(conversationId: string): Promise<Artifact[]> {
    return this.artifactRepo.list(conversationId);
  }
}