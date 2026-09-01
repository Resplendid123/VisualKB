import { HttpChatRepository } from '@/infra/http/ChatRepository';
import { HttpConversationRepository } from '@/infra/http/ConversationRepository';
import { HttpDocumentRepository } from '@/infra/http/DocumentRepository';
import { HttpTreeRepository } from '@/infra/http/TreeRepository';
import { HttpProjectRepository } from '@/infra/http/ProjectRepository';
import { HttpArtifactRepository } from '@/infra/http/ArtifactRepository';
import { SendMessageUseCase } from '@/application/usecases/chat/sendMessage';
import { ReplayEventsUseCase } from '@/application/usecases/chat/replayEvents';
import { ListConversationsUseCase } from '@/application/usecases/chat/listConversations';
import { CreateConversationUseCase } from '@/application/usecases/chat/createConversation';
import { ArchiveConversationUseCase } from '@/application/usecases/chat/archiveConversation';
import { GetMessagesUseCase } from '@/application/usecases/chat/getMessages';
import { GetActiveProjectUseCase } from '@/application/usecases/chat/getActiveProject';
import { SwitchActiveProjectUseCase } from '@/application/usecases/chat/switchActiveProject';
import { ArchiveProjectUseCase } from '@/application/usecases/chat/archiveProject';
import { ListProjectsUseCase } from '@/application/usecases/chat/listProjects';
import { CreateProjectUseCase } from '@/application/usecases/chat/createProject';
import { RenameProjectUseCase } from '@/application/usecases/chat/renameProject';
import { GetLatestArtifactUseCase } from '@/application/usecases/chat/getLatestArtifact';
import { ListArtifactsUseCase } from '@/application/usecases/chat/listArtifacts';
import { ListDocumentsUseCase } from '@/application/usecases/documents/listDocuments';
import { GetDocumentUseCase } from '@/application/usecases/documents/getDocument';
import { CreateDocumentUseCase } from '@/application/usecases/documents/createDocument';
import { UploadDocumentUseCase } from '@/application/usecases/documents/uploadDocument';
import { UpdateDocumentUseCase } from '@/application/usecases/documents/updateDocument';
import { ArchiveDocumentUseCase } from '@/application/usecases/documents/archiveDocument';
import { MoveDocumentUseCase } from '@/application/usecases/documents/moveDocument';
import { IngestAllDocumentsUseCase } from '@/application/usecases/documents/ingestAllDocuments';
import { ListTreeUseCase } from '@/application/usecases/tree/listTree';
import { CreateFolderUseCase } from '@/application/usecases/tree/createFolder';
import { RenameNodeUseCase } from '@/application/usecases/tree/renameNode';
import { MoveNodeUseCase } from '@/application/usecases/tree/moveNode';
import { DeleteFolderUseCase } from '@/application/usecases/tree/deleteFolder';

import { authedFetch } from './authContainer';

// 客户端用例容器 — 装配 repo + use case,供 presentation 消费。
const convoRepo = new HttpConversationRepository(authedFetch);
// EventSource 不支持自定义 header,HttpChatRepository 走同源 cookie,不接 authedFetch。
const chatRepo = new HttpChatRepository();
const projectRepo = new HttpProjectRepository(authedFetch);
const artifactRepo = new HttpArtifactRepository(authedFetch);
const documentRepo = new HttpDocumentRepository(authedFetch);
const treeRepo = new HttpTreeRepository(authedFetch);

export const chatUseCases = {
  sendMessage: new SendMessageUseCase(chatRepo),
  replayEvents: new ReplayEventsUseCase(chatRepo),
  listConversations: new ListConversationsUseCase(convoRepo),
  createConversation: new CreateConversationUseCase(convoRepo),
  archiveConversation: new ArchiveConversationUseCase(convoRepo),
  getMessages: new GetMessagesUseCase(convoRepo),
  getActiveProject: new GetActiveProjectUseCase(projectRepo),
  switchActiveProject: new SwitchActiveProjectUseCase(projectRepo),
  archiveProject: new ArchiveProjectUseCase(projectRepo),
  listProjects: new ListProjectsUseCase(projectRepo),
  createProject: new CreateProjectUseCase(projectRepo),
  renameProject: new RenameProjectUseCase(projectRepo),
  getLatestArtifact: new GetLatestArtifactUseCase(artifactRepo),
  listArtifacts: new ListArtifactsUseCase(artifactRepo),
};

export const documentUseCases = {
  list: new ListDocumentsUseCase(documentRepo),
  get: new GetDocumentUseCase(documentRepo),
  create: new CreateDocumentUseCase(documentRepo),
  upload: new UploadDocumentUseCase(documentRepo),
  update: new UpdateDocumentUseCase(documentRepo),
  archive: new ArchiveDocumentUseCase(documentRepo),
  move: new MoveDocumentUseCase(documentRepo),
  ingestAll: new IngestAllDocumentsUseCase(documentRepo),
};

export const treeUseCases = {
  list: new ListTreeUseCase(treeRepo),
  createFolder: new CreateFolderUseCase(treeRepo),
  renameNode: new RenameNodeUseCase(treeRepo),
  moveNode: new MoveNodeUseCase(treeRepo),
  deleteFolder: new DeleteFolderUseCase(treeRepo),
};

export { documentRepo, treeRepo };