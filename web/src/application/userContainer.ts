import { HttpUserRepository } from '@/infra/http/UserRepository';
import { CreateUserUseCase } from '@/application/usecases/user/createUser';
import {
  GetMyPortraitUseCase,
  UpdateMyImmutableUseCase,
  UpdateMyMutableUseCase,
} from '@/application/usecases/user/portrait';
import { authedFetch } from './authContainer';

// 客户端用例容器 — 注入 authedFetch,401 自动 refresh + 重试。
const userRepo = new HttpUserRepository(authedFetch);

export const userUseCases = {
  createUser: new CreateUserUseCase(userRepo),
  getMyPortrait: new GetMyPortraitUseCase(userRepo),
  updateMyImmutable: new UpdateMyImmutableUseCase(userRepo),
  updateMyMutable: new UpdateMyMutableUseCase(userRepo),
};
