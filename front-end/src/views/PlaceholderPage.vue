<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ title }}</h1>
        <p>{{ description }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const title = computed(() => route.meta.title || 'Page')
const description = computed(() => route.meta.description || '')
</script>

<style scoped>
.page-head p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-dim);
}
</style>
