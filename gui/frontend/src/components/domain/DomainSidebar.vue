<script setup lang="ts">
import { computed } from 'vue'
import ServerDomainTree from './ServerDomainTree.vue'
import ClientDomainTree from './ClientDomainTree.vue'
import { viewStore } from '../../stores/viewStore'

// Use viewStore for panel state - directly access reactive state
const serverSectionVisible = computed(() => viewStore.viewState.value.panels.serverDomains.visible)
const serverSectionExpanded = computed(() => viewStore.viewState.value.panels.serverDomains.expanded)
const clientSectionVisible = computed(() => viewStore.viewState.value.panels.clientDomains.visible)
const clientSectionExpanded = computed(() => viewStore.viewState.value.panels.clientDomains.expanded)

function toggleServerExpanded() {
  viewStore.togglePanelExpanded('serverDomains')
}

function toggleClientExpanded() {
  viewStore.togglePanelExpanded('clientDomains')
}
</script>

<template>
  <div class="domain-sidebar">
    <!-- Server Domains Section -->
    <div v-if="serverSectionVisible" class="section">
      <div class="section-header" @click="toggleServerExpanded">
        <span class="section-icon" :class="{ expanded: serverSectionExpanded }">▶</span>
        <span class="section-title">SERVER DOMAINS</span>
      </div>
      <div v-if="serverSectionExpanded" class="section-content">
        <ServerDomainTree />
      </div>
    </div>

    <!-- Divider (only show if both sections visible) -->
    <div v-if="serverSectionVisible && clientSectionVisible" class="section-divider"></div>

    <!-- Client Domains Section -->
    <div v-if="clientSectionVisible" class="section">
      <div class="section-header" @click="toggleClientExpanded">
        <span class="section-icon" :class="{ expanded: clientSectionExpanded }">▶</span>
        <span class="section-title">CLIENT DOMAINS</span>
      </div>
      <div v-if="clientSectionExpanded" class="section-content">
        <ClientDomainTree />
      </div>
    </div>

  </div>
</template>

<style scoped>
.domain-sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1e1e1e;
  border-right: 1px solid #333;
  overflow: hidden;
}

.section {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.section:first-child {
  flex: 1;
}

.section:last-child {
  flex: 1;
}

.section-header {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  user-select: none;
  background: #252525;
  border-bottom: 1px solid #333;
}

.section-header:hover {
  background: #2a2a2a;
}

.section-icon {
  font-size: 10px;
  color: #888;
  margin-right: 8px;
  transition: transform 0.15s;
}

.section-icon.expanded {
  transform: rotate(90deg);
}

.section-title {
  font-size: 11px;
  font-weight: 600;
  color: #888;
  letter-spacing: 0.5px;
}

.section-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.section-divider {
  height: 1px;
  background: #333;
}

</style>