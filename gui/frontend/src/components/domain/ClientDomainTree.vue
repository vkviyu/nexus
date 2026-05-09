<script setup lang="ts">
import { ref, computed } from 'vue'
import TreeNode from './TreeNode.vue'
import { workspaceStore } from '../../stores/workspaceStore'
import { viewStore } from '../../stores/viewStore'
import type { ClientDomain, Client, SavedRequest } from '../../types/domain'

// Context menu state
const contextMenu = ref<{
  show: boolean
  x: number
  y: number
  type: 'domain' | 'client' | 'request' | 'empty'
  domainId?: string
  clientId?: string
  requestId?: string
}>({
  show: false,
  x: 0,
  y: 0,
  type: 'empty'
})

// Edit dialog state
const editDialog = ref<{
  show: boolean
  mode: 'addDomain' | 'renameDomain' | 'addClient' | 'renameClient'
  domainId?: string
  clientId?: string
  name: string
}>({
  show: false,
  mode: 'addDomain',
  name: ''
})

const clientDomains = computed(() => workspaceStore.clientDomains.value)
const activeClientDomainId = computed(() => workspaceStore.activeClientDomainId.value)
const activeClientId = computed(() => workspaceStore.activeClientId.value)
const activeTabId = computed(() => workspaceStore.activeTabId.value)

// Context menu handlers
function showDomainContextMenu(e: MouseEvent, domain: ClientDomain) {
  e.stopPropagation()
  contextMenu.value = {
    show: true,
    x: e.clientX,
    y: e.clientY,
    type: 'domain',
    domainId: domain.id
  }
}

function showClientContextMenu(e: MouseEvent, domainId: string, client: Client) {
  e.stopPropagation()
  contextMenu.value = {
    show: true,
    x: e.clientX,
    y: e.clientY,
    type: 'client',
    domainId,
    clientId: client.id
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

// Selection handlers
function selectDomain(domainId: string) {
  workspaceStore.setActiveClientDomain(domainId)
}

function selectClient(domainId: string, clientId: string) {
  if (workspaceStore.activeClientDomainId.value !== domainId) {
    workspaceStore.setActiveClientDomain(domainId)
  }
  workspaceStore.setActiveClient(clientId)
}

// CRUD operations
function openAddDomainDialog() {
  editDialog.value = {
    show: true,
    mode: 'addDomain',
    name: 'New Workspace'
  }
  hideContextMenu()
}

function openRenameDomainDialog(domainId: string) {
  const domain = workspaceStore.getClientDomain(domainId)
  if (!domain) return
  editDialog.value = {
    show: true,
    mode: 'renameDomain',
    domainId,
    name: domain.name
  }
  hideContextMenu()
}

function deleteDomain(domainId: string) {
  if (confirm('Delete this workspace and all its clients?')) {
    workspaceStore.deleteClientDomain(domainId)
  }
  hideContextMenu()
}

function openAddClientDialog(domainId: string) {
  editDialog.value = {
    show: true,
    mode: 'addClient',
    domainId,
    name: 'New Client'
  }
  hideContextMenu()
}

function openRenameClientDialog(domainId: string, clientId: string) {
  const domain = workspaceStore.getClientDomain(domainId)
  const client = domain?.clients.find(c => c.id === clientId)
  if (!client) return
  editDialog.value = {
    show: true,
    mode: 'renameClient',
    domainId,
    clientId,
    name: client.name
  }
  hideContextMenu()
}

function deleteClient(domainId: string, clientId: string) {
  if (confirm('Delete this client and all its tabs?')) {
    workspaceStore.deleteClient(clientId, domainId)
  }
  hideContextMenu()
}

// Dialog actions
function saveDialog() {
  const { mode, domainId, clientId, name } = editDialog.value

  switch (mode) {
    case 'addDomain':
      const newDomain = workspaceStore.addClientDomain(name)
      // Also add a default client to the new domain
      workspaceStore.addClient(newDomain.id, { name: 'Default Client' })
      break
    case 'renameDomain':
      if (domainId) {
        const domain = workspaceStore.getClientDomain(domainId)
        if (domain) {
          workspaceStore.updateClientDomain({ ...domain, name })
        }
      }
      break
    case 'addClient':
      if (domainId) {
        workspaceStore.addClient(domainId, { name })
      }
      break
    case 'renameClient':
      if (domainId && clientId) {
        const domain = workspaceStore.getClientDomain(domainId)
        const client = domain?.clients.find(c => c.id === clientId)
        if (client) {
          workspaceStore.updateClient({ ...client, name }, domainId)
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

// Check if a domain is the active one
function isDomainActive(domainId: string): boolean {
  return activeClientDomainId.value === domainId
}

// Check if a client is the active one
function isClientActive(domainId: string, clientId: string): boolean {
  return activeClientDomainId.value === domainId && activeClientId.value === clientId
}

// Check if a saved request is currently open in tabs
function isRequestOpen(requestId: string): boolean {
  return workspaceStore.openedTabs.value.some(t => t.savedRequestId === requestId)
}

// Request operations
function showRequestContextMenu(e: MouseEvent, domainId: string, clientId: string, req: SavedRequest) {
  e.stopPropagation()
  contextMenu.value = {
    show: true,
    x: e.clientX,
    y: e.clientY,
    type: 'request',
    domainId,
    clientId,
    requestId: req.id
  }
}

function openRequest(domainId: string, clientId: string, requestId: string) {
  workspaceStore.openSavedRequest(requestId, clientId, domainId)
  hideContextMenu()
}

async function addNewRequest(domainId: string, clientId: string) {
  try {
    // Create a new saved request and immediately open it
    const savedRequest = await workspaceStore.addSavedRequest(clientId, domainId, 'New Request')
    workspaceStore.openSavedRequest(savedRequest.id, clientId, domainId)
  } catch (err) {
    console.error('Failed to add request:', err)
    alert('Failed to add request: ' + (err as Error).message)
  }
  hideContextMenu()
}

// Delete confirmation dialog state
const deleteConfirmDialog = ref<{
  show: boolean
  domainId: string
  clientId: string
  requestId: string
}>({
  show: false,
  domainId: '',
  clientId: '',
  requestId: ''
})

function deleteRequest(domainId: string, clientId: string, requestId: string) {
  console.log('deleteRequest called:', { domainId, clientId, requestId })
  hideContextMenu()
  
  // 显示自定义确认对话框
  deleteConfirmDialog.value = {
    show: true,
    domainId,
    clientId,
    requestId
  }
}

function cancelDeleteRequest() {
  deleteConfirmDialog.value.show = false
}

async function confirmDeleteRequest() {
  const { domainId, clientId, requestId } = deleteConfirmDialog.value
  deleteConfirmDialog.value.show = false
  
  console.log('Starting deletion process...')
  
  try {
    // 直接更新本地状态（不依赖后端）
    if (workspaceStore.workspace.value) {
      const domainIndex = workspaceStore.workspace.value.clientDomains.findIndex(d => d.id === domainId)
      console.log('Found domain at index:', domainIndex)
      
      if (domainIndex !== -1) {
        const clientIndex = workspaceStore.workspace.value.clientDomains[domainIndex].clients.findIndex(c => c.id === clientId)
        console.log('Found client at index:', clientIndex)
        
        if (clientIndex !== -1) {
          const client = workspaceStore.workspace.value.clientDomains[domainIndex].clients[clientIndex]
          const beforeCount = client.savedRequests?.length ?? 0
          console.log('Before deletion, savedRequests count:', beforeCount)
          
          if (client.savedRequests) {
            // 使用数组替换来触发 Vue 响应式
            const newRequests = client.savedRequests.filter(r => r.id !== requestId)
            workspaceStore.workspace.value.clientDomains[domainIndex].clients[clientIndex].savedRequests = newRequests
            console.log('After deletion, savedRequests count:', newRequests.length)
          }
        }
      }
    }
    
    // 关闭关联的打开 Tab
    const openTab = workspaceStore.openedTabs.value.find(t => t.savedRequestId === requestId)
    if (openTab) {
      console.log('Closing associated tab:', openTab.id)
      workspaceStore.closeTab(openTab.id)
    }
    
    // 保存到后端
    console.log('Saving to backend...')
    await workspaceStore.saveNow()
    console.log('Deletion completed successfully')
    
  } catch (err) {
    console.error('Failed to delete request:', err)
  }
}

function renameRequest(domainId: string, clientId: string, requestId: string, newName: string) {
  // 使用专门的重命名函数，会同步更新打开的 Tab
  workspaceStore.renameSavedRequest(requestId, clientId, domainId, newName)
}

// Format request label with method
function formatRequestLabel(req: SavedRequest): string {
  const method = req.request.method
  const path = req.request.path || req.name
  return `${method} ${path || req.name}`
}
</script>

<template>
  <div class="client-domain-tree" @click="handleGlobalClick" @contextmenu.prevent="showEmptyContextMenu">
    <!-- Client domains -->
    <TreeNode
      v-for="domain in clientDomains"
      :key="domain.id"
      :label="domain.name"
      icon="📁"
      :expandable="true"
      :expanded="viewStore.isNodeExpanded(`client-domain-${domain.id}`)"
      :selected="isDomainActive(domain.id)"
      :editable="true"
      @select="selectDomain(domain.id)"
      @contextmenu="showDomainContextMenu($event, domain)"
      @rename="(name) => workspaceStore.updateClientDomain({ ...domain, name })"
      @update:expanded="(val) => viewStore.setNodeExpanded(`client-domain-${domain.id}`, val)"
    >
      <!-- Clients in this domain -->
      <TreeNode
        v-for="client in domain.clients"
        :key="client.id"
        :label="client.name"
        icon="👤"
        :level="1"
        :expandable="true"
        :expanded="viewStore.isNodeExpanded(`client-${client.id}`)"
        :selected="isClientActive(domain.id, client.id)"
        :editable="true"
        @select="selectClient(domain.id, client.id)"
        @contextmenu="showClientContextMenu($event, domain.id, client)"
        @rename="(name) => workspaceStore.updateClient({ ...client, name }, domain.id)"
        @update:expanded="(val) => viewStore.setNodeExpanded(`client-${client.id}`, val)"
      >
        <!-- Saved Requests in this client -->
        <TreeNode
          v-for="req in (client.savedRequests || [])"
          :key="req.id"
          :label="req.name"
          :method="req.request.method"
          :level="2"
          :selected="isRequestOpen(req.id)"
          :editable="true"
          @select="openRequest(domain.id, client.id, req.id)"
          @contextmenu="showRequestContextMenu($event, domain.id, client.id, req)"
          @rename="(name) => renameRequest(domain.id, client.id, req.id, name)"
        />

        <!-- Add request button -->
        <div class="add-item nested" @click.stop="addNewRequest(domain.id, client.id)">
          <span class="add-icon">+</span>
          <span class="add-label">Add Request</span>
        </div>
      </TreeNode>

      <!-- Add client button -->
      <div class="add-item" @click.stop="openAddClientDialog(domain.id)">
        <span class="add-icon">+</span>
        <span class="add-label">Add Client</span>
      </div>
    </TreeNode>

    <!-- Add domain button -->
    <div class="add-item root" @click.stop="openAddDomainDialog">
      <span class="add-icon">+</span>
      <span class="add-label">Add Workspace</span>
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
          <div class="menu-item" @click="openAddClientDialog(contextMenu.domainId!)">
            Add Client
          </div>
          <div class="menu-item" @click="openRenameDomainDialog(contextMenu.domainId!)">
            Rename
          </div>
          <div class="menu-divider"></div>
          <div class="menu-item danger" @click="deleteDomain(contextMenu.domainId!)">
            Delete
          </div>
        </template>
        <template v-else-if="contextMenu.type === 'client'">
          <div class="menu-item" @click="addNewRequest(contextMenu.domainId!, contextMenu.clientId!)">
            Add Request
          </div>
          <div class="menu-item" @click="openRenameClientDialog(contextMenu.domainId!, contextMenu.clientId!)">
            Rename
          </div>
          <div class="menu-divider"></div>
          <div class="menu-item danger" @click="deleteClient(contextMenu.domainId!, contextMenu.clientId!)">
            Delete
          </div>
        </template>
        <template v-else-if="contextMenu.type === 'request'">
          <div class="menu-item" @click="openRequest(contextMenu.domainId!, contextMenu.clientId!, contextMenu.requestId!)">
            Open
          </div>
          <div class="menu-divider"></div>
          <div class="menu-item danger" @click="deleteRequest(contextMenu.domainId!, contextMenu.clientId!, contextMenu.requestId!)">
            Delete
          </div>
        </template>
        <template v-else>
          <div class="menu-item" @click="openAddDomainDialog">
            Add Workspace
          </div>
        </template>
      </div>
    </Teleport>

    <!-- Delete Confirmation Dialog -->
    <Teleport to="body">
      <div v-if="deleteConfirmDialog.show" class="dialog-overlay" @click="cancelDeleteRequest">
        <div class="dialog" @click.stop>
          <div class="dialog-header">Delete Request</div>
          <div class="dialog-body">
            <p>Are you sure you want to delete this request?</p>
          </div>
          <div class="dialog-footer">
            <button class="btn btn-secondary" @click="cancelDeleteRequest">Cancel</button>
            <button class="btn btn-danger" @click="confirmDeleteRequest">Delete</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Edit Dialog -->
    <Teleport to="body">
      <div v-if="editDialog.show" class="dialog-overlay" @click="cancelDialog">
        <div class="dialog" @click.stop>
          <div class="dialog-header">
            {{ editDialog.mode === 'addDomain' ? 'Add Workspace' :
               editDialog.mode === 'renameDomain' ? 'Rename Workspace' :
               editDialog.mode === 'addClient' ? 'Add Client' : 'Rename Client' }}
          </div>
          <div class="dialog-body">
            <div class="form-group">
              <label>Name</label>
              <input 
                v-model="editDialog.name" 
                type="text" 
                placeholder="Name"
                @keydown.enter="saveDialog"
              />
            </div>
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
.client-domain-tree {
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

.add-item.nested {
  padding-left: 74px;
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

.menu-divider {
  height: 1px;
  background: #444;
  margin: 4px 0;
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

.btn-danger {
  background: #d32f2f;
  color: white;
}

.btn-danger:hover {
  background: #e53935;
}
</style>