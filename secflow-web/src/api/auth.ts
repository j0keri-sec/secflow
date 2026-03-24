import http from '@/utils/http'
import type {
  LoginRequest,
  LoginResponse,
  UserProfile,
  InviteCode,
  PageData,
} from '@/types'

export const authApi = {
  login: (data: LoginRequest) => http.post<LoginResponse>('/auth/login', data),
  register: (data: { username: string; password: string; email: string; invite_code: string }) =>
    http.post<LoginResponse>('/auth/register', data),
  me: () => http.get<UserProfile>('/auth/me'),
  changePassword: (data: { old_password: string; new_password: string }) =>
    http.put<null>('/auth/password', data),
  generateInviteCode: () => http.post<InviteCode>('/auth/invite', {}),
  listInviteCodes: () => http.get<InviteCode[]>('/auth/invite'),

  // Admin-only
  adminListUsers: () => http.get<UserProfile[]>('/users'),
  adminUpdateUser: (id: string, data: Partial<UserProfile & { active: boolean }>) =>
    http.patch<UserProfile>(`/users/${id}`, data),
  adminDeleteUser: (id: string) => http.del<null>(`/users/${id}`),
}

export const userApi = {
  list: (page = 1, pageSize = 20) =>
    http.get<PageData<UserProfile>>(`/users?page=${page}&page_size=${pageSize}`),
  updateRole: (id: string, role: string) => http.patch<null>(`/users/${id}/role`, { role }),
  delete: (id: string) => http.del<null>(`/users/${id}`),
}
