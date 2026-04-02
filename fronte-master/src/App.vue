<template>
  <!-- Главная точка монтирования страниц приложения -->
  <router-view />
</template>

<script setup lang="ts">
// Никакой сложной логики здесь не нужно,
// роутер сам подставит нужный View (Auth, Home или Session).
</script>

<style>
/* Подключаем нужные шрифты прямо из Google Fonts */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;700&display=swap');

/* Глобальный сброс для всего приложения, чтобы IDE занимало ровно 100% экрана */
html, body, #app {
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0;
  overflow: hidden; /* Защита от появления двойных скроллов */
  background-color: var(--bg-color);
  color: var(--text-main);
}
</style>