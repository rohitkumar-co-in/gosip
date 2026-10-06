<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'

defineProps<{ label: string }>()
const emit = defineEmits<{ close: [] }>()
const panel = ref<HTMLElement>()
const validation = ref('')
let previousFocus: HTMLElement | null = null
let previousOverflow = ''

function invalid(event: Event) {
  event.preventDefault()
  const field = event.target as HTMLInputElement
  validation.value = field.validationMessage || 'Please check the required fields.'
}
function keydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
  if (event.key !== 'Tab') return
  const controls = Array.from(panel.value?.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href], [tabindex="0"]') || []).filter(el => el.getClientRects().length)
  const first = controls[0], last = controls[controls.length - 1]
  if (!first || !last) { event.preventDefault(); panel.value?.focus(); return }
  if (event.shiftKey && (document.activeElement === first || document.activeElement === panel.value)) { event.preventDefault(); last.focus() }
  else if (!event.shiftKey && (document.activeElement === last || document.activeElement === panel.value)) { event.preventDefault(); first.focus() }
}
onMounted(() => {
  previousFocus = document.activeElement as HTMLElement | null
  previousOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  panel.value?.focus()
})
onBeforeUnmount(() => {
  document.body.style.overflow = previousOverflow
  previousFocus?.focus()
})
</script>

<template>
  <Teleport to="body">
    <div class="console-dialog-overlay" @click.self="emit('close')" @keydown="keydown">
      <section ref="panel" class="console-dialog-panel" role="dialog" aria-modal="true" :aria-label="label" tabindex="-1" @invalid.capture="invalid" @input="validation=''">
        <p v-if="validation" role="alert" class="mb-4 rounded-lg bg-red-50 p-3 text-sm text-red-800">{{ validation }}</p>
        <slot />
      </section>
    </div>
  </Teleport>
</template>
