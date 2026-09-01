import {
  expectNoContent,
  parseEnvelope,
} from './AuthClient';
import type { AuthedFetch } from './AuthClient';
import type {
  DocumentRepository,
  UploadDocumentInput,
} from '@/domain/repositories/documentRepository';
import type {
  CreateDocumentInput,
  CreateDocumentResult,
  Document,
  DocumentDetail,
  ListDocumentsInput,
  ListDocumentsResult,
  MoveDocumentInput,
} from '@/domain/entities/document';

interface DocumentPayload {
  id: number;
  title: string;
  source: 'note' | 'knowledge';
  lang: string;
  content_type: string;
  chunk_status: number;
  current_version_id: number | null;
  archived_at: string | null;
  created_at: string;
  updated_at: string | null;
}

interface DocumentDetailPayload extends DocumentPayload {
  content: string;
}

interface ListResponse {
  items: DocumentPayload[];
  total: number;
}

interface CreateResponse {
  document_id: number;
}

interface VersionResponse {
  document_id: number;
  version: number;
}

export class HttpDocumentRepository implements DocumentRepository {
  constructor(public readonly authedFetch: AuthedFetch) {}

  async list(input: ListDocumentsInput): Promise<ListDocumentsResult> {
    const qs = buildListQuery(input);
    const res = await this.authedFetch(`/api/v1/documents${qs}`);
    const data = await parseEnvelope<ListResponse>(res, 'list documents');
    return { items: data.items.map(toDocument), total: data.total };
  }

  async get(id: number): Promise<DocumentDetail> {
    const res = await this.authedFetch(`/api/v1/documents/${id}`);
    const data = await parseEnvelope<DocumentDetailPayload>(res, 'get document');
    return toDocumentDetail(data);
  }

  async create(input: CreateDocumentInput): Promise<CreateDocumentResult> {
    const res = await this.authedFetch('/api/v1/documents', {
      method: 'POST',
      body: JSON.stringify(input),
    });
    const data = await parseEnvelope<CreateResponse>(res, 'create document');
    return { document_id: data.document_id };
  }

  async upload(input: UploadDocumentInput): Promise<CreateDocumentResult> {
    const fd = new FormData();
    fd.append('source', input.source);
    fd.append('title', input.title);
    if (input.lang) fd.append('lang', input.lang);
    if (input.parent_tree_id != null)
      fd.append('parent_tree_id', String(input.parent_tree_id));
    fd.append('file', input.file, input.file.name);
    // authedFetch 看到 FormData 会跳过 JSON 默认值,浏览器自己补 multipart/form-data。
    const res = await this.authedFetch('/api/v1/documents/upload', {
      method: 'POST',
      body: fd,
    });
    const data = await parseEnvelope<CreateResponse>(res, 'upload document');
    return { document_id: data.document_id };
  }

  async patch(
    id: number,
    ops: Array<{ type: string; args: Record<string, unknown> }>,
    title?: string
  ): Promise<{ document_id: number; version: number }> {
    const body: { ops: typeof ops; title?: string } = { ops };
    if (title !== undefined && title !== '') body.title = title;
    const res = await this.authedFetch(`/api/v1/documents/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    });
    const data = await parseEnvelope<VersionResponse>(res, 'patch document');
    return { document_id: data.document_id, version: data.version };
  }

  async archive(id: number): Promise<void> {
    const res = await this.authedFetch(`/api/v1/documents/${id}/archive`, {
      method: 'POST',
    });
    await expectNoContent(res, 'archive document');
  }

  async move(id: number, input: MoveDocumentInput): Promise<void> {
    const res = await this.authedFetch(`/api/v1/documents/${id}/move`, {
      method: 'POST',
      body: JSON.stringify(input),
    });
    await expectNoContent(res, 'move document');
  }

  async ingestAll(source: 'note' | 'knowledge'): Promise<number> {
    const res = await this.authedFetch(
      `/api/v1/documents/ingest-all?source=${encodeURIComponent(source)}`,
      { method: 'POST' }
    );
    const data = await parseEnvelope<{ enqueued: number }>(res, 'ingest all documents');
    return data.enqueued;
  }
}

function toDocument(p: DocumentPayload): Document {
  return {
    id: p.id,
    title: p.title,
    source: p.source,
    lang: p.lang,
    content_type: p.content_type,
    chunk_status: p.chunk_status as Document['chunk_status'],
    current_version_id: p.current_version_id,
    archived_at: p.archived_at,
    created_at: p.created_at,
    updated_at: p.updated_at,
  };
}

function toDocumentDetail(p: DocumentDetailPayload): DocumentDetail {
  return { ...toDocument(p), content: p.content };
}

function buildListQuery(input: ListDocumentsInput): string {
  const sp = new URLSearchParams();
  if (input.source) sp.set('source', input.source);
  if (input.chunk_status !== undefined) sp.set('chunk_status', String(input.chunk_status));
  if (input.include_archived) sp.set('include_archived', 'true');
  if (input.limit !== undefined) sp.set('limit', String(input.limit));
  if (input.offset !== undefined) sp.set('offset', String(input.offset));
  const qs = sp.toString();
  return qs ? `?${qs}` : '';
}