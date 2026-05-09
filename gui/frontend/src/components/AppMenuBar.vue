<script setup lang="ts">
import { ref } from 'vue'
import { viewStore } from '../stores/viewStore'

// Menu state
const activeMenu = ref<string | null>(null)

function toggleMenu(menu: string) {
  if (activeMenu.value === menu) {
    activeMenu.value = null
  } else {
    activeMenu.value = menu
  }
}

function closeMenu() {
  activeMenu.value = null
}

// View menu actions
function toggleServerDomains() {
  viewStore.togglePanelVisible('serverDomains')
  closeMenu()
}

function toggleClientDomains() {
  viewStore.togglePanelVisible('clientDomains')
  closeMenu()
}

function toggleContractEditor() {
  viewStore.togglePanelVisible('contractEditor')
  closeMenu()
}

function collapseAll() {
  viewStore.collapseAll()
  closeMenu()
}

function expandAll() {
  viewStore.expandAll()
  closeMenu()
}

// Close menu when clicking outside
function handleGlobalClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('.menu-bar')) {
    closeMenu()
  }
}
</script>

<template>
  <div class="menu-bar" @click.stop>
    <!-- File Menu -->
    <div class="menu-item" @click="toggleMenu('file')">
      <span class="menu-label">File</span>
      <div v-if="activeMenu === 'file'" class="menu-dropdown">
        <div class="dropdown-item disabled">New Workspace</div>
        <div class="dropdown-item disabled">Open Workspace</div>
        <div class="dropdown-divider"></div>
        <div class="dropdown-item disabled">Import</div>
        <div class="dropdown-item disabled">Export</div>
      </div>
    </div>

    <!-- Edit Menu -->
    <div class="menu-item" @click="toggleMenu('edit')">
      <span class="menu-label">Edit</span>
      <div v-if="activeMenu === 'edit'" class="menu-dropdown">
        <div class="dropdown-item disabled">Undo</div>
        <div class="dropdown-item disabled">Redo</div>
        <div class="dropdown-divider"></div>
        <div class="dropdown-item disabled">Cut</div>
        <div class="dropdown-item disabled">Copy</div>
        <div class="dropdown-item disabled">Paste</div>
      </div>
    </div>

    <!-- View Menu -->
    <div class="menu-item" @click="toggleMenu('view')">
      <span class="menu-label">View</span>
      <div v-if="activeMenu === 'view'" class="menu-dropdown">
        <div class="dropdown-item checkbox" @click.stop="toggleServerDomains">
          <span class="checkbox-icon">{{ viewStore.isPanelVisible('serverDomains') ? '☑' : '☐' }}</span>
          <span>Server Domains</span>
        </div>
        <div class="dropdown-item checkbox" @click.stop="toggleClientDomains">
          <span class="checkbox-icon">{{ viewStore.isPanelVisible('clientDomains') ? '☑' : '☐' }}</span>
          <span>Client Domains</span>
        </div>
        <div class="dropdown-item checkbox" @click.stop="toggleContractEditor">
          <span class="checkbox-icon">{{ viewStore.isPanelVisible('contractEditor') ? '☑' : '☐' }}</span>
          <span>Contract Editor</span>
        </div>
        <div class="dropdown-divider"></div>
        <div class="dropdown-item" @click.stop="collapseAll">
          <span>Collapse All Panels</span>
        </div>
        <div class="dropdown-item" @click.stop="expandAll">
          <span>Expand All Panels</span>
        </div>
      </div>
    </div>

    <!-- Help Menu -->
    <div class="menu-item" @click="toggleMenu('help')">
      <span class="menu-label">Help</span>
      <div v-if="activeMenu === 'help'" class="menu-dropdown">
        <div class="dropdown-item disabled">Documentation</div>
        <div class="dropdown-item disabled">About Nexus</div>
      </div>
    </div>

    <!-- App Title (right side) -->
    <div class="app-title">Nexus</div>

    <!-- Global click handler overlay -->
    <div v-if="activeMenu" class="menu-overlay" @click="closeMenu"></div>
  </div>
</template>

<style scoped>
.menu-bar {
  display: flex;
  align-items: center;
  height: 28px;
  background: #1e1e1e;
  border-bottom: 1px solid #333;
  padding: 0 8px;
  font-size: 13px;
  user-select: none;
  position: relative;
  z-index: 1000;
}

.menu-item {
  position: relative;
  padding: 4px 10px;
  cursor: pointer;
  border-radius: 4px;
}

.menu-item:hover {
  background: rgba(255, 255, 255, 0.1);
}

.menu-label {
  color: #d4d4d4;
}

.menu-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  background: #2a2a2a;
  border: 1px solid #444;
  border-radius: 4px;
  min-width: 180px;
  padding: 4px 0;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
  z-index: 1001;
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  color: #e0e0e0;
  cursor: pointer;
}

.dropdown-item:hover:not(.disabled) {
  background: rgba(255, 255, 255, 0.1);
}

.dropdown-item.disabled {
  color: #666;
  cursor: not-allowed;
}

.dropdown-item.checkbox {
  padding-left: 8px;
}

.checkbox-icon {
  font-size: 14px;
  width: 18px;
  text-align: center;
}

.dropdown-divider {
  height: 1px;
  background: #444;
  margin: 4px 0;
}

.app-title {
  margin-left: auto;
  color: #888;
  font-weight: 500;
  font-size: 12px;
  letter-spacing: 0.5px;
}

.menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 999;
}
</style>