// build_artifact 工具产物:每次 Coding Agent 调一次 build_artifact,后端
// 跑一次 ephemeral build pod,把产物落到 JuiceFS /static/{uid}/{pid}/{ts}/,
// 返回 public URL 给前端 iframe 直接嵌。
//
// ID 是后端生成的 UUID,front-end 不解析它的格式,只用来做缓存 key。
export type ArtifactStatus = 'building' | 'succeeded' | 'failed';

export interface Artifact {
  id: string;
  projectId: string;
  // next / vite / static / hugo / jekyll / mkdocs / 空字符串。
  framework: string;
  // /static/{uid}/{pid}/{ts}/index.html — JuiceFS 上的相对路径,用来 log / debug。
  path: string;
  // http://<gateway>/static/{uid}/{pid}/{ts}/index.html — iframe 直接 src。
  url: string;
  status: ArtifactStatus;
  errorMsg: string;
  builtAt: string; // RFC3339
}