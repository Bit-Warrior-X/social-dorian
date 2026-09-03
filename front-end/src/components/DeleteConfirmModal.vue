<template>
  <ModalDialog :open="open" title="Disconnect account" title-id="delete-title" compact @close="emit('close')">
    <p class="copy">
      This disconnects {{ accountLabel }} and revokes its access. This can't be undone.
    </p>
    <div class="actions">
      <button class="btn" type="button" @click="emit('close')">Cancel</button>
      <button class="btn btn-danger" type="button" :disabled="saving" @click="emit('confirm')">
        {{ saving ? 'Disconnecting…' : 'Disconnect' }}
      </button>
    </div>
  </ModalDialog>
</template>

<script setup>
import { computed } from 'vue'
import ModalDialog from './ModalDialog.vue'
import { accountDisplayName } from '../constants/accounts'

const props = defineProps({
  open: { type: Boolean, default: false },
  account: { type: Object, default: null },
  saving: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'confirm'])

const accountLabel = computed(() => accountDisplayName(props.account))
</script>

<style scoped>
.copy {
  font-size: 14px;
  color: var(--text-dim);
  margin: 0 0 1rem;
  line-height: 1.6;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
