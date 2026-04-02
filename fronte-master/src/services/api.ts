import type {
  AuthRequest,
  AuthResponse,
  Session,
  CreateSessionRequest,
  ContentResponse,
  UpdateContentRequest,
  UpdateContentResponse,
  SessionEvent,
  AiReview,
  FileNode,
  CreateFileRequest,
  DeleteFileResponse,
  ApiResponse, UserProfile, LeaderboardEntry, RestoreVersionResponse, HistoryVersion, JoinByLinkResponse,
  InviteLinkResponse, DbParticipant,
} from './types';

export class ApiRequestError extends Error {
  public code: number;

  constructor(message: string, code: number) {
    super(message);
    this.name = 'ApiRequestError';
    this.code = code;
  }
}

function resolveApiBaseUrl(): string {
  // const envUrl = (import.meta.env.VITE_API_URL as string | undefined)?.trim();
  // if (envUrl) return envUrl.replace(/\/$/, '');
  // if (typeof window !== 'undefined') return window.location.origin;
    return 'http://localhost:8080';
}

function isApiError<T>(value: ApiResponse<T>): value is Extract<ApiResponse<T>, { success: false }> {
  return value.success === false;
}

export class ApiClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl;
  }

  public setToken(token: string) {
    this.token = token;
  }

  public clearToken() {
    this.token = null;
  }

  public getToken(): string | null {
    return this.token;
  }


  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const headers = new Headers(options.headers);
    if (!(options.body instanceof FormData)) {
      headers.set('Content-Type', 'application/json');
    }

    if (this.token) {
      headers.set('Authorization', `Bearer ${this.token}`);
    }

    let response: Response;
    try {
      response = await fetch(`${this.baseUrl}${endpoint}`, {
        ...options,
        headers,
      });
    } catch (error) {
      throw new ApiRequestError('Сервер недоступен. Проверь, что бэкенд запущен.', 0);
    }

    let json: ApiResponse<T> | null = null;
    try {
      json = (await response.json()) as ApiResponse<T>;
    } catch {
      if (!response.ok) {
        throw new ApiRequestError(`Ошибка HTTP ${response.status}`, response.status);
      }
      throw new ApiRequestError('Сервер вернул некорректный ответ.', response.status);
    }

    if (!response.ok) {
      if (json && isApiError(json)) {
        throw new ApiRequestError(json.error, json.code || response.status);
      }
      throw new ApiRequestError(`Ошибка HTTP ${response.status}`, response.status);
    }

    if (isApiError(json)) {
      throw new ApiRequestError(json.error, json.code);
    }

    return json.data;
  }

  async register(data: AuthRequest): Promise<AuthResponse> {
    const res = await this.request<AuthResponse>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    this.setToken(res.token);
    return res;
  }

  async login(data: AuthRequest): Promise<AuthResponse> {
    const res = await this.request<AuthResponse>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    this.setToken(res.token);
    return res;
  }

  async getMySessions(): Promise<Session[]> {
    return this.request<Session[]>('/api/user/sessions');
  }

  async createSession(data: CreateSessionRequest): Promise<Session> {
    return this.request<Session>('/api/sessions', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getSession(id: string): Promise<Session> {
    return this.request<Session>(`/api/sessions/${id}`);
  }

  async getSessionEvents(id: string): Promise<SessionEvent[]> {
    return this.request<SessionEvent[]>(`/api/sessions/${id}/events`);
  }

  async getAiReviews(id: string): Promise<AiReview[]> {
    return this.request<AiReview[]>(`/api/sessions/${id}/ai-reviews`);
  }

  async applyAiReview(sessionId: string, reviewId: string): Promise<{ status: 'applied' }> {
    return this.request<{ status: 'applied' }>(`/api/sessions/${sessionId}/ai-reviews/${reviewId}/apply`, {
      method: 'POST',
    });
  }


  async getSessionContent(id: string): Promise<ContentResponse> {
    return this.request<ContentResponse>(`/api/sessions/${id}/content`);
  }

  async updateSessionContent(id: string, data: UpdateContentRequest): Promise<UpdateContentResponse> {
    return this.request<UpdateContentResponse>(`/api/sessions/${id}/content`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async getFiles(sessionId: string): Promise<FileNode[]> {
    return this.request<FileNode[]>(`/api/sessions/${sessionId}/files`);
  }


  async createFile(sessionId: string, data: CreateFileRequest): Promise<FileNode> {
    return this.request<FileNode>(`/api/sessions/${sessionId}/files`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getFileContent(sessionId: string, path: string): Promise<ContentResponse> {
    const encodedPath = encodeURIComponent(path);
    return this.request<ContentResponse>(`/api/sessions/${sessionId}/files/${encodedPath}`);
  }

  async getAiHint(sessionId: string): Promise<{ hint: string }> {
    return this.request<{ hint: string }>(`/api/sessions/${sessionId}/hint`);
  }

  async updateFileContent(sessionId: string, path: string, data: UpdateContentRequest): Promise<UpdateContentResponse> {
    const encodedPath = encodeURIComponent(path);
    return this.request<UpdateContentResponse>(`/api/sessions/${sessionId}/files/${encodedPath}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteFile(sessionId: string, path: string): Promise<DeleteFileResponse> {
    const encodedPath = encodeURIComponent(path);
    return this.request<DeleteFileResponse>(`/api/sessions/${sessionId}/files/${encodedPath}`, {
      method: 'DELETE',
    });
  }

  async getSessionHistory(id: string): Promise<HistoryVersion[]> {
    return this.request<HistoryVersion[]>(`/api/sessions/${id}/history`);
  }

  async restoreSessionVersion(id: string, version: number): Promise<RestoreVersionResponse> {
    return this.request<RestoreVersionResponse>(`/api/sessions/${id}/restore`, {
      method: 'POST',
      body: JSON.stringify({ version }),
    });
  }

  // --- ЛИДЕРБОРД И ПРОФИЛЬ ---
  async getLeaderboard(id: string): Promise<LeaderboardEntry[]> {
    return this.request<LeaderboardEntry[]>(`/api/sessions/${id}/leaderboard`);
  }

  async getSessionProfile(id: string): Promise<UserProfile> {
    return this.request<UserProfile>(`/api/sessions/${id}/profile`);
  }

  async updateSessionProfile(id: string, data: UserProfile): Promise<UserProfile> {
    return this.request<UserProfile>(`/api/sessions/${id}/profile`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async getParticipants(sessionId: string): Promise<DbParticipant[]> {
    return this.request<DbParticipant[]>(`/api/sessions/${sessionId}/participants`);
  }

  async inviteUser(sessionId: string, username: string): Promise<{ status: string; user: string }> {
    return this.request<{ status: string; user: string }>(`/api/sessions/${sessionId}/invite`, {
      method: 'POST',
      body: JSON.stringify({ username }),
    });
  }

  async getInviteLink(sessionId: string): Promise<InviteLinkResponse> {
    return this.request<InviteLinkResponse>(`/api/sessions/${sessionId}/invite-link`);
  }

async joinByLink(inviteToken: string): Promise<JoinByLinkResponse> {
    console.log('[API] Joining with token:', inviteToken);
    try {
        const result = await this.request<JoinByLinkResponse>('/api/sessions/join-by-invite', {
            method: 'POST',
            body: JSON.stringify({ invite_token: inviteToken }),
        });
        console.log('[API] Join result:', result);
        return result;
    } catch (error) {
        console.error('[API] Join error:', error);
        throw error;
    }
}

  async removeParticipant(sessionId: string, userId: string): Promise<{ status: string; user_id: string }> {
    return this.request<{ status: string; user_id: string }>(`/api/sessions/${sessionId}/participants/${userId}`, {
      method: 'DELETE',
    });
  }

}

export const api = new ApiClient(resolveApiBaseUrl());
