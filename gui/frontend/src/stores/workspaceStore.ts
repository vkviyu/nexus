import { ref, computed } from "vue";
import type {
  Workspace,
  ServerDomain,
  ClientDomain,
  Server,
  Client,
  SavedRequest,
  OpenedTab,
  RequestTab,
  ServerStatus,
} from "../types/domain";
import {
  newServerDomain,
  newServer,
  newClientDomain,
  newClient,
  newSavedRequest,
  newOpenedTab,
  openedTabFromSavedRequest,
  newWorkspace,
  newMockBehavior,
  generateId,
} from "../types/domain";

// ============================================================================
// State
// ============================================================================

const workspace = ref<Workspace | null>(null);
const activeClientDomainId = ref<string>("");
const activeClientId = ref<string>("");
const activeTabId = ref<string>("");
const initialized = ref(false);
const loading = ref(false);

// Server runtime statuses (not persisted)
const serverStatuses = ref<Record<string, ServerStatus>>({});

// Server actual ports (for port 0 auto-assignment)
const serverActualPorts = ref<Record<string, number>>({});

// Server last error messages
const serverErrors = ref<Record<string, string>>({});

// ============================================================================
// Opened Tabs State (runtime only, not persisted)
// ============================================================================

const openedTabs = ref<OpenedTab[]>([]);

// Debounce timer for auto-save
let saveTimeout: ReturnType<typeof setTimeout> | null = null;
const SAVE_DEBOUNCE_MS = 500;

// ============================================================================
// Computed - Server Domains
// ============================================================================

const serverDomains = computed(() => workspace.value?.serverDomains ?? []);

const allServers = computed(() => {
  const servers: Array<Server & { domainId: string; domainName: string }> = [];
  for (const domain of serverDomains.value) {
    for (const server of domain.servers) {
      servers.push({ ...server, domainId: domain.id, domainName: domain.name });
    }
  }
  return servers;
});

// ============================================================================
// Computed - Client Domains
// ============================================================================

const clientDomains = computed(() => workspace.value?.clientDomains ?? []);

const activeClientDomain = computed(() => {
  if (!workspace.value || !activeClientDomainId.value) return null;
  return (
    workspace.value.clientDomains.find(
      (d) => d.id === activeClientDomainId.value,
    ) ?? null
  );
});

const clients = computed(() => activeClientDomain.value?.clients ?? []);

const activeClient = computed(() => {
  if (!activeClientDomain.value || !activeClientId.value) return null;
  return (
    activeClientDomain.value.clients.find(
      (c) => c.id === activeClientId.value,
    ) ?? null
  );
});

// Opened tabs (runtime state, shown in top tab bar)
const activeTabs = computed(() => openedTabs.value);

// Currently active tab
const activeTab = computed(() => {
  if (!activeTabId.value) return null;
  return openedTabs.value.find((t) => t.id === activeTabId.value) ?? null;
});

// Get all saved requests for active client
const activeSavedRequests = computed(
  () => activeClient.value?.savedRequests ?? [],
);

// ============================================================================
// Backend API Helpers
// ============================================================================

async function loadWorkspaceFromBackend(): Promise<Workspace | null> {
  try {
    const { LoadWorkspace } = await import("../../wailsjs/go/main/App");
    const result = await LoadWorkspace();
    return result as unknown as Workspace;
  } catch (err) {
    console.error("Failed to load workspace:", err);
    return null;
  }
}

async function saveWorkspaceToBackend(ws: Workspace): Promise<boolean> {
  try {
    const { SaveWorkspace } = await import("../../wailsjs/go/main/App");
    await SaveWorkspace(ws as any);
    return true;
  } catch (err) {
    console.error("Failed to save workspace:", err);
    return false;
  }
}

// ============================================================================
// Core Functions
// ============================================================================

/**
 * Initialize the store by loading data from backend.
 */
async function init(): Promise<void> {
  if (initialized.value) return;
  loading.value = true;

  const ws = await loadWorkspaceFromBackend();
  if (ws) {
    workspace.value = ws;

    // Restore active state from workspace or set defaults
    if (ws.activeClientDomainId) {
      activeClientDomainId.value = ws.activeClientDomainId;
    } else if (ws.clientDomains.length > 0) {
      activeClientDomainId.value = ws.clientDomains[0].id;
    }

    if (ws.activeClientId) {
      activeClientId.value = ws.activeClientId;
    } else {
      const domain = ws.clientDomains.find(
        (d) => d.id === activeClientDomainId.value,
      );
      if (domain && domain.clients.length > 0) {
        activeClientId.value = domain.clients[0].id;
      }
    }

    if (ws.activeTabId) {
      activeTabId.value = ws.activeTabId;
    } else {
      const domain = ws.clientDomains.find(
        (d) => d.id === activeClientDomainId.value,
      );
      const client = domain?.clients.find((c) => c.id === activeClientId.value);
      if (client && client.tabs.length > 0) {
        activeTabId.value = client.tabs[0].id;
      }
    }

    initialized.value = true;
  }

  // Listen for server status change events from backend
  try {
    const { EventsOn } = await import("../../wailsjs/runtime/runtime");
    EventsOn(
      "server:status",
      (
        serverId: string,
        status: string,
        actualPort: number,
        errMsg: string,
      ) => {
        console.log(
          "[workspaceStore] server:status event received:",
          serverId,
          status,
          "actualPort:",
          actualPort,
          "error:",
          errMsg,
        );
        serverStatuses.value[serverId] = status as ServerStatus;
        if (actualPort > 0) {
          serverActualPorts.value[serverId] = actualPort;
        } else {
          delete serverActualPorts.value[serverId];
        }
        if (errMsg) {
          serverErrors.value[serverId] = errMsg;
        } else {
          delete serverErrors.value[serverId];
        }
      },
    );
  } catch (err) {
    console.error("Failed to setup server status listener:", err);
  }

  loading.value = false;
}

/**
 * Save current state to backend (debounced).
 */
function save(): void {
  if (!workspace.value) return;

  if (saveTimeout) {
    clearTimeout(saveTimeout);
  }

  saveTimeout = setTimeout(async () => {
    if (!workspace.value) return;

    // Update active state in workspace before saving
    workspace.value.activeClientDomainId = activeClientDomainId.value;
    workspace.value.activeClientId = activeClientId.value;
    workspace.value.activeTabId = activeTabId.value;

    await saveWorkspaceToBackend(workspace.value);
  }, SAVE_DEBOUNCE_MS);
}

/**
 * Force immediate save without debounce.
 */
async function saveNow(): Promise<boolean> {
  if (!workspace.value) return false;

  if (saveTimeout) {
    clearTimeout(saveTimeout);
    saveTimeout = null;
  }

  workspace.value.activeClientDomainId = activeClientDomainId.value;
  workspace.value.activeClientId = activeClientId.value;
  workspace.value.activeTabId = activeTabId.value;

  return await saveWorkspaceToBackend(workspace.value);
}

// ============================================================================
// Server Domain CRUD
// ============================================================================

function addServerDomain(name?: string): ServerDomain {
  if (!workspace.value) throw new Error("Workspace not initialized");

  const domain = newServerDomain(name);
  workspace.value.serverDomains.push(domain);
  save();
  return domain;
}

function updateServerDomain(domain: ServerDomain): void {
  if (!workspace.value) return;

  const index = workspace.value.serverDomains.findIndex(
    (d) => d.id === domain.id,
  );
  if (index !== -1) {
    workspace.value.serverDomains[index] = { ...domain };
    save();
  }
}

function deleteServerDomain(domainId: string): void {
  if (!workspace.value) return;

  const index = workspace.value.serverDomains.findIndex(
    (d) => d.id === domainId,
  );
  if (index !== -1) {
    workspace.value.serverDomains.splice(index, 1);
    save();
  }
}

function getServerDomain(domainId: string): ServerDomain | undefined {
  return workspace.value?.serverDomains.find((d) => d.id === domainId);
}

// ============================================================================
// Server CRUD (within a domain)
// ============================================================================

function addServer(domainId: string, server?: Partial<Server>): Server {
  if (!workspace.value) throw new Error("Workspace not initialized");

  const domain = workspace.value.serverDomains.find((d) => d.id === domainId);
  if (!domain) throw new Error("Server domain not found");

  const srv: Server = {
    id: generateId(),
    name: server?.name ?? "New Server",
    port: server?.port ?? 8080,
    description: server?.description ?? "",
    behaviors: server?.behaviors ?? [
      newMockBehavior(200, {
        message: `Hello from ${server?.name ?? "New Server"}`,
      }),
    ],
  };

  domain.servers.push(srv);
  save();
  return srv;
}

function updateServer(domainId: string, server: Server): void {
  if (!workspace.value) return;

  const domain = workspace.value.serverDomains.find((d) => d.id === domainId);
  if (!domain) return;

  const index = domain.servers.findIndex((s) => s.id === server.id);
  if (index !== -1) {
    domain.servers[index] = { ...server };
    save();
  }
}

async function deleteServer(domainId: string, serverId: string): Promise<void> {
  if (!workspace.value) return;

  try {
    // Call backend API
    const { DeleteServer } = await import("../../wailsjs/go/main/App");
    await DeleteServer(domainId, serverId);

    // Update local state after successful backend deletion
    const domainIndex = workspace.value.serverDomains.findIndex(
      (d) => d.id === domainId,
    );
    if (domainIndex !== -1) {
      const domain = workspace.value.serverDomains[domainIndex];
      workspace.value.serverDomains[domainIndex].servers =
        domain.servers.filter((s) => s.id !== serverId);
    }
  } catch (err) {
    console.error("Failed to delete server:", err);
    throw err;
  }
}

function getServer(domainId: string, serverId: string): Server | undefined {
  const domain = workspace.value?.serverDomains.find((d) => d.id === domainId);
  return domain?.servers.find((s) => s.id === serverId);
}

/**
 * Find a server across all domains by server ID only.
 */
function findServerById(
  serverId: string,
): { server: Server; domainId: string } | undefined {
  if (!workspace.value) return undefined;

  for (const domain of workspace.value.serverDomains) {
    const server = domain.servers.find((s) => s.id === serverId);
    if (server) {
      return { server, domainId: domain.id };
    }
  }
  return undefined;
}

// ============================================================================
// Client Domain CRUD
// ============================================================================

function addClientDomain(name?: string): ClientDomain {
  if (!workspace.value) throw new Error("Workspace not initialized");

  const domain = newClientDomain(name);
  workspace.value.clientDomains.push(domain);
  save();
  return domain;
}

function updateClientDomain(domain: ClientDomain): void {
  if (!workspace.value) return;

  const index = workspace.value.clientDomains.findIndex(
    (d) => d.id === domain.id,
  );
  if (index !== -1) {
    workspace.value.clientDomains[index] = { ...domain };
    save();
  }
}

function deleteClientDomain(domainId: string): void {
  if (!workspace.value) return;

  const index = workspace.value.clientDomains.findIndex(
    (d) => d.id === domainId,
  );
  if (index !== -1) {
    workspace.value.clientDomains.splice(index, 1);

    // Update active domain if deleted
    if (activeClientDomainId.value === domainId) {
      const remaining = workspace.value.clientDomains;
      if (remaining.length > 0) {
        setActiveClientDomain(remaining[0].id);
      } else {
        activeClientDomainId.value = "";
        activeClientId.value = "";
        activeTabId.value = "";
      }
    }

    save();
  }
}

function getClientDomain(domainId: string): ClientDomain | undefined {
  return workspace.value?.clientDomains.find((d) => d.id === domainId);
}

function setActiveClientDomain(domainId: string): void {
  activeClientDomainId.value = domainId;

  // Set first client as active
  const domain = workspace.value?.clientDomains.find((d) => d.id === domainId);
  if (domain && domain.clients.length > 0) {
    activeClientId.value = domain.clients[0].id;
    if (domain.clients[0].tabs.length > 0) {
      activeTabId.value = domain.clients[0].tabs[0].id;
    } else {
      activeTabId.value = "";
    }
  } else {
    activeClientId.value = "";
    activeTabId.value = "";
  }

  save();
}

// ============================================================================
// Client CRUD (within a domain)
// ============================================================================

function addClient(domainId?: string, client?: Partial<Client>): Client {
  const targetDomainId = domainId ?? activeClientDomainId.value;
  if (!workspace.value || !targetDomainId)
    throw new Error("No client domain selected");

  const domain = workspace.value.clientDomains.find(
    (d) => d.id === targetDomainId,
  );
  if (!domain) throw new Error("Client domain not found");

  const cli = newClient(client?.name);
  if (client?.tabs) {
    cli.tabs = client.tabs;
  }

  domain.clients.push(cli);
  save();
  return cli;
}

function updateClient(client: Client, domainId?: string): void {
  const targetDomainId = domainId ?? activeClientDomainId.value;
  if (!workspace.value || !targetDomainId) return;

  const domain = workspace.value.clientDomains.find(
    (d) => d.id === targetDomainId,
  );
  if (!domain) return;

  const index = domain.clients.findIndex((c) => c.id === client.id);
  if (index !== -1) {
    domain.clients[index] = { ...client };
    save();
  }
}

function deleteClient(clientId: string, domainId?: string): void {
  const targetDomainId = domainId ?? activeClientDomainId.value;
  if (!workspace.value || !targetDomainId) return;

  const domain = workspace.value.clientDomains.find(
    (d) => d.id === targetDomainId,
  );
  if (!domain) return;

  const index = domain.clients.findIndex((c) => c.id === clientId);
  if (index !== -1) {
    domain.clients.splice(index, 1);

    // Update active client if deleted
    if (activeClientId.value === clientId) {
      if (domain.clients.length > 0) {
        setActiveClient(domain.clients[0].id);
      } else {
        activeClientId.value = "";
        activeTabId.value = "";
      }
    }

    save();
  }
}

function setActiveClient(clientId: string): void {
  activeClientId.value = clientId;

  // Set first tab of new client as active
  const domain = workspace.value?.clientDomains.find(
    (d) => d.id === activeClientDomainId.value,
  );
  const client = domain?.clients.find((c) => c.id === clientId);
  if (client && client.tabs.length > 0) {
    activeTabId.value = client.tabs[0].id;
  } else {
    activeTabId.value = "";
  }

  save();
}

// ============================================================================
// Opened Tab Management (Runtime State)
// ============================================================================

/**
 * Create a new unsaved tab and add to opened tabs.
 */
function addNewTab(): OpenedTab {
  const tab = newOpenedTab();
  openedTabs.value.push(tab);
  activeTabId.value = tab.id;
  return tab;
}

/**
 * Open a saved request in a new tab (or switch to existing tab).
 */
function openSavedRequest(
  savedRequestId: string,
  clientId: string,
  domainId: string,
): OpenedTab {
  // Check if already open
  const existingTab = openedTabs.value.find(
    (t) => t.savedRequestId === savedRequestId,
  );
  if (existingTab) {
    activeTabId.value = existingTab.id;
    return existingTab;
  }

  // Find the saved request
  const domain = workspace.value?.clientDomains.find((d) => d.id === domainId);
  const client = domain?.clients.find((c) => c.id === clientId);
  const savedRequest = client?.savedRequests?.find(
    (r) => r.id === savedRequestId,
  );

  if (!savedRequest) {
    throw new Error("Saved request not found");
  }

  // Create new opened tab from saved request
  const tab = openedTabFromSavedRequest(savedRequest, clientId, domainId);
  openedTabs.value.push(tab);
  activeTabId.value = tab.id;
  return tab;
}

/**
 * Update an opened tab and mark as dirty.
 * - For tabs linked to saved requests: marks dirty for unsaved changes
 * - For new unsaved tabs: also marks dirty to prompt save on close
 */
function updateTab(tab: OpenedTab): void {
  const index = openedTabs.value.findIndex((t) => t.id === tab.id);
  if (index !== -1) {
    // Always mark as dirty on update - this covers both:
    // 1. Tabs with savedRequestId (changes to saved requests)
    // 2. New tabs without savedRequestId (unsaved new requests)
    tab.isDirty = true;
    openedTabs.value[index] = { ...tab };
  }
}

/**
 * Close a tab. Returns true if closed, false if cancelled.
 * If tab has unsaved changes, caller should handle the prompt.
 */
function closeTab(tabId: string): { closed: boolean; tab: OpenedTab | null } {
  const index = openedTabs.value.findIndex((t) => t.id === tabId);
  if (index === -1) return { closed: false, tab: null };

  const tab = openedTabs.value[index];

  // Remove from opened tabs
  openedTabs.value.splice(index, 1);

  // Update active tab if closed
  if (activeTabId.value === tabId) {
    if (openedTabs.value.length > 0) {
      activeTabId.value = openedTabs.value[Math.max(0, index - 1)].id;
    } else {
      activeTabId.value = "";
    }
  }

  return { closed: true, tab };
}

/**
 * Set active tab by ID.
 */
function setActiveTab(tabId: string): void {
  activeTabId.value = tabId;
}

/**
 * Check if a tab has unsaved changes.
 */
function isTabDirty(tabId: string): boolean {
  const tab = openedTabs.value.find((t) => t.id === tabId);
  return tab?.isDirty ?? false;
}

/**
 * Rename a tab and sync with saved request (does NOT trigger dirty flag).
 * This is for renaming only - the name is immediately persisted.
 */
function renameTab(tabId: string, newName: string): void {
  const tabIndex = openedTabs.value.findIndex((t) => t.id === tabId);
  if (tabIndex === -1) return;

  const tab = openedTabs.value[tabIndex];

  // Update the tab name (without marking as dirty)
  openedTabs.value[tabIndex] = { ...tab, name: newName };

  // If linked to a saved request, also update that
  if (tab.savedRequestId && tab.clientId && tab.domainId && workspace.value) {
    const domainIndex = workspace.value.clientDomains.findIndex(
      (d) => d.id === tab.domainId,
    );
    if (domainIndex !== -1) {
      const clientIndex = workspace.value.clientDomains[
        domainIndex
      ].clients.findIndex((c) => c.id === tab.clientId);
      if (clientIndex !== -1) {
        const client =
          workspace.value.clientDomains[domainIndex].clients[clientIndex];
        const savedIndex =
          client.savedRequests?.findIndex((r) => r.id === tab.savedRequestId) ??
          -1;
        if (savedIndex !== -1 && client.savedRequests) {
          // Update saved request name
          const updatedRequest = {
            ...client.savedRequests[savedIndex],
            name: newName,
          };
          const newSavedRequests = [...client.savedRequests];
          newSavedRequests[savedIndex] = updatedRequest;
          workspace.value.clientDomains[domainIndex].clients[
            clientIndex
          ].savedRequests = newSavedRequests;
        }
      }
    }
    // Persist immediately
    save();
  }
}

/**
 * Rename a saved request and sync with any open tabs.
 */
function renameSavedRequest(
  savedRequestId: string,
  clientId: string,
  domainId: string,
  newName: string,
): void {
  if (!workspace.value) return;

  // Update the saved request
  const domainIndex = workspace.value.clientDomains.findIndex(
    (d) => d.id === domainId,
  );
  if (domainIndex === -1) return;

  const clientIndex = workspace.value.clientDomains[
    domainIndex
  ].clients.findIndex((c) => c.id === clientId);
  if (clientIndex === -1) return;

  const client =
    workspace.value.clientDomains[domainIndex].clients[clientIndex];
  const savedIndex =
    client.savedRequests?.findIndex((r) => r.id === savedRequestId) ?? -1;
  if (savedIndex === -1 || !client.savedRequests) return;

  // Update saved request
  const updatedRequest = { ...client.savedRequests[savedIndex], name: newName };
  const newSavedRequests = [...client.savedRequests];
  newSavedRequests[savedIndex] = updatedRequest;
  workspace.value.clientDomains[domainIndex].clients[
    clientIndex
  ].savedRequests = newSavedRequests;

  // Sync with any open tab linked to this saved request
  const tabIndex = openedTabs.value.findIndex(
    (t) => t.savedRequestId === savedRequestId,
  );
  if (tabIndex !== -1) {
    openedTabs.value[tabIndex] = {
      ...openedTabs.value[tabIndex],
      name: newName,
    };
  }

  // Persist
  save();
}

// ============================================================================
// Saved Request Management (Persisted)
// ============================================================================

/**
 * Save current tab to a client's savedRequests.
 * If tab is already linked, updates the saved request.
 * If tab is new, creates a new saved request.
 */
function saveTab(
  tabId: string,
  targetClientId?: string,
  targetDomainId?: string,
): SavedRequest | null {
  if (!workspace.value) return null;

  const tabIndex = openedTabs.value.findIndex((t) => t.id === tabId);
  if (tabIndex === -1) return null;

  const tab = openedTabs.value[tabIndex];

  const domainId = targetDomainId ?? tab.domainId ?? activeClientDomainId.value;
  const clientId = targetClientId ?? tab.clientId ?? activeClientId.value;

  if (!domainId || !clientId) {
    throw new Error("No client selected for saving");
  }

  const domainIndex = workspace.value.clientDomains.findIndex(
    (d) => d.id === domainId,
  );
  if (domainIndex === -1) {
    throw new Error("Domain not found");
  }

  const clientIndex = workspace.value.clientDomains[
    domainIndex
  ].clients.findIndex((c) => c.id === clientId);
  if (clientIndex === -1) {
    throw new Error("Client not found");
  }

  const client =
    workspace.value.clientDomains[domainIndex].clients[clientIndex];
  const savedRequests = client.savedRequests || [];

  if (tab.savedRequestId) {
    // Update existing saved request
    const savedIndex = savedRequests.findIndex(
      (r) => r.id === tab.savedRequestId,
    );
    if (savedIndex !== -1) {
      const updatedRequest: SavedRequest = {
        id: tab.savedRequestId,
        name: tab.name,
        serverDomainId: tab.serverDomainId,
        serverId: tab.serverId,
        request: JSON.parse(JSON.stringify(tab.request)),
      };

      // 使用数组替换确保响应式更新
      const newSavedRequests = [...savedRequests];
      newSavedRequests[savedIndex] = updatedRequest;
      workspace.value.clientDomains[domainIndex].clients[
        clientIndex
      ].savedRequests = newSavedRequests;

      // 更新 tab 状态
      const updatedTab = { ...tab, isDirty: false };
      openedTabs.value[tabIndex] = updatedTab;

      save();
      return updatedRequest;
    }
  }

  // Create new saved request
  const savedRequest: SavedRequest = {
    id: generateId(),
    name: tab.name,
    serverDomainId: tab.serverDomainId,
    serverId: tab.serverId,
    request: JSON.parse(JSON.stringify(tab.request)),
  };

  // 使用数组替换确保响应式更新
  workspace.value.clientDomains[domainIndex].clients[
    clientIndex
  ].savedRequests = [...savedRequests, savedRequest];

  // Link tab to saved request - 使用新对象更新
  const updatedTab = {
    ...tab,
    savedRequestId: savedRequest.id,
    clientId: clientId,
    domainId: domainId,
    isDirty: false,
  };
  openedTabs.value[tabIndex] = updatedTab;

  save();
  return savedRequest;
}

/**
 * Create a new saved request in a client (calls backend API).
 */
async function addSavedRequest(
  clientId: string,
  domainId: string,
  name?: string,
): Promise<SavedRequest> {
  const savedRequest = newSavedRequest(name);

  try {
    // Call backend API
    const { AddSavedRequest } = await import("../../wailsjs/go/main/App");
    const result = await AddSavedRequest(
      domainId,
      clientId,
      savedRequest as any,
    );

    // Update savedRequest with backend-assigned ID if different
    if (result && result.id) {
      savedRequest.id = result.id;
    }

    // Update local state after successful backend addition
    if (workspace.value) {
      const domainIndex = workspace.value.clientDomains.findIndex(
        (d) => d.id === domainId,
      );
      if (domainIndex !== -1) {
        const clientIndex = workspace.value.clientDomains[
          domainIndex
        ].clients.findIndex((c) => c.id === clientId);
        if (clientIndex !== -1) {
          const client =
            workspace.value.clientDomains[domainIndex].clients[clientIndex];
          workspace.value.clientDomains[domainIndex].clients[
            clientIndex
          ].savedRequests = [...(client.savedRequests || []), savedRequest];
        }
      }
    }

    return savedRequest;
  } catch (err) {
    console.error("Failed to add saved request:", err);
    throw err;
  }
}

/**
 * Update a saved request.
 */
function updateSavedRequest(
  savedRequest: SavedRequest,
  clientId: string,
  domainId: string,
): void {
  const domain = workspace.value?.clientDomains.find((d) => d.id === domainId);
  const client = domain?.clients.find((c) => c.id === clientId);
  if (!client || !client.savedRequests) return;

  const index = client.savedRequests.findIndex((r) => r.id === savedRequest.id);
  if (index !== -1) {
    client.savedRequests[index] = { ...savedRequest };
    save();
  }
}

/**
 * Delete a saved request (calls backend API).
 */
async function deleteSavedRequest(
  savedRequestId: string,
  clientId: string,
  domainId: string,
): Promise<void> {
  try {
    // Call backend API
    const { DeleteSavedRequest } = await import("../../wailsjs/go/main/App");
    await DeleteSavedRequest(domainId, clientId, savedRequestId);

    // Update local state after successful backend deletion
    if (workspace.value) {
      const domainIndex = workspace.value.clientDomains.findIndex(
        (d) => d.id === domainId,
      );
      if (domainIndex !== -1) {
        const clientIndex = workspace.value.clientDomains[
          domainIndex
        ].clients.findIndex((c) => c.id === clientId);
        if (clientIndex !== -1) {
          const client =
            workspace.value.clientDomains[domainIndex].clients[clientIndex];
          if (client.savedRequests) {
            workspace.value.clientDomains[domainIndex].clients[
              clientIndex
            ].savedRequests = client.savedRequests.filter(
              (r) => r.id !== savedRequestId,
            );
          }
        }
      }
    }

    // Unlink any open tabs linked to this saved request
    const tabIndex = openedTabs.value.findIndex(
      (t) => t.savedRequestId === savedRequestId,
    );
    if (tabIndex !== -1) {
      const updatedTab = { ...openedTabs.value[tabIndex] };
      updatedTab.savedRequestId = undefined;
      updatedTab.clientId = undefined;
      updatedTab.domainId = undefined;
      openedTabs.value[tabIndex] = updatedTab;
    }
  } catch (err) {
    console.error("Failed to delete saved request:", err);
    throw err;
  }
}

/**
 * Get a saved request by ID.
 */
function getSavedRequest(
  savedRequestId: string,
  clientId: string,
  domainId: string,
): SavedRequest | undefined {
  const domain = workspace.value?.clientDomains.find((d) => d.id === domainId);
  const client = domain?.clients.find((c) => c.id === clientId);
  return client?.savedRequests?.find((r) => r.id === savedRequestId);
}

// ============================================================================
// Legacy Tab Functions (for backward compatibility during migration)
// ============================================================================

/**
 * @deprecated Use addNewTab() instead
 */
function addTab(clientId?: string, domainId?: string): OpenedTab {
  return addNewTab();
}

/**
 * @deprecated Use closeTab() instead
 */
function deleteTab(tabId: string, clientId?: string, domainId?: string): void {
  closeTab(tabId);
}

// ============================================================================
// URL Resolution (Cross-Domain)
// ============================================================================

/**
 * Get the full URL for a tab, resolving server binding across domains.
 * Supports port 0 auto-assignment by using actualPort from runtime.
 */
function getTabUrl(tab: RequestTab): string {
  if (!tab.serverDomainId || !tab.serverId) {
    // External URL mode - use path as full URL
    return tab.request.path;
  }

  const server = getServer(tab.serverDomainId, tab.serverId);
  if (!server) {
    // Server not found, fallback to path
    return tab.request.path;
  }

  // Use actual port if available (for port 0 auto-assignment), otherwise use configured port
  const actualPort = serverActualPorts.value[tab.serverId];
  const port = actualPort > 0 ? actualPort : server.port;

  const baseURL = `http://localhost:${port}`;
  const path = tab.request.path.replace(/^\//, "");
  return path ? `${baseURL}/${path}` : baseURL;
}

// ============================================================================
// Server Runtime Management
// ============================================================================

/**
 * Get the runtime status of a server.
 */
function getServerStatus(serverId: string): ServerStatus {
  return serverStatuses.value[serverId] ?? "stopped";
}

/**
 * Get the last error message for a server.
 */
function getServerError(serverId: string): string {
  return serverErrors.value[serverId] ?? "";
}

/**
 * Start a local server.
 */
async function startServer(domainId: string, serverId: string): Promise<void> {
  try {
    const { StartServer } = await import("../../wailsjs/go/main/App");
    await StartServer(domainId, serverId);
    serverStatuses.value[serverId] = "running";
  } catch (err) {
    console.error("Failed to start server:", err);
    serverStatuses.value[serverId] = "error";
    throw err;
  }
}

/**
 * Stop a running local server.
 */
async function stopServer(domainId: string, serverId: string): Promise<void> {
  try {
    const { StopServer } = await import("../../wailsjs/go/main/App");
    await StopServer(domainId, serverId);
    serverStatuses.value[serverId] = "stopped";
  } catch (err) {
    console.error("Failed to stop server:", err);
    throw err;
  }
}

/**
 * Refresh server statuses from backend.
 */
async function refreshServerStatuses(): Promise<void> {
  try {
    const { GetServerStatuses } = await import("../../wailsjs/go/main/App");
    const statuses = await GetServerStatuses();
    for (const [id, status] of Object.entries(statuses)) {
      serverStatuses.value[id] = status as ServerStatus;
    }
  } catch (err) {
    console.error("Failed to refresh server statuses:", err);
  }
}

/**
 * Get display name for a server binding (e.g., "E-Commerce > dev").
 */
function getServerDisplayName(
  serverDomainId: string,
  serverId: string,
): string {
  const domain = getServerDomain(serverDomainId);
  const server = getServer(serverDomainId, serverId);
  if (!domain || !server) return "External URL";
  return `${domain.name} > ${server.name}`;
}

// ============================================================================
// Export
// ============================================================================

export const workspaceStore = {
  // State (reactive refs)
  workspace,
  activeClientDomainId,
  activeClientId,
  activeTabId,
  initialized,
  loading,
  serverStatuses,
  serverActualPorts,

  // Computed - Server Domains
  serverDomains,
  allServers,

  // Computed - Client Domains
  clientDomains,
  activeClientDomain,
  clients,
  activeClient,
  activeTabs,
  activeTab,

  // Core functions
  init,
  save,
  saveNow,

  // Server Domain CRUD
  addServerDomain,
  updateServerDomain,
  deleteServerDomain,
  getServerDomain,

  // Server CRUD
  addServer,
  updateServer,
  deleteServer,
  getServer,
  findServerById,

  // Client Domain CRUD
  addClientDomain,
  updateClientDomain,
  deleteClientDomain,
  getClientDomain,
  setActiveClientDomain,

  // Client CRUD
  addClient,
  updateClient,
  deleteClient,
  setActiveClient,

  // Opened Tab Management
  openedTabs,
  addNewTab,
  openSavedRequest,
  updateTab,
  closeTab,
  setActiveTab,
  isTabDirty,
  renameTab,
  renameSavedRequest,

  // Saved Request Management
  activeSavedRequests,
  saveTab,
  addSavedRequest,
  updateSavedRequest,
  deleteSavedRequest,
  getSavedRequest,

  // Legacy (for backward compatibility)
  addTab,
  deleteTab,

  // URL Resolution
  getTabUrl,
  getServerDisplayName,

  // Server Runtime Management
  getServerStatus,
  startServer,
  stopServer,
  refreshServerStatuses,
  serverErrors,
  getServerError,

  // Utilities
  generateId,
};

export default workspaceStore;
