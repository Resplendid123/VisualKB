import type {
  CreateDocumentInput,
  CreateDocumentResult,
  DocumentDetail,
  ListDocumentsInput,
  ListDocumentsResult,
  MoveDocumentInput,
} from '@/domain/entities/document';

// PDF 上传入参;后端按 file MIME 推 content_type,前端不再传。
export interface UploadDocumentInput {
  source: 'knowledge';
  title: string;
  lang?: string;
  parent_tree_id?: number | null;
  file: File;
}

// 文档端口:与后端 /api/v1/documents 对齐;目录能力(move/listPaths/path 过滤)拆到 treeRepository。
export interface DocumentRepository {
  list(input: ListDocumentsInput): Promise<ListDocumentsResult>;
  get(id: number): Promise<DocumentDetail>;
  create(input: CreateDocumentInput): Promise<CreateDocumentResult>;
  upload(input: UploadDocumentInput): Promise<CreateDocumentResult>;
  patch(
    id: number,
    ops: Array<{ type: string; args: Record<string, unknown> }>,
    title?: string
  ): Promise<{ document_id: number; version: number }>;
  archive(id: number): Promise<void>;
  move(id: number, input: MoveDocumentInput): Promise<void>;
  ingestAll(source: 'note' | 'knowledge'): Promise<number>;
}