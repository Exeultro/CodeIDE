import { api } from './api';
import type { WsClientMessage, WsServerMessage } from './types';

type Listener = (payload: any) => void;
type BinaryListener = (data: ArrayBuffer) => void;

function resolveWsBaseUrl(): string {
  // const envUrl = (import.meta.env.VITE_WS_URL as string | undefined)?.trim();
  // if (envUrl) return envUrl.replace(/\/$/, '');
  // if (typeof window !== 'undefined') {
  //   const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  //   return `${protocol}//${window.location.host}/ws`;
  // }
  return 'ws://localhost:8080/ws';
}

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private url: string;
  private listeners: Map<string, Set<Listener>> = new Map();
  private binaryListeners: Set<BinaryListener> = new Set();

  constructor(baseUrl: string = resolveWsBaseUrl()) {
    this.url = baseUrl;
  }

  public isOpen(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  public connect(sessionId: string, userId: string, username: string) {
    this.disconnect(false);

    const params = new URLSearchParams({
      room: sessionId,
      user: userId,
      username,
    });

    const token = api.getToken();
    if (token) params.set('token', token);

    this.ws = new WebSocket(`${this.url}?${params.toString()}`);
    this.ws.binaryType = 'arraybuffer';

    this.ws.onopen = () => {
      this.send({ type: 'join', payload: { username } });
    };

    this.ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        this.binaryListeners.forEach((cb) => cb(event.data));
        return;
      }

      try {
        const data = JSON.parse(event.data) as WsServerMessage | Record<string, any>;
        const type = String((data as any).type || '');
        if (!type) return;

        let payload = (data as any).payload;
        if (payload === undefined) {
          payload = { ...data };
          delete payload.type;
        }
        this.emit(type, payload);
      } catch (error) {
        console.error('[WS] JSON Parse Error:', error);
      }
    };

    this.ws.onclose = () => console.log('[WS] Disconnected');
    this.ws.onerror = (err) => console.error('[WS] Error:', err);
  }

  public disconnect(clearListeners = true) {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    if (clearListeners) {
      this.listeners.clear();
      this.binaryListeners.clear();
    }
  }

  public send(message: WsClientMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  public sendBinary(data: Uint8Array | ArrayBuffer) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(data);
    }
  }

  public on<T extends WsServerMessage['type']>(type: T, callback: (payload: any) => void) {
    if (!this.listeners.has(type)) this.listeners.set(type, new Set());
    this.listeners.get(type)!.add(callback);
    return () => {
      this.listeners.get(type)?.delete(callback);
    };
  }

  public onBinary(callback: BinaryListener) {
    this.binaryListeners.add(callback);
    return () => {
      this.binaryListeners.delete(callback);
    };
  }

  private emit(type: string, payload: any) {
    if (this.listeners.has(type)) {
      this.listeners.get(type)!.forEach((cb) => cb(payload));
    }
  }
}

export const wsClient = new WebSocketClient();
