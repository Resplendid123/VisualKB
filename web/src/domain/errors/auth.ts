// 认证模块领域错误(继承 DomainError 形态,新增 auth 专用 code)。
import { DomainError } from '@/domain/errors';

export { DomainError };

export const ErrInvalidName = () => new DomainError('INVALID_NAME', '昵称不能为空');

export const ErrInvalidEmail = () => new DomainError('INVALID_EMAIL', '邮箱格式错误');

export const ErrInvalidPassword = () =>
  new DomainError('INVALID_PASSWORD', '密码至少 6 位');

export const ErrInvalidCredentials = () =>
  new DomainError('INVALID_CREDENTIALS', '邮箱或密码错误');

export const ErrAuthRegisterFailed = () =>
  new DomainError('AUTH_REGISTER_FAILED', '注册失败,请稍后再试');

export const ErrAuthLoginFailed = () =>
  new DomainError('AUTH_LOGIN_FAILED', '登录失败,请稍后再试');

// auth 模块后端错误(code 1003/1004/1006/1007/1008)→ DomainError。
export function mapAuthBackendError(code: number, message: string): DomainError {
  switch (code) {
    case 1003:
      return new DomainError('INVALID_EMAIL', message || '邮箱格式错误');
    case 1004:
      return ErrAuthRegisterFailed();
    case 1006:
      return ErrInvalidCredentials();
    default:
      return new DomainError('UNKNOWN', message || '未知错误');
  }
}
