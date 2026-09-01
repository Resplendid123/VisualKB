export interface Conversation {
  id: string;
  title: string;
  // 当前对话绑定的 active project id;null 表示未绑定。后端 ActiveProjectID 字段透传。
  activeProjectId?: string;
}