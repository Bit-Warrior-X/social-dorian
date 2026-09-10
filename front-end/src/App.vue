<template>
  <div class="app">
    <RouterView />
    <AppNotifications />
  </div>
</template>

<script setup>
import { RouterView } from 'vue-router'
import AppNotifications from './components/AppNotifications.vue'
</script>
