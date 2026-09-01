export class DomainError extends Error {
  constructor(public readonly code: string, message: string) {
    super(message);
    this.name = 'DomainError';
  }
}

export const ErrInvalidName = () => new DomainError('INVALID_NAME', '昵称不能为空');

export const ErrInvalidEmail = () => new DomainError('INVALID_EMAIL', '邮箱格式错误');

export const ErrEmptyMessage = () => new DomainError('EMPTY_MESSAGE', '消息不能为空');

export const ErrUserNotFound = () => new DomainError('USER_NOT_FOUND', '用户不存在');

export const ErrUserCreateFailed = () => new DomainError('USER_CREATE_FAILED', '用户创建失败');

export const ErrPortraitTooLong = () => new DomainError('PORTRAIT_TOO_LONG', '画像超过 4000 字符');

// 按后端响应 code 翻成 DomainError;infra/http 层调用。
export function mapBackendCode(code: number, message: string): DomainError {
  switch (code) {
    case 1003:
      return new DomainError('INVALID_EMAIL', message || '邮箱格式错误');
    case 1004:
      return ErrUserCreateFailed();
    case 1010:
      return ErrPortraitTooLong();
    case 1106:
      return ErrUserNotFound();
    default:
      return new DomainError('UNKNOWN', message || '未知错误');
  }
}