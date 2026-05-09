<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { workspaceStore } from '../../stores/workspaceStore'

const props = withDefaults(defineProps<{
  serverDomainId?: string
  serverId?: string
  path?: string
  disabled?: boolean
}>(), {
  serverDomainId: '',
  serverId: '',
  path: '',
  disabled: false
})

const emit = defineEmits<{
  (e: 'update:serverDomainId', value: string): void
  (e: 'update:serverId', value: string): void
  (e: 'update:path', value: string): void
  (e: 'update:server', value: { serverDomainId: string; serverId: string }): void
}>()

// Dropdown state
const dropdownOpen = ref(false)

// Server domains with their servers (only running servers)
const serverDomains = computed(() => {
  return workspaceStore.serverDomains.value.map(domain => ({
    ...domain,
    servers: domain.servers.filter(server => 
      workspaceStore.getServerStatus(server.id) === 'running'
    )
  })).filter(domain => domain.servers.length > 0) // Only show domains with running servers
})

// Original server domains (for checking if any servers exist)
const hasAnyServers = computed(() => 
  workspaceStore.serverDomains.value.some(domain => domain.servers.length > 0)
)

// Current selection display
const selectionDisplay = computed(() => {
  if (!props.serverDomainId || !props.serverId) {
    return 'External URL'
  }
  return workspaceStore.getServerDisplayName(props.serverDomainId, props.serverId)
})

// Get display port for a server (actual port if running with port 0, otherwise configured port)
function getDisplayPort(serverId: string, configuredPort: number): number {
  const actualPort = workspaceStore.serverActualPorts.value[serverId]
  return actualPort > 0 ? actualPort : configuredPort
}

// Selected server's base URL (uses actual port for port 0)
const selectedServerBaseURL = computed(() => {
  if (!props.serverDomainId || !props.serverId) {
    return ''
  }
  const server = workspaceStore.getServer(props.serverDomainId, props.serverId)
  if (!server) return ''
  const port = getDisplayPort(props.serverId, server.port)
  return `http://localhost:${port}`
})

// Is external mode
const isExternal = computed(() => !props.serverDomainId || !props.serverId)

// Select external URL mode
function selectExternal() {
  emit('update:server', { serverDomainId: '', serverId: '' })
  dropdownOpen.value = false
}

// Select a specific server
function selectServer(domainId: string, serverId: string) {
  console.log('selectServer called:', { domainId, serverId })
  emit('update:server', { serverDomainId: domainId, serverId: serverId })
  dropdownOpen.value = false
}

// Update path
function updatePath(value: string) {
  console.log('[ServerSelector] updatePath called with:', value)
  emit('update:path', value)
}

// Toggle dropdown
function toggleDropdown() {
  if (props.disabled) return
  dropdownOpen.value = !dropdownOpen.value
}

// Close dropdown when clicking outside
function handleClickOutside(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('.server-selector')) {
    dropdownOpen.value = false
  }
}

// Add/remove global click listener
watch(dropdownOpen, (open) => {
  if (open) {
    setTimeout(() => {
      document.addEventListener('click', handleClickOutside)
    }, 0)
  } else {
    document.removeEventListener('click', handleClickOutside)
  }
})
</script>

<template>
  <div class="server-selector" :class="{ disabled: props.disabled }">
    <!-- 单行布局：Target 下拉 + URL 输入 -->
    <div class="dropdown-container">
      <button class="dropdown-trigger" @click="toggleDropdown" :disabled="props.disabled">
        <span class="trigger-text">{{ selectionDisplay }}</span>
        <span class="trigger-arrow">▼</span>
      </button>

      <!-- Dropdown menu -->
      <div v-if="dropdownOpen" class="dropdown-menu">
        <!-- External URL option -->
        <div 
          class="menu-item" 
          :class="{ selected: isExternal }"
          @click="selectExternal"
        >
          <span class="item-icon">🌐</span>
          <span class="item-text">External URL</span>
        </div>

        <!-- Server domains with servers -->
        <template v-for="domain in serverDomains" :key="domain.id">
          <div class="menu-group-header">
            <span class="group-icon">📂</span>
            <span class="group-name">{{ domain.name }}</span>
          </div>
          <div 
            v-for="server in domain.servers" 
            :key="server.id"
            class="menu-item server-item"
            :class="{ selected: props.serverDomainId === domain.id && props.serverId === server.id }"
            @click="selectServer(domain.id, server.id)"
          >
            <span class="item-icon">🖥️</span>
            <span class="item-text">{{ server.name }}</span>
            <span class="item-url">:{{ getDisplayPort(server.id, server.port) }}</span>
          </div>
        </template>

        <!-- Empty state -->
        <div v-if="serverDomains.length === 0" class="menu-empty">
          <template v-if="hasAnyServers">
            No running servers. Start a server from the Server Domains panel.
          </template>
          <template v-else>
            No server domains configured
          </template>
        </div>
      </div>
    </div>

    <!-- URL 输入框 -->
    <template v-if="isExternal">
      <input
        :value="path"
        @input="updatePath(($event.target as HTMLInputElement).value)"
        type="text"
        class="url-input full"
        placeholder="https://api.example.com/users"
        :disabled="props.disabled"
      />
    </template>
    <template v-else>
      <span class="base-url">{{ selectedServerBaseURL }}</span>
      <input
        :value="path"
        @input="updatePath(($event.target as HTMLInputElement).value)"
        type="text"
        class="url-input path"
        placeholder="/api/users"
        :disabled="props.disabled"
      />
    </template>
  </div>
</template>

<style scoped>
.server-selector {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 0;
  flex: 1;
}

.dropdown-container {
  position: relative;
  flex-shrink: 0;
}

.dropdown-trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: #3c3c3c;
  border: 1px solid #555;
  border-right: none;
  border-radius: 4px 0 0 4px;
  color: #d4d4d4;
  font-size: 13px;
  cursor: pointer;
  text-align: left;
  min-width: 140px;
  max-width: 200px;
  height: 36px;
  box-sizing: border-box;
}

.dropdown-trigger:hover {
  border-color: #666;
}

.dropdown-trigger:focus {
  outline: none;
  border-color: #007acc;
}

.trigger-text {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.trigger-arrow {
  font-size: 10px;
  color: #888;
  margin-left: 8px;
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 4px;
  background: #2a2a2a;
  border: 1px solid #444;
  border-radius: 4px;
  max-height: 300px;
  overflow-y: auto;
  z-index: 100;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  gap: 8px;
}

.menu-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.menu-item.selected {
  background: rgba(66, 133, 244, 0.2);
}

.menu-item.server-item {
  padding-left: 24px;
}

.item-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.item-text {
  flex: 1;
  font-size: 13px;
  color: #e0e0e0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-url {
  font-size: 11px;
  color: #888;
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-group-header {
  display: flex;
  align-items: center;
  padding: 6px 12px;
  gap: 8px;
  background: #252525;
  border-top: 1px solid #333;
  margin-top: 4px;
}

.menu-group-header:first-child {
  margin-top: 0;
  border-top: none;
}

.group-icon {
  font-size: 12px;
}

.group-name {
  font-size: 11px;
  font-weight: 600;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.menu-empty {
  padding: 16px;
  text-align: center;
  color: #888;
  font-size: 13px;
}

.base-url {
  padding: 8px 10px;
  background: #2d2d30;
  border: 1px solid #555;
  border-right: none;
  border-left: none;
  color: #858585;
  font-size: 13px;
  white-space: nowrap;
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  height: 36px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
}

.url-input {
  flex: 1;
  padding: 8px 10px;
  background: #3c3c3c;
  border: 1px solid #555;
  color: #d4d4d4;
  font-size: 13px;
  height: 36px;
  box-sizing: border-box;
}

.url-input.full {
  border-radius: 0 4px 4px 0;
}

.url-input.path {
  border-radius: 0 4px 4px 0;
}

.url-input:focus {
  outline: none;
  border-color: #007acc;
}

.url-input::placeholder {
  color: #6e6e6e;
}

/* Disabled state */
.server-selector.disabled {
  opacity: 0.5;
  pointer-events: none;
}

.dropdown-trigger:disabled {
  cursor: not-allowed;
}

.url-input:disabled {
  cursor: not-allowed;
  background: #2d2d2d;
}
</style>