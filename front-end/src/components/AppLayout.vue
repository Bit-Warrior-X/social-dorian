<template>
  <div class="shell">
    <h2 class="sr-only">Admin page with a left sidebar for managing connected social accounts</h2>

    <div v-if="menuOpen" class="backdrop" @click="menuOpen = false" />

    <div class="shell__frame" :class="{ 'is-open': menuOpen }">
      <AppSidebar />
      <div class="content">
        <button class="menu-btn" type="button" aria-label="Open menu" @click="menuOpen = true">
          <i class="ti ti-menu-2" aria-hidden="true" />
        </button>
        <RouterView />
      </div>
    </div>

    <AppNotifications />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppNotifications from './AppNotifications.vue'
import AppSidebar from './AppSidebar.vue'

const route = useRoute()
const menuOpen = ref(false)

watch(() => route.path, () => {
  menuOpen.value = false
})
</script>

<style scoped>
.shell {
  height: 100%;
  min-height: 100vh;
  background: var(--bg);
}

.shell__frame {
  display: flex;
  min-height: 100vh;
}

.content {
  flex: 1;
  min-width: 0;
  background: var(--bg);
  padding: 1.25rem;
}

.menu-btn {
  display: none;
  margin-bottom: 1rem;
  padding: 6px 8px;
  border: 0.5px solid var(--hairline-strong);
  border-radius: var(--radius);
  background: var(--panel);
  color: var(--text);
}

.menu-btn .ti {
  font-size: 18px;
}

.backdrop {
  display: none;
}

@media (max-width: 800px) {
  .shell__frame :deep(.sidebar) {
    position: fixed;
    inset: 0 auto 0 0;
    z-index: 40;
    transform: translateX(-100%);
    transition: transform 0.2s ease;
  }

  .shell__frame.is-open :deep(.sidebar) {
    transform: translateX(0);
  }

  .menu-btn {
    display: inline-flex;
  }

  .backdrop {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 30;
    background: rgba(0, 0, 0, 0.55);
  }
}
</style>
