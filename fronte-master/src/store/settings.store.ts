import {ref, watch} from "vue";
import {defineStore} from "pinia";

export const useSettingsStore = defineStore('settings', () => {
    const theme = ref(localStorage.getItem('ide-theme') || 'vs-dark');
    const fontSize = ref(Number(localStorage.getItem('ide-font-size')) || 14);
    const lineHeight = ref(Number(localStorage.getItem('ide-line-height')) || 1.5);

    watch([theme, fontSize, lineHeight], () => {
        localStorage.setItem('ide-theme', theme.value);
        localStorage.setItem('ide-font-size', fontSize.value.toString());
        localStorage.setItem('ide-line-height', lineHeight.value.toString());
        applySettings();
    }, { immediate: true });

    function applySettings() {
        const root = document.documentElement;
        root.style.setProperty('--editor-font-size', fontSize.value + 'px');
        root.style.setProperty('--editor-line-height', lineHeight.value.toString());
        // Для Monaco темы меняются через API, но для UI используем атрибут
        root.setAttribute('data-theme', theme.value);
    }

    return { theme, fontSize, lineHeight };
});