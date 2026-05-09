import { ref, watch } from 'vue'

// ============================================================================
// Types
// ============================================================================

export interface PanelState {
  visible: boolean
  expanded: boolean
}

export interface ViewState {
  panels: {
    serverDomains: PanelState
    clientDomains: PanelState
    contractEditor: PanelState
  }
  // 存储每个域/节点的展开状态，key 为 domain id
  expandedNodes: Record<string, boolean>
}

// ============================================================================
// Default State
// ============================================================================

const DEFAULT_VIEW_STATE: ViewState = {
  panels: {
    serverDomains: { visible: true, expanded: true },
    clientDomains: { visible: true, expanded: true },
    contractEditor: { visible: false, expanded: true }
  },
  expandedNodes: {}
}

// ============================================================================
// State
// ============================================================================

const STORAGE_KEY = 'nexus-view-state'

function loadFromStorage(): ViewState {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) {
      return { ...DEFAULT_VIEW_STATE, ...JSON.parse(stored) }
    }
  } catch (e) {
    console.warn('Failed to load view state from storage:', e)
  }
  return { ...DEFAULT_VIEW_STATE }
}

const viewState = ref<ViewState>(loadFromStorage())

// Auto-save to localStorage
watch(
  viewState,
  (state) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
    } catch (e) {
      console.warn('Failed to save view state:', e)
    }
  },
  { deep: true }
)

// ============================================================================
// Panel Actions
// ============================================================================

type PanelName = keyof ViewState['panels']

function togglePanelVisible(panel: PanelName): void {
  viewState.value.panels[panel].visible = !viewState.value.panels[panel].visible
}

function togglePanelExpanded(panel: PanelName): void {
  viewState.value.panels[panel].expanded = !viewState.value.panels[panel].expanded
}

function setPanelVisible(panel: PanelName, visible: boolean): void {
  viewState.value.panels[panel].visible = visible
}

function setPanelExpanded(panel: PanelName, expanded: boolean): void {
  viewState.value.panels[panel].expanded = expanded
}

function isPanelVisible(panel: PanelName): boolean {
  return viewState.value.panels[panel].visible
}

function isPanelExpanded(panel: PanelName): boolean {
  return viewState.value.panels[panel].expanded
}

// ============================================================================
// Bulk Actions
// ============================================================================

function collapseAll(): void {
  for (const panel of Object.keys(viewState.value.panels) as PanelName[]) {
    viewState.value.panels[panel].expanded = false
  }
}

function expandAll(): void {
  for (const panel of Object.keys(viewState.value.panels) as PanelName[]) {
    viewState.value.panels[panel].expanded = true
  }
}

function showAll(): void {
  for (const panel of Object.keys(viewState.value.panels) as PanelName[]) {
    viewState.value.panels[panel].visible = true
  }
}

function hideAll(): void {
  for (const panel of Object.keys(viewState.value.panels) as PanelName[]) {
    viewState.value.panels[panel].visible = false
  }
}

// ============================================================================
// Node Expand State (for tree nodes like domains)
// ============================================================================

function isNodeExpanded(nodeId: string, defaultValue: boolean = true): boolean {
  if (nodeId in viewState.value.expandedNodes) {
    return viewState.value.expandedNodes[nodeId]
  }
  return defaultValue
}

function setNodeExpanded(nodeId: string, expanded: boolean): void {
  viewState.value.expandedNodes[nodeId] = expanded
}

function toggleNodeExpanded(nodeId: string, defaultValue: boolean = true): void {
  const current = isNodeExpanded(nodeId, defaultValue)
  viewState.value.expandedNodes[nodeId] = !current
}

// ============================================================================
// Export
// ============================================================================

export const viewStore = {
  // State
  viewState,

  // Panel actions
  togglePanelVisible,
  togglePanelExpanded,
  setPanelVisible,
  setPanelExpanded,
  isPanelVisible,
  isPanelExpanded,

  // Bulk actions
  collapseAll,
  expandAll,
  showAll,
  hideAll,

  // Node expand state
  isNodeExpanded,
  setNodeExpanded,
  toggleNodeExpanded
}

export default viewStore