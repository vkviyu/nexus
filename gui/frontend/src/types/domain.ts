// Domain types - matching Go backend types
// Supports multiple server domains and client domains (multi-domain architecture)

// ============================================================================
// Basic Types
// ============================================================================

/**
 * Key-value pair for params or headers.
 */
export interface KVPair {
  key: string
  value: string
}

/**
 * HTTP response data.
 */
export interface ResponseData {
  status: string
  time: number
  size: number
  body: string
  headers: string
}

/**
 * HTTP request configuration.
 */
export interface RequestConfig {
  method: string
  path: string // Relative path when bound to server, or full URL for external (supports {param} placeholders)
  contentType: string
  pathParams: KVPair[] // Path parameters (e.g., {schoolId} -> schoolId: "3")
  params: KVPair[]     // Query parameters
  headers: KVPair[]
  body?: any
}

// ============================================================================
// Saved Request (persisted in Client)
// ============================================================================

/**
 * A saved request stored in a Client collection.
 * This is the persisted version that appears in the left sidebar.
 */
export interface SavedRequest {
  id: string
  name: string
  serverDomainId: string // Target server domain ID (empty = external URL mode)
  serverId: string       // Target server ID within the domain
  request: RequestConfig
}

// ============================================================================
// Opened Tab (runtime state)
// ============================================================================

/**
 * An opened tab in the editor area.
 * Can be linked to a SavedRequest or be a new unsaved request.
 */
export interface OpenedTab {
  id: string
  name: string
  savedRequestId?: string // Link to SavedRequest (undefined = new unsaved request)
  clientId?: string       // Which client the savedRequest belongs to
  domainId?: string       // Which domain the client belongs to
  isDirty: boolean        // Has unsaved changes
  serverDomainId: string
  serverId: string
  request: RequestConfig
  response?: ResponseData
}

// Legacy alias for backward compatibility
export type RequestTab = OpenedTab

// ============================================================================
// Client Domain Types
// ============================================================================

/**
 * A client with saved requests (collection).
 */
export interface Client {
  id: string
  name: string
  savedRequests: SavedRequest[]  // Renamed from 'tabs' to 'savedRequests'
  tabs?: RequestTab[]  // Legacy field for migration, will be removed
}

/**
 * Collection of related clients (workspace).
 */
export interface ClientDomain {
  id: string
  name: string
  clients: Client[]
}

// ============================================================================
// Server Domain Types
// ============================================================================

/**
 * Behavior type for server request processing.
 */
export type BehaviorType = 'forward' | 'mock' | 'return'

/**
 * Server runtime status.
 */
export type ServerStatus = 'stopped' | 'starting' | 'running' | 'stopping' | 'error'

/**
 * Forward behavior configuration.
 */
export interface ForwardConfig {
  target: string // URL or "server:{serverID}"
}

/**
 * Mock behavior configuration.
 */
export interface MockConfig {
  statusCode: number
  headers?: Record<string, string>
  body: any
}

/**
 * Return behavior configuration.
 */
export interface ReturnConfig {
  statusCode?: number
  headers?: Record<string, string>
}

/**
 * A single behavior in the server's behavior chain.
 */
export interface Behavior {
  type: BehaviorType
  config: any // Type-specific configuration (ForwardConfig | MockConfig | ReturnConfig)
}

/**
 * A local server managed by Nexus.
 * Processes requests through a behavior chain.
 */
export interface Server {
  id: string
  name: string
  port: number
  description?: string
  behaviors: Behavior[]
}

/**
 * Collection of related servers (by project/service/environment).
 */
export interface ServerDomain {
  id: string
  name: string
  servers: Server[]
}

// ============================================================================
// Top-level Container
// ============================================================================

/**
 * Workspace is the top-level container for all domain data.
 * Supports multiple server domains and client domains.
 */
export interface Workspace {
  serverDomains: ServerDomain[]
  clientDomains: ClientDomain[]
  
  // Active state for UI
  activeClientDomainId?: string
  activeClientId?: string
  activeTabId?: string
}

// ============================================================================
// Legacy Support (for migration)
// ============================================================================

/**
 * Legacy DomainData structure (single domain).
 * @deprecated Use Workspace instead
 */
export interface DomainData {
  serverDomain: ServerDomain
  clientDomain: ClientDomain
}

// ============================================================================
// Factory Functions
// ============================================================================

/**
 * Generates a unique ID (client-side fallback, prefer backend generation).
 */
export function generateId(): string {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  const hex = Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}

/**
 * Creates a new empty KVPair.
 */
export function newKVPair(): KVPair {
  return { key: '', value: '' }
}

/**
 * Creates a new default RequestConfig.
 */
export function newRequestConfig(): RequestConfig {
  return {
    method: 'GET',
    path: '',
    contentType: 'application/json',
    pathParams: [],
    params: [newKVPair()],
    headers: [newKVPair()],
    body: null
  }
}

/**
 * Creates a new SavedRequest with default values.
 */
export function newSavedRequest(name: string = 'New Request'): SavedRequest {
  return {
    id: generateId(),
    name,
    serverDomainId: '',
    serverId: '',
    request: newRequestConfig()
  }
}

/**
 * Creates a new OpenedTab from scratch (unsaved new request).
 */
export function newOpenedTab(name: string = 'New Request'): OpenedTab {
  return {
    id: generateId(),
    name,
    isDirty: false,
    serverDomainId: '',
    serverId: '',
    request: newRequestConfig()
  }
}

/**
 * Creates an OpenedTab from a SavedRequest.
 */
export function openedTabFromSavedRequest(
  savedRequest: SavedRequest, 
  clientId: string, 
  domainId: string
): OpenedTab {
  return {
    id: generateId(),
    name: savedRequest.name,
    savedRequestId: savedRequest.id,
    clientId,
    domainId,
    isDirty: false,
    serverDomainId: savedRequest.serverDomainId,
    serverId: savedRequest.serverId,
    request: JSON.parse(JSON.stringify(savedRequest.request)) // Deep copy
  }
}

/**
 * Legacy alias for backward compatibility.
 * @deprecated Use newOpenedTab instead
 */
export function newRequestTab(name: string = 'New Request'): RequestTab {
  return newOpenedTab(name)
}

/**
 * Creates a new mock behavior with the given body.
 */
export function newMockBehavior(statusCode: number = 200, body: any = { message: 'Hello from mock server' }): Behavior {
  return {
    type: 'mock',
    config: { statusCode, headers: { 'Content-Type': 'application/json' }, body }
  }
}

/**
 * Creates a new forward behavior with the given target.
 */
export function newForwardBehavior(target: string): Behavior {
  return {
    type: 'forward',
    config: { target }
  }
}

/**
 * Creates a new Server with default mock behavior.
 */
export function newServer(name: string = 'New Server', port: number = 8080): Server {
  return {
    id: generateId(),
    name,
    port,
    description: '',
    behaviors: [newMockBehavior(200, { message: `Hello from ${name}` })]
  }
}

/**
 * Creates a new ServerDomain with default values.
 */
export function newServerDomain(name: string = 'New Server Domain'): ServerDomain {
  return {
    id: generateId(),
    name,
    servers: []
  }
}

/**
 * Creates a new Client with empty savedRequests.
 */
export function newClient(name: string = 'New Client'): Client {
  return {
    id: generateId(),
    name,
    savedRequests: []
  }
}

/**
 * Creates a new ClientDomain with default values.
 */
export function newClientDomain(name: string = 'New Workspace'): ClientDomain {
  return {
    id: generateId(),
    name,
    clients: []
  }
}

/**
 * Creates a new default Workspace.
 */
export function newWorkspace(): Workspace {
  const serverDomain = newServerDomain('Default Servers')
  serverDomain.servers.push(newServer('mock-server', 8080))

  const clientDomain = newClientDomain('Default Workspace')
  const client = newClient('Default Client')
  // Add a default saved request
  const defaultRequest = newSavedRequest('Example Request')
  client.savedRequests.push(defaultRequest)
  clientDomain.clients.push(client)

  return {
    serverDomains: [serverDomain],
    clientDomains: [clientDomain],
    activeClientDomainId: clientDomain.id,
    activeClientId: client.id
  }
}