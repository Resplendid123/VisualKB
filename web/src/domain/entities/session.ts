// 已登录会话;access_token 由后端写入 HttpOnly cookie,JS 不可读,不在这里。
export interface Session {
  user: {
    id: number;
    name: string;
    email: string;
  };
}