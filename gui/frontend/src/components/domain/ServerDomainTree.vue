<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import TreeNode from './TreeNode.vue'
import { workspaceStore } from '../../stores/workspaceStore'
import { viewStore } from '../../stores/viewStore'
import type { ServerDomain, Server, ServerStatus } from '../../types/domain'

// Context menu state
const contextMenu = ref<{
  show: boolean
  x: number
  y: number
  type: 'domain' | 'server' | 'empty'
  domainId?: string
  serverId?: string
}>({
  show: false,
  x: 0,
  y: 0,
  type: 'empty'
})

// Tooltip state
const tooltip = ref<{
  show: boolean
  x: number
  y: number
  text: string
}>({
  show: false,
  x: 0,
  y: 0,
  text: ''
})

// Edit dialog state
const editDialog = ref<{
  show: boolean
  mode: 'addDomain' | 'renameDomain' | 'addServer' | 'editServer'
  domainId?: string
  serverId?: string
  name: string
  port: number
  description: string
}>({
  show: false,
  mode: 'addDomain',
  name: '',
  port: 8080,
  description: ''
})

// Status refresh interval
let statusInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  // Initial status fetch
  workspaceStore.refreshServerStatuses()
  // Poll for status updates every 2 seconds
  statusInterval = setInterval(() => {
    workspaceStore.refreshServerStatuses()
  }, 2000)
})

onUnmounted(() => {
  if (statusInterval) {
    clearInterval(statusInterval)
  }
})

// Get server status
function getStatus(serverId: string): ServerStatus {
  return workspaceStore.getServerStatus(serverId)
}
// Get status indicator emoji
function getStatusIndicator(serverId: string): string {
  const status = getStatus(serverId)
  switch (status) {
    case 'running': return '🟢'
    case 'starting': return '🟡'
    case 'stopping': return '🟡'
    case 'error': return '⚠️'
    default: return '🔴'
  }
}

// Get short error message for label display
function getShortError(serverId: string): string {
  const status = getStatus(serverId)
  if (status === 'error') {
    const error = workspaceStore.getServerError(serverId)
    if (error) {
      // Truncate to 30 chars for display in label
      return error.length > 30 ? error.substring(0, 30) + '...' : error
    }
  }
  return ''
}

function getStatusTooltip(serverId: string): string {
  const status = getStatus(serverId)
  if (status === 'error') {
    const error = workspaceStore.getServerError(serverId)
    return error ? `Error: ${error}` : 'Error'
  }
  return status
}

// Show error tooltip
function showErrorTooltip(e: MouseEvent, serverId: string) {
  const error = workspaceStore.getServerError(serverId)
  if (error) {
    tooltip.value = {
      show: true,
      x: e.clientX + 10,
      y: e.clientY + 10,
      text: error
    }
  }
}

function hideTooltip() {
  tooltip.value.show = false
}

// Server actions
async function handleStartServer(domainId: string, serverId: string) {
  try {
    await workspaceStore.startServer(domainId, serverId)
  } catch (err) {
    console.error('Failed to start server:', err)
    // Manually set error state in store so tooltip shows it
    // The backend emits an error event, but sometimes the promise rejection happens first
    const msg = (err as Error).message || String(err)
    workspaceStore.serverStatuses.value[serverId] = 'error'
    workspaceStore.serverErrors.value[serverId] = msg
  }
  hideContextMenu()
}

async function handleStopServer(domainId: string, serverId: string) {
  try {
    await workspaceStore.stopServer(domainId, serverId)
  } catch (err) {
    console.error('Failed to stop server:', err)
    alert('Failed to stop server: ' + (err as Error).message)
  }
  hideContextMenu()
}

const serverDomains = computed(() => workspaceStore.serverDomains.value)

// Context menu handlers
function showDomainContextMenu(e: MouseEvent, domain: ServerDomain) {
  contextMenu.value = {
    show: true,
    x: e.clientX,
    y: e.clientY,
    type: 'domain',
    domainId: domain.id
  }
}

function showServerContextMenu(e: MouseEvent, domainId: string, server: Server) {
  contextMenu.value = {
    show: true,
    x: e.clientX,
    y: e.clientY,
    type: 'server',
    domainId,
    serverId: server.id
  }
}

function showEmptyContextMenu(e: MouseEvent) {
  contextMenu.value = {
    show: true,
    x: e.clientX,
    y: e.clientY,
    type: 'empty'
  }
}

function hideContextMenu() {
  contextMenu.value.show = false
}

// CRUD operations
function openAddDomainDialog() {
  editDialog.value = {
    show: true,
    mode: 'addDomain',
    name: 'New Server Domain',
    port: 8080,
    description: ''
  }
  hideContextMenu()
}

function openRenameDomainDialog(domainId: string) {
  const domain = workspaceStore.getServerDomain(domainId)
  if (!domain) return
  editDialog.value = {
    show: true,
    mode: 'renameDomain',
    domainId,
    name: domain.name,
    port: 8080,
    description: ''
  }
  hideContextMenu()
}

function deleteDomain(domainId: string) {
  if (confirm('Delete this server domain and all its servers?')) {
    workspaceStore.deleteServerDomain(domainId)
  }
  hideContextMenu()
}

function openAddServerDialog(domainId: string) {
  editDialog.value = {
    show: true,
    mode: 'addServer',
    domainId,
    name: 'New Server',
    port: 8080,
    description: ''
  }
  hideContextMenu()
}

function openEditServerDialog(domainId: string, serverId: string) {
  const server = workspaceStore.getServer(domainId, serverId)
  if (!server) return
  editDialog.value = {
    show: true,
    mode: 'editServer',
    domainId,
    serverId,
    name: server.name,
    port: server.port,
    description: server.description || ''
  }
  hideContextMenu()
}

async function deleteServer(domainId: string, serverId: string) {
  try {
    await workspaceStore.deleteServer(domainId, serverId)
  } catch (err) {
    console.error('Failed to delete server:', err)
  }
  hideContextMenu()
}

// Dialog actions
function saveDialog() {
  const { mode, domainId, serverId, name, port, description } = editDialog.value

  switch (mode) {
    case 'addDomain':
      workspaceStore.addServerDomain(name)
      break
    case 'renameDomain':
      if (domainId) {
        const domain = workspaceStore.getServerDomain(domainId)
        if (domain) {
          workspaceStore.updateServerDomain({ ...domain, name })
        }
      }
      break
    case 'addServer':
      if (domainId) {
        workspaceStore.addServer(domainId, { name, port, description })
      }
      break
    case 'editServer':
      if (domainId && serverId) {
        const existingServer = workspaceStore.getServer(domainId, serverId)
        if (existingServer) {
          workspaceStore.updateServer(domainId, {
            ...existingServer,
            name,
            port,
            description
          })
        }
      }
      break
  }

  editDialog.value.show = false
}

function cancelDialog() {
  editDialog.value.show = false
}

// Close context menu when clicking outside
function handleGlobalClick() {
  hideContextMenu()
}
</script>

<template>
  <div class="server-domain-tree" @click="handleGlobalClick" @contextmenu.prevent="showEmptyContextMenu">
    <!-- Server domains -->
    <TreeNode
      v-for="domain in serverDomains"
      :key="domain.id"
      :label="domain.name"
      icon="📂"
      :expandable="true"
      :expanded="viewStore.isNodeExpanded(`server-domain-${domain.id}`)"
      :editable="true"
      @contextmenu="showDomainContextMenu($event, domain)"
      @rename="(name) => workspaceStore.updateServerDomain({ ...domain, name })"
      @update:expanded="(val) => viewStore.setNodeExpanded(`server-domain-${domain.id}`, val)"
    >
      <!-- Servers in this domain -->
      <TreeNode
        v-for="server in domain.servers"
        :key="server.id"
        :label="`${server.name} (:${server.port})`"
        icon="🖥️"
        :level="1"
        :editable="false"
        @contextmenu="showServerContextMenu($event, domain.id, server)"
        @dblclick="openEditServerDialog(domain.id, server.id)"
      >
        <template #suffix>
          <!-- Status icon with custom tooltip -->
          <span 
            style="margin-left: 4px; cursor: help;"
            @mouseenter="getStatus(server.id) === 'error' ? showErrorTooltip($event, server.id) : null"
            @mouseleave="hideTooltip"
          >
            {{ getStatusIndicator(server.id) }}
          </span>
        </template>
      </TreeNode>

      <!-- Add server button -->
      <div class="add-item" @click.stop="openAddServerDialog(domain.id)">
        <span class="add-icon">+</span>
        <span class="add-label">Add Server</span>
      </div>
    </TreeNode>

    <!-- Add domain button -->
    <div class="add-item root" @click.stop="openAddDomainDialog">
      <span class="add-icon">+</span>
      <span class="add-label">Add Server Domain</span>
    </div>

    <!-- Context Menu -->
    <Teleport to="body">
      <div
        v-if="contextMenu.show"
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        @click.stop
      >
        <template v-if="contextMenu.type === 'domain'">
          <div class="menu-item" @click="openAddServerDialog(contextMenu.domainId!)">
            Add Server
          </div>
          <div class="menu-item" @click="openRenameDomainDialog(contextMenu.domainId!)">
            Rename
          </div>
          <div class="menu-divider"></div>
          <div class="menu-item danger" @click="deleteDomain(contextMenu.domainId!)">
            Delete
          </div>
        </template>
        <template v-else-if="contextMenu.type === 'server'">
          <div
            v-if="getStatus(contextMenu.serverId!) !== 'running'"
            class="menu-item success"
            @click="handleStartServer(contextMenu.domainId!, contextMenu.serverId!)"
          >
            ▶ Start Server
          </div>
          <div
            v-else
            class="menu-item warning"
            @click="handleStopServer(contextMenu.domainId!, contextMenu.serverId!)"
          >
            ⏹ Stop Server
          </div>
          <div class="menu-divider"></div>
          <div class="menu-item" @click="openEditServerDialog(contextMenu.domainId!, contextMenu.serverId!)">
            Edit
          </div>
          <div class="menu-divider"></div>
          <div class="menu-item danger" @click="deleteServer(contextMenu.domainId!, contextMenu.serverId!)">
            Delete
          </div>
        </template>
        <template v-else>
          <div class="menu-item" @click="openAddDomainDialog">
            Add Server Domain
          </div>
        </template>
      </div>
    </Teleport>

    <!-- Custom Tooltip -->
    <Teleport to="body">
      <div 
        v-if="tooltip.show" 
        class="custom-tooltip"
        :style="{ left: tooltip.x + 'px', top: tooltip.y + 'px' }"
      >
        {{ tooltip.text }}
      </div>
    </Teleport>

    <!-- Edit Dialog -->
    <Teleport to="body">
      <div v-if="editDialog.show" class="dialog-overlay" @click="cancelDialog">
        <div class="dialog" @click.stop>
          <div class="dialog-header">
            {{ editDialog.mode === 'addDomain' ? 'Add Server Domain' :
               editDialog.mode === 'renameDomain' ? 'Rename Domain' :
               editDialog.mode === 'addServer' ? 'Add Server' : 'Edit Server' }}
          </div>
          <div class="dialog-body">
            <div class="form-group">
              <label>Name</label>
              <input v-model="editDialog.name" type="text" placeholder="Name" />
            </div>
            <template v-if="editDialog.mode === 'addServer' || editDialog.mode === 'editServer'">
              <div class="form-group">
                <label>Port</label>
                <input v-model.number="editDialog.port" type="number" min="1" max="65535" placeholder="8080" />
              </div>
              <div class="form-group">
                <label>Description</label>
                <input v-model="editDialog.description" type="text" placeholder="Optional description" />
              </div>
            </template>
          </div>
          <div class="dialog-footer">
            <button class="btn btn-secondary" @click="cancelDialog">Cancel</button>
            <button class="btn btn-primary" @click="saveDialog">Save</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.server-domain-tree {
  min-height: 100px;
}

.add-item {
  display: flex;
  align-items: center;
  padding: 4px 8px 4px 42px;
  cursor: pointer;
  color: #888;
  font-size: 12px;
  gap: 4px;
}

.add-item.root {
  padding-left: 12px;
  margin-top: 4px;
}

.add-item:hover {
  color: #4285f4;
  background: rgba(66, 133, 244, 0.1);
}

.add-icon {
  font-size: 14px;
  font-weight: bold;
}

.add-label {
  font-size: 12px;
}

/* Context Menu */
.context-menu {
  position: fixed;
  background: #2a2a2a;
  border: 1px solid #444;
  border-radius: 4px;
  padding: 4px 0;
  min-width: 150px;
  z-index: 1000;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.menu-item {
  padding: 8px 16px;
  cursor: pointer;
  font-size: 13px;
  color: #e0e0e0;
}

.menu-item:hover {
  background: rgba(255, 255, 255, 0.1);
}

.menu-item.danger {
  color: #f44336;
}

.menu-item.success {
  color: #4caf50;
}

.menu-item.warning {
  color: #ff9800;
}

.menu-divider {
  height: 1px;
  background: #444;
  margin: 4px 0;
}

/* Custom Tooltip */
.custom-tooltip {
  position: fixed;
  background: #330000;
  color: #ff4444;
  border: 1px solid #ff4444;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 12px;
  z-index: 2000;
  pointer-events: none;
  max-width: 300px;
  word-wrap: break-word;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
}

/* Dialog */
.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1001;
}

.dialog {
  background: #2a2a2a;
  border: 1px solid #444;
  border-radius: 8px;
  width: 400px;
  max-width: 90vw;
}

.dialog-header {
  padding: 16px;
  font-size: 16px;
  font-weight: 500;
  border-bottom: 1px solid #444;
  color: #e0e0e0;
}

.dialog-body {
  padding: 16px;
}

.form-group {
  margin-bottom: 12px;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  font-size: 12px;
  color: #888;
  margin-bottom: 4px;
}

.form-group input {
  width: 100%;
  padding: 8px 12px;
  background: #1e1e1e;
  border: 1px solid #444;
  border-radius: 4px;
  color: #e0e0e0;
  font-size: 14px;
}

.form-group input:focus {
  outline: none;
  border-color: #4285f4;
}

.dialog-footer {
  padding: 12px 16px;
  border-top: 1px solid #444;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btn {
  padding: 8px 16px;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  border: none;
}

.btn-secondary {
  background: #444;
  color: #e0e0e0;
}

.btn-secondary:hover {
  background: #555;
}

.btn-primary {
  background: #4285f4;
  color: white;
}

.btn-primary:hover {
  background: #5a95f5;
}
</style>