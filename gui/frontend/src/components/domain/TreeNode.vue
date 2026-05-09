<script setup lang="ts">
import { ref, computed, watch } from 'vue'

interface Props {
  label: string
  icon?: string
  expandable?: boolean
  expanded?: boolean
  selected?: boolean
  level?: number
  editable?: boolean
  method?: string  // HTTP method for request nodes (GET, POST, etc.)
  title?: string   // Tooltip text
}

const props = withDefaults(defineProps<Props>(), {
  icon: '',
  expandable: false,
  expanded: false,
  selected: false,
  level: 0,
  editable: false,
  method: ''
})

const emit = defineEmits<{
  (e: 'toggle'): void
  (e: 'select'): void
  (e: 'contextmenu', event: MouseEvent): void
  (e: 'rename', newName: string): void
  (e: 'dblclick'): void
  (e: 'update:expanded', value: boolean): void
}>()

const isExpanded = ref(props.expanded)

// 同步外部 props 变化
watch(() => props.expanded, (newVal) => {
  isExpanded.value = newVal
})
const isEditing = ref(false)
const editValue = ref(props.label)

const paddingLeft = computed(() => `${12 + props.level * 16}px`)

function handleClick() {
  if (props.expandable) {
    isExpanded.value = !isExpanded.value
    emit('toggle')
    emit('update:expanded', isExpanded.value)
  }
  emit('select')
}

function handleContextMenu(e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation() // 阻止事件冒泡到父级节点
  emit('contextmenu', e)
}

function handleDblClick() {
  if (props.editable) {
    startEditing()
  }
  emit('dblclick')
}

function startEditing() {
  editValue.value = props.label
  isEditing.value = true
}

function finishEditing() {
  if (isEditing.value && editValue.value.trim()) {
    emit('rename', editValue.value.trim())
  }
  isEditing.value = false
}

function cancelEditing() {
  isEditing.value = false
  editValue.value = props.label
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    finishEditing()
  } else if (e.key === 'Escape') {
    cancelEditing()
  }
}

// Expose for parent components
defineExpose({
  startEditing,
  isExpanded
})
</script>

<template>
  <div class="tree-node" :class="{ selected, expandable }" @contextmenu="handleContextMenu">
    <div 
      class="node-content" 
      :style="{ paddingLeft }"
      :title="title"
      @click="handleClick"
      @dblclick="handleDblClick"
    >
      <!-- Expand/Collapse arrow -->
      <span v-if="expandable" class="expand-icon" :class="{ expanded: isExpanded }">
        ▶
      </span>
      <span v-else class="expand-placeholder"></span>

      <!-- Icon -->
      <span v-if="icon && !method" class="node-icon">{{ icon }}</span>

      <!-- HTTP Method badge (for request nodes) -->
      <span v-if="method" :class="['method-badge', `method-${method.toLowerCase()}`]">
        {{ method }}
      </span>

      <!-- Label or Edit input -->
      <input
        v-if="isEditing"
        ref="editInput"
        v-model="editValue"
        class="edit-input"
        @blur="finishEditing"
        @keydown="handleKeydown"
        @click.stop
        autofocus
      />
      <span v-else class="node-label">{{ label }}</span>

      <!-- Suffix slot (e.g., status icon) -->
      <slot name="suffix"></slot>
    </div>

    <!-- Children slot (collapsible) -->
    <div v-if="expandable && isExpanded" class="node-children">
      <slot></slot>
    </div>
  </div>
</template>

<style scoped>
.tree-node {
  user-select: none;
}

.node-content {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  cursor: pointer;
  border-radius: 4px;
  gap: 4px;
}

.node-content:hover {
  background: rgba(255, 255, 255, 0.05);
}

.tree-node.selected > .node-content {
  background: rgba(66, 133, 244, 0.2);
}

.expand-icon {
  font-size: 10px;
  color: #888;
  transition: transform 0.15s;
  width: 14px;
  text-align: center;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.expand-placeholder {
  width: 14px;
}

.node-icon {
  font-size: 14px;
  margin-right: 4px;
}

.node-label {
  flex: 0 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: #e0e0e0;
  margin-right: 4px;
}

.edit-input {
  flex: 1;
  background: #2a2a2a;
  border: 1px solid #4285f4;
  border-radius: 2px;
  color: #e0e0e0;
  font-size: 13px;
  padding: 2px 4px;
  outline: none;
}

.node-children {
  /* Children are indented via paddingLeft in child nodes */
}

/* Method badge - Postman style colors */
.method-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 4px;
  border-radius: 3px;
  text-transform: uppercase;
  flex-shrink: 0;
  margin-right: 4px;
}

.method-get {
  color: #61affe;
}

.method-post {
  color: #49cc90;
}

.method-put {
  color: #fca130;
}

.method-patch {
  color: #50e3c2;
}

.method-delete {
  color: #f93e3e;
}

.method-head {
  color: #9012fe;
}

.method-options {
  color: #0d5aa7;
}
</style>