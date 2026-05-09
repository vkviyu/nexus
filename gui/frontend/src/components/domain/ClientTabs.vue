<template>
  <div class="client-tabs">
    <div class="tabs-container">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        :class="[
          'tab-item',
          { active: activeTabId === tab.id, dirty: tab.isDirty },
        ]"
        @click="selectTab(tab.id)"
        @dblclick="startEditName(tab)"
        @contextmenu.prevent="showContextMenu($event, tab)"
      >
        <span :class="['method-badge', `method-${tab.request.method.toLowerCase()}`]">
          {{ tab.request.method }}
        </span>
        <span v-if="editingTabId !== tab.id" class="tab-name">
          {{ tab.name }}<span class="dirty-indicator" v-if="tab.isDirty">*</span>
        </span>
        <input
          v-else
          ref="nameInput"
          v-model="editingName"
          class="tab-name-input"
          @blur="finishEditName(tab)"
          @keyup.enter="finishEditName(tab)"
          @keyup.escape="cancelEditName"
          @click.stop
        />
        <button
          @click.stop="handleCloseTab(tab)"
          class="close-btn"
          title="Close Tab"
        >
          ×
        </button>
      </div>
      <button @click="handleAddNewTab" class="add-tab-btn" title="New Tab">
        +
      </button>
    </div>

    <!-- Tab Context Menu -->
    <Teleport to="body">
      <div
        v-if="contextMenu.show"
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        @click.stop
      >
        <div class="menu-item" @click="saveCurrentTab">
          <span>Save</span>
          <span class="shortcut">⌘S</span>
        </div>
        <div class="menu-item" @click="saveAsNewRequest">Save to...</div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="closeFromContextMenu">Close</div>
      </div>
    </Teleport>

    <!-- Save Location Dialog (for new tabs) -->
    <Teleport to="body">
      <div
        v-if="saveDialog.show"
        class="dialog-overlay"
        @click="cancelSaveDialog"
      >
        <div class="dialog" @click.stop>
          <div class="dialog-header">Save Request</div>
          <div class="dialog-body">
            <div class="form-group">
              <label>Name</label>
              <input
                v-model="saveDialog.name"
                type="text"
                placeholder="Request name"
              />
            </div>
            <div class="form-group">
              <label>Save to Client</label>
              <select v-model="saveDialog.selectedClientKey">
                <option
                  v-for="opt in clientOptions"
                  :key="opt.key"
                  :value="opt.key"
                >
                  {{ opt.label }}
                </option>
              </select>
            </div>
          </div>
          <div class="dialog-footer">
            <div class="dialog-buttons">
              <button class="btn btn-secondary" @click="cancelSaveDialog">
                Cancel
              </button>
              <button class="btn btn-primary" @click="confirmSave">Save</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Close Confirmation Dialog -->
    <Teleport to="body">
      <div v-if="closeDialog.show" class="dialog-overlay" @click="cancelClose">
        <div class="dialog" @click.stop>
          <div class="dialog-header">Save Changes</div>
          <div class="dialog-body">
            <p class="dialog-message">
              Do you want to save the changes you made to "<strong>{{
                closeDialog.tabName
              }}</strong
              >"?
            </p>
            <p class="dialog-hint" v-if="closeDialog.saveTarget">
              Will be saved to: <strong>{{ closeDialog.saveTarget }}</strong>
            </p>
            <p class="dialog-warning" v-else>
              Your changes will be lost if you don't save them.
            </p>
            <div class="dialog-option">
              <label class="checkbox-label">
                <input type="checkbox" v-model="alwaysDiscard" />
                Always discard unsaved changes when closing a tab
              </label>
              <p class="option-hint">
                You'll no longer be prompted to save changes when closing a tab.
                You can change this anytime from your Settings.
              </p>
            </div>
          </div>
          <div class="dialog-footer">
            <div class="dialog-buttons">
              <button class="btn btn-secondary" @click="cancelClose">
                Cancel
              </button>
              <button class="btn btn-danger" @click="discardAndClose">
                Don't Save
              </button>
              <button class="btn btn-primary" @click="saveAndClose">
                Save Changes
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted } from "vue";
import { workspaceStore } from "../../stores/workspaceStore";
import type { OpenedTab } from "../../types/domain";

const tabs = computed(() => workspaceStore.activeTabs.value);
const activeTabId = computed(() => workspaceStore.activeTabId.value);

const editingTabId = ref<string | null>(null);
const editingName = ref("");
const nameInput = ref<HTMLInputElement[]>();

// Close dialog state
const closeDialog = ref<{
  show: boolean;
  tabId: string;
  tabName: string;
  saveTarget: string;
}>({
  show: false,
  tabId: "",
  tabName: "",
  saveTarget: "",
});

// Context menu state
const contextMenu = ref<{
  show: boolean;
  x: number;
  y: number;
  tabId: string;
}>({
  show: false,
  x: 0,
  y: 0,
  tabId: "",
});

// Save dialog state (for new tabs without savedRequestId)
const saveDialog = ref<{
  show: boolean;
  tabId: string;
  name: string;
  selectedClientKey: string;
  closeAfterSave: boolean;
}>({
  show: false,
  tabId: "",
  name: "",
  selectedClientKey: "",
  closeAfterSave: false,
});

// Client options for save dialog
const clientOptions = computed(() => {
  const options: Array<{
    key: string;
    label: string;
    domainId: string;
    clientId: string;
  }> = [];
  for (const domain of workspaceStore.clientDomains.value) {
    for (const client of domain.clients) {
      options.push({
        key: `${domain.id}:${client.id}`,
        label: `${domain.name} / ${client.name}`,
        domainId: domain.id,
        clientId: client.id,
      });
    }
  }
  return options;
});

// Settings - 强制读取当前值，并输出调试信息
const storedAlwaysDiscard = localStorage.getItem("nexus-always-discard");
console.log(
  "[ClientTabs] localStorage nexus-always-discard =",
  storedAlwaysDiscard,
);
const alwaysDiscard = ref(storedAlwaysDiscard === "true");
console.log("[ClientTabs] alwaysDiscard initialized to:", alwaysDiscard.value);

// Get save target description for a tab
function getSaveTarget(tab: OpenedTab): string {
  if (tab.savedRequestId && tab.clientId && tab.domainId) {
    const domain = workspaceStore.clientDomains.value.find(
      (d) => d.id === tab.domainId,
    );
    const client = domain?.clients.find((c) => c.id === tab.clientId);
    if (domain && client) {
      return `${domain.name} / ${client.name}`;
    }
  }
  return "";
}

function selectTab(tabId: string) {
  workspaceStore.setActiveTab(tabId);
  hideContextMenu();
}

function handleAddNewTab() {
  workspaceStore.addNewTab();
}

// Context menu functions
function showContextMenu(e: MouseEvent, tab: OpenedTab) {
  contextMenu.value = {
    show: true,
    x: e.clientX,
    y: e.clientY,
    tabId: tab.id,
  };
}

function hideContextMenu() {
  contextMenu.value.show = false;
}

function saveCurrentTab() {
  const tabId = contextMenu.value.tabId || workspaceStore.activeTabId.value;
  const tab = tabs.value.find((t) => t.id === tabId);
  hideContextMenu();

  if (!tab) return;

  // If tab has a linked savedRequest, save directly
  if (tab.savedRequestId && tab.clientId && tab.domainId) {
    workspaceStore.saveTab(tabId);
    return;
  }

  // Otherwise show save dialog to choose location
  openSaveDialog(tabId, false);
}

function saveAsNewRequest() {
  const tabId = contextMenu.value.tabId;
  hideContextMenu();
  openSaveDialog(tabId, false);
}

function closeFromContextMenu() {
  const tab = tabs.value.find((t) => t.id === contextMenu.value.tabId);
  hideContextMenu();
  if (tab) {
    handleCloseTab(tab);
  }
}

function openSaveDialog(tabId: string, closeAfterSave: boolean) {
  const tab = tabs.value.find((t) => t.id === tabId);
  if (!tab) return;

  // Default to active client
  const defaultKey =
    clientOptions.value.length > 0
      ? `${workspaceStore.activeClientDomainId.value}:${workspaceStore.activeClientId.value}`
      : "";

  saveDialog.value = {
    show: true,
    tabId: tabId,
    name: tab.name,
    selectedClientKey: defaultKey,
    closeAfterSave: closeAfterSave,
  };
}

function cancelSaveDialog() {
  saveDialog.value.show = false;
}

function confirmSave() {
  const { tabId, name, selectedClientKey, closeAfterSave } = saveDialog.value;

  if (!selectedClientKey) {
    alert("Please select a client to save to");
    return;
  }

  const [domainId, clientId] = selectedClientKey.split(":");
  const tab = tabs.value.find((t) => t.id === tabId);

  if (tab) {
    // Update tab name if changed
    if (name && name !== tab.name) {
      workspaceStore.updateTab({ ...tab, name });
    }

    // Save to selected client
    workspaceStore.saveTab(tabId, clientId, domainId);

    if (closeAfterSave) {
      workspaceStore.closeTab(tabId);
    }
  }

  saveDialog.value.show = false;
}

// Keyboard shortcuts
function handleKeyDown(e: KeyboardEvent) {
  // Cmd+S / Ctrl+S to save
  if ((e.metaKey || e.ctrlKey) && e.key === "s") {
    e.preventDefault();
    const tab = workspaceStore.activeTab.value;
    if (tab) {
      if (tab.savedRequestId && tab.clientId && tab.domainId) {
        workspaceStore.saveTab(tab.id);
      } else {
        openSaveDialog(tab.id, false);
      }
    }
  }
}

// Setup and cleanup
onMounted(() => {
  document.addEventListener("keydown", handleKeyDown);
  document.addEventListener("click", hideContextMenu);
});

onUnmounted(() => {
  document.removeEventListener("keydown", handleKeyDown);
  document.removeEventListener("click", hideContextMenu);
});

function handleCloseTab(tab: OpenedTab) {
  // Check if tab has unsaved changes and user hasn't opted to always discard
  if (tab.isDirty && !alwaysDiscard.value) {
    closeDialog.value = {
      show: true,
      tabId: tab.id,
      tabName: tab.name,
      saveTarget: getSaveTarget(tab),
    };
    return;
  }

  // Close directly
  workspaceStore.closeTab(tab.id);
}

function cancelClose() {
  closeDialog.value.show = false;
}

function discardAndClose() {
  // Save preference if checked
  if (alwaysDiscard.value) {
    localStorage.setItem("nexus-always-discard", "true");
  }

  workspaceStore.closeTab(closeDialog.value.tabId);
  closeDialog.value.show = false;
}

function saveAndClose() {
  const tabId = closeDialog.value.tabId;
  const tab = tabs.value.find((t) => t.id === tabId);

  // If tab doesn't have a saved location, show save dialog
  if (tab && !tab.savedRequestId) {
    closeDialog.value.show = false;
    openSaveDialog(tabId, true);
    return;
  }

  // Save the tab
  try {
    workspaceStore.saveTab(tabId);
    workspaceStore.closeTab(tabId);
  } catch (err) {
    alert("Failed to save: " + (err as Error).message);
    return;
  }

  closeDialog.value.show = false;
}

function startEditName(tab: OpenedTab) {
  editingTabId.value = tab.id;
  editingName.value = tab.name;
  nextTick(() => {
    if (nameInput.value && nameInput.value.length > 0) {
      nameInput.value[0].focus();
      nameInput.value[0].select();
    }
  });
}

function finishEditName(tab: OpenedTab) {
  if (editingTabId.value !== tab.id) return;

  const newName = editingName.value.trim();
  if (newName && newName !== tab.name) {
    // 使用 renameTab 而不是 updateTab，这样不会触发脏标记
    // 并且会同步更新 savedRequest
    workspaceStore.renameTab(tab.id, newName);
  }
  editingTabId.value = null;
  editingName.value = "";
}

function cancelEditName() {
  editingTabId.value = null;
  editingName.value = "";
}
</script>

<style scoped>
.client-tabs {
  background: #2d2d30;
  border-bottom: 1px solid #3c3c3c;
}

.tabs-container {
  display: flex;
  align-items: center;
  overflow-x: auto;
  padding: 0 4px;
}

.tabs-container::-webkit-scrollbar {
  height: 4px;
}

.tabs-container::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 2px;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: #969696;
  cursor: pointer;
  white-space: nowrap;
  min-width: 80px;
  max-width: 150px;
  transition:
    color 0.15s,
    border-color 0.15s;
}

.tab-item:hover {
  color: #d4d4d4;
}

.tab-item.active {
  color: #d4d4d4;
  border-bottom-color: #007acc;
}

.tab-name {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 13px;
}

.tab-name-input {
  width: 100%;
  padding: 2px 4px;
  background: #3c3c3c;
  border: 1px solid #007acc;
  border-radius: 2px;
  color: #d4d4d4;
  font-size: 13px;
  outline: none;
}

.close-btn {
  width: 18px;
  height: 18px;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: 3px;
  color: #858585;
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.15s;
}

.tab-item:hover .close-btn {
  opacity: 1;
}

.close-btn:hover {
  background: #5a1d1d;
  color: #f44336;
}

.add-tab-btn {
  width: 28px;
  height: 28px;
  margin: 4px;
  padding: 0;
  background: transparent;
  border: 1px dashed #555;
  border-radius: 4px;
  color: #858585;
  font-size: 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.add-tab-btn:hover {
  border-color: #007acc;
  color: #007acc;
}

/* Method badge - Postman style colors */
.method-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 2px 4px;
  border-radius: 3px;
  text-transform: uppercase;
  flex-shrink: 0;
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

/* Dirty indicator */
.dirty-indicator {
  color: #e0e0e0;
  font-size: 14px;
  margin-left: 2px;
  font-weight: bold;
}

.tab-item.dirty .tab-name {
  font-style: italic;
}

/* Dialog styles */
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
  width: 450px;
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

.dialog-body p {
  color: #d4d4d4;
  margin-bottom: 8px;
}

.dialog-body .dialog-hint {
  color: #888;
  font-size: 13px;
}

.dialog-body .dialog-message {
  font-size: 14px;
  line-height: 1.5;
}

.dialog-body .dialog-warning {
  color: #ffc107;
  font-size: 13px;
}

.dialog-option {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid #3c3c3c;
}

.dialog-option .checkbox-label {
  margin-bottom: 4px;
  color: #d4d4d4;
}

.dialog-option .option-hint {
  color: #888;
  font-size: 12px;
  margin: 0;
  padding-left: 24px;
  line-height: 1.4;
}

.dialog-footer {
  padding: 12px 16px;
  border-top: 1px solid #444;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #888;
  margin-bottom: 12px;
  cursor: pointer;
}

.checkbox-label input {
  cursor: pointer;
}

.dialog-buttons {
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
  background: #5a1d1d;
  color: #f44336;
}

.btn-danger:hover {
  background: #6a2d2d;
}

/* Context Menu */
.context-menu {
  position: fixed;
  background: #2a2a2a;
  border: 1px solid #444;
  border-radius: 4px;
  padding: 4px 0;
  min-width: 180px;
  z-index: 1002;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.menu-item {
  padding: 8px 16px;
  cursor: pointer;
  font-size: 13px;
  color: #e0e0e0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.menu-item:hover {
  background: rgba(255, 255, 255, 0.1);
}

.menu-item .shortcut {
  color: #888;
  font-size: 12px;
  margin-left: 16px;
}

.menu-divider {
  height: 1px;
  background: #444;
  margin: 4px 0;
}

/* Form Group */
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

.form-group input,
.form-group select {
  width: 100%;
  padding: 8px 12px;
  background: #1e1e1e;
  border: 1px solid #444;
  border-radius: 4px;
  color: #e0e0e0;
  font-size: 14px;
}

.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: #4285f4;
}

.form-group select {
  cursor: pointer;
}
</style>
