// 与后端 domain/document 对齐;前端只用到展示层需要的字段。
export type DocumentSource = 'note' | 'knowledge';
export type DocumentContentType = 'markdown' | string;
// 0=dirty, 1=chunked, 2=error
export type DocumentChunkStatus = 0 | 1 | 2;

export interface Document {
  id: number;
  title: string;
  source: DocumentSource;
  lang: string;
  content_type: DocumentContentType;
  chunk_status: DocumentChunkStatus;
  current_version_id: number | null;
  archived_at: string | null;
  created_at: string;
  updated_at: string | null;
}

// 详情 — 列表返回 Document,详情额外带 raw content。
export interface DocumentDetail extends Document {
  content: string;
}

// content 必填,lang / content_type / parent_tree_id 可空;parent_tree_id 仅 knowledge 生效。
export interface CreateDocumentInput {
  source: DocumentSource;
  title: string;
  lang?: string;
  content_type?: DocumentContentType;
  parent_tree_id?: number | null;
  content: string;
}

export interface CreateDocumentResult {
  document_id: number;
}

export interface ListDocumentsInput {
  source?: DocumentSource;
  chunk_status?: DocumentChunkStatus;
  include_archived?: boolean;
  limit?: number;
  offset?: number;
}

export interface ListDocumentsResult {
  items: Document[];
  total: number;
}

// 文档移动入参;parent_tree_id 可空(根级)。
export interface MoveDocumentInput {
  parent_tree_id: number | null;
}