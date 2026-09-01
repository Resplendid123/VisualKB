export interface User {
  id: number;
  name: string;
  email: string;
}

// 用户画像:immutable 由用户自己写,mutable 由 agent write_memory 工具写。
export interface UserPortrait {
  immutable: string;
  mutable: string;
}