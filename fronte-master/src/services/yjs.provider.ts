import * as Y from 'yjs';
import { WebsocketProvider } from 'y-websocket';
import { MonacoBinding } from 'y-monaco';
import type { editor } from 'monaco-editor';

export class YjsService {
    private doc: Y.Doc | null = null;
    private provider: WebsocketProvider | null = null;
    private binding: MonacoBinding | null = null;
    private currentFilePath: string = '';

    public setup(options: any) {
        this.destroy();
        const { sessionId, filePath, userId, username, editorInstance, initialContent, serverUrl = 'ws://localhost:1234' } = options;

        this.currentFilePath = filePath;
        this.doc = new Y.Doc();
        const yText = this.doc.getText(filePath);
        const url = `${serverUrl}/yjs?user=${userId}&room=${sessionId}`;

        try {
            this.provider = new WebsocketProvider(url, sessionId, this.doc, { connect: true });

            const awareness = this.provider.awareness;
            awareness.setLocalStateField('user', { name: username, color: '#D92525', id: userId });

            let isBound = false;

            const doBind = () => {
                if (isBound) return;
                isBound = true;

                const currentMonacoValue = editorInstance.getValue();

                // ЛОГИКА ВЛИВАНИЯ КОНТЕНТА
                if (yText.length === 0) {
                    // Если Yjs пустой И в редакторе пусто - льем из БД
                    if (!currentMonacoValue || currentMonacoValue.trim() === '') {
                        if (initialContent) {
                            console.log(`[Yjs] Инициализация контента из БД`);
                            yText.insert(0, initialContent);
                        }
                    } else {
                        // Если в редакторе уже есть текст (юзер начал писать),
                        // переносим его в Yjs, чтобы не потерять
                        console.log(`[Yjs] Сохранение локального текста в новую комнату`);
                        yText.insert(0, currentMonacoValue);
                    }
                } else {
                    // Если на сервере Yjs уже есть текст - очищаем локальный редактор,
                    // чтобы данные с сервера не приклеились к данным из БД
                    console.log(`[Yjs] Загрузка текста с сервера (очистка локального)`);
                    editorInstance.setValue('');
                }

                this.binding = new MonacoBinding(
                    yText,
                    editorInstance.getModel()!,
                    new Set([editorInstance]),
                    awareness
                );
            };

            this.provider.on('sync', (isSynced: boolean) => {
                if (isSynced) doBind();
            });

            // Тайм-аут на случай плохой связи
            setTimeout(() => { if (!isBound) doBind(); }, 2000);

            this.provider.on('status', (ev: any) => {
                console.log(`[Yjs] Состояние: ${ev.status}`);
            });

        } catch (err) {
            console.error(`[Yjs] Ошибка:`, err);
        }
    }

    public applyValue(newContent: string) {
        if (!this.doc || !this.currentFilePath) return;
        const yText = this.doc.getText(this.currentFilePath);
        this.doc.transact(() => {
            yText.delete(0, yText.length);
            yText.insert(0, newContent);
        });
    }

    public destroy() {
        if (this.binding) this.binding.destroy();
        if (this.provider) { this.provider.disconnect(); this.provider.destroy(); }
        if (this.doc) this.doc.destroy();
        this.binding = null; this.provider = null; this.doc = null;
    }
}
export const yjsService = new YjsService();