/**
 * 📦 Базовые типы API
 */
export type ApiSuccess<T> = {
    success: true;
    data: T;
};

export type ApiError = {
    success: false;
    error: string;
    code: number;
};

export type ApiResponse<T> = ApiSuccess<T> | ApiError;

/**
 * 🔐 1. Аутентификация и пользователи
 */
export interface AuthRequest {
    username: string;
    password?: string; // Пароль нужен только для запроса, в ответах его нет
}

export interface AuthResponse {
    token: string;
    user_id: string;
    username: string;
}

/**
 * 📝 2. Сессии
 */
export interface Session {
    id: string;
    name: string;
    file_name: string;
    language: string;
    owner_id?: string;
    created_at: string;
    active: boolean;
    content?: string;
    version?: number;
}

export interface CreateSessionRequest {
    name: string;
    file_name: string;
    language: string;
}

export interface ContentResponse {
    content: string;
    version: number;
}

export interface UpdateContentRequest {
    content: string;
    base_version: number;
}

export interface UpdateContentResponse {
    new_version: number;
    version?: number;
    status?: 'updated';
}

export interface SessionEvent {
    id: string;
    user_id: string;
    event_type: 'join' | 'leave' | 'save';
    details: Record<string, any>;
    created_at: string;
}

export interface AiReviewLocation {
    start_line: number;
    end_line: number;
}

export interface AiReview {
    id: string;
    type: 'code_review' | string;
    location: AiReviewLocation;
    original_snippet: string;
    suggested_snippet: string;
    message: string;
    resolved: boolean;
    created_at: string;
}

/**
 * 📁 3. Файлы и папки
 */
export interface FileNode {
    id: string;
    path: string;
    is_dir: boolean;
    version?: number; // Только для файлов
    created_at: string;
    updated_at?: string;
}

export interface CreateFileRequest {
    path: string;
    is_dir: boolean;
    content?: string; // Только если is_dir: false
}

export interface DeleteFileResponse {
    status: 'deleted';
}

/**
 * 🔌 4. WebSocket (Реальное время)
 * Сообщения от КЛИЕНТА к СЕРВЕРУ
 */
export type WsClientMessage =
    | { type: 'join'; payload: { username: string } }
    | { type: 'cursor'; payload: { line: number; column: number } }
    | { type: 'save_text'; payload: { content: string } }
    | { type: 'run_code'; payload?: Record<string, never> }
    | { type: 'terminal_exec'; payload: { command: string } }
    | { type: 'select_file'; payload: { path: string } };

// Для бинарных сообщений Yjs используется тип ArrayBuffer / Uint8Array (в TS не описывается как JSON интерфейс)

/**
 * 🔌 4. WebSocket (Реальное время)
 * Сообщения от СЕРВЕРА к КЛИЕНТУ
 */
export interface WsParticipant {
    user_id: string;
    username: string;
}

export type WsServerMessage =

    | { type: 'full_state'; payload: { content: string; version?: number } | string }
    | { type: 'participants'; payload: WsParticipant[] }
    | {
    type: 'cursor_update';
    payload: {
        user_id: string;
        username: string;
        payload: { line: number; column: number };
    };
}
    | { type: 'ai_suggestion'; payload: { message?: string; created_at?: string } | string }
    | {
    type: 'code_output';
    payload: { output?: string; error?: boolean; error_msg?: string };
}
    | {
    type: 'terminal_output';
    payload: { command?: string; output?: string; error?: boolean; error_msg?: string };
}
    | {
    type: 'file_created';
    payload: { id: string; path: string; is_dir: boolean; created_at: string };
}
    | { type: 'file_updated'; payload: { path: string; version: number } }
    | { type: 'file_deleted'; payload: { path: string } }
    | { type: 'error'; payload: { message?: string } }
    | { type: 'user_invited'; payload: { user_id: string; username: string } }
    | { type: 'user_removed'; payload: { user_id: string; username: string } };


/**
 * 🚀 Пример API Сервиса (Опционально: можно использовать для понимания структуры эндпоинтов)
 * Все пути и ожидаемые типы собраны в одном месте.
 */
export interface ApiEndpoints {
    // Auth
    'POST /api/auth/register': { req: AuthRequest; res: ApiResponse<AuthResponse> };
    'POST /api/auth/login': { req: AuthRequest; res: ApiResponse<AuthResponse> };

    // Users
    'GET /api/users/me/sessions': { req: void; res: ApiResponse<Session[]> };

    // Sessions
    'POST /api/sessions': { req: CreateSessionRequest; res: ApiResponse<Session> };
    'GET /api/sessions/:id': { req: void; res: ApiResponse<Session> };
    'GET /api/sessions/:id/content': { req: void; res: ApiResponse<ContentResponse> }; // Устарело
    'PUT /api/sessions/:id/content': { req: UpdateContentRequest; res: ApiResponse<UpdateContentResponse> }; // Устарело
    'GET /api/sessions/:id/events': { req: void; res: ApiResponse<SessionEvent[]> };
    'GET /api/sessions/:id/ai-reviews': { req: void; res: ApiResponse<AiReview[]> };
    'POST /api/sessions/:id/ai-reviews/:review_id/apply': { req: void; res: ApiResponse<{ status: 'applied' }> };

    // Files
    'GET /api/sessions/:id/files': { req: void; res: ApiResponse<FileNode[]> };
    'POST /api/sessions/:id/files': { req: CreateFileRequest; res: ApiResponse<FileNode> };
    'GET /api/sessions/:id/files/:path': { req: void; res: ApiResponse<ContentResponse> };
    'PUT /api/sessions/:id/files/:path': { req: UpdateContentRequest; res: ApiResponse<UpdateContentResponse> };
    'DELETE /api/sessions/:id/files/:path': { req: void; res: ApiResponse<DeleteFileResponse> };
}



/**
 * 📜 5. История и события
 */
export interface HistoryVersion {
    version: number;
    created_at: string;
    preview: string;
}

export interface RestoreVersionRequest {
    version: number;
}

export interface RestoreVersionResponse {
    version: number;
    content: string;
}

/**
 * 🏆 6. Геймификация и профиль
 */
export interface LeaderboardEntry {
    user_id: string;
    username: string;
    display_name: string;
    points: number;
    incognito: boolean;
    nickname: string;
    updated_at: string;
}

export interface UserProfile {
    incognito: boolean;
    nickname: string;
}


/**
 * 👥 7. Участники и приглашения
 */
export interface DbParticipant {
    user_id: string;
    username: string;
    joined_at: string;
}

export interface InviteLinkResponse {
    invite_link: string;
    session_id: string;
}

export interface JoinByLinkRequest {
    invite_token: string;
}

export interface JoinByLinkResponse {
    session_id: string;
    session_name: string;
    joined: boolean;
}