<template>
  <div class="app-container" :class="{ 'is-dragging': isDragging }">
    <!-- 客户端选项卡栏 -->
    <ClientTabs />

    <!-- 顶部请求栏 - 仅当有 Tab 时显示 -->
    <div v-if="hasActiveTab" class="request-bar">
      <select v-model="currentRequest.method" class="method-select" @change="syncToStore">
        <option v-for="m in methods" :key="m" :value="m">{{ m }}</option>
      </select>
      
      <!-- 服务器选择器（支持跨域选择） -->
      <ServerSelector
        :serverDomainId="currentRequest.serverDomainId"
        :serverId="currentRequest.serverId"
        :path="addressBarPath"
        @update:server="updateServerBinding"
        @update:path="updatePath"
      />
      
      <button class="send-btn" @click="sendRequest" :disabled="loading">
        {{ loading ? 'Sending...' : 'Send' }}
      </button>
    </div>

    <!-- 主内容区：可拖拽分割 -->
    <Splitpanes 
      :key="`split-${sidebarVisible}-${contractEditorVisible}`"
      class="main-content" 
      @resize="onPaneResize" 
      @resizestart="onResizeStart" 
      @resizeend="onResizeEnd"
    >
      <!-- 左侧：域管理侧边栏 (仅当有面板可见时显示) -->
      <Pane v-if="sidebarVisible" :size="18" :min-size="12" :max-size="30">
        <DomainSidebar />
      </Pane>

      <!-- 中间：请求配置 + 响应 (当 Contract Editor 隐藏时自动扩展) -->
      <Pane :size="contractEditorVisible ? 47 : (sidebarVisible ? 82 : 100)" :min-size="30">
        <!-- 空状态视图 - 没有 Tab 时显示 -->
        <div v-if="!hasActiveTab" class="empty-state-center">
          <div class="empty-state-content">
            <div class="empty-state-icon">📋</div>
            <h2 class="empty-state-title">No Request Open</h2>
            <p class="empty-state-desc">Create a new request tab or open a saved request from the sidebar</p>
            <button class="empty-state-btn" @click="createNewTab">
              + New Request
            </button>
          </div>
        </div>
        <!-- 有 Tab 时显示请求配置和响应 -->
        <Splitpanes v-else horizontal @resize="onPaneResize" @resizestart="onResizeStart" @resizeend="onResizeEnd">
          <!-- 请求配置 Tabs -->
          <Pane :size="50" :min-size="20">
            <div class="request-section">
              <div class="tabs">
                <button
                  v-for="tab in requestTabs"
                  :key="tab"
                  :class="['tab', { active: activeRequestTab === tab }]"
                  @click="activeRequestTab = tab"
                >
                  {{ tab }}
                </button>
              </div>
              <div class="tab-content">
                <div v-show="activeRequestTab === 'Params'" class="params-section">
                  <!-- Query Parameters -->
                  <div class="params-group">
                    <div class="params-group-header">Query Parameters</div>
                    <div class="kv-editor">
                      <div v-for="(param, i) in currentRequest.params" :key="i" class="kv-row">
                        <input v-model="param.key" placeholder="Key" @input="syncToStore" />
                        <input v-model="param.value" placeholder="Value" @input="syncToStore" />
                        <button @click="removeParam(i)" class="remove-btn">×</button>
                      </div>
                      <button @click="addParam" class="add-btn">+ Add Parameter</button>
                    </div>
                  </div>
                </div>
                <div v-show="activeRequestTab === 'Headers'" class="kv-editor">
                  <div v-for="(header, i) in currentRequest.headers" :key="i" class="kv-row">
                    <input v-model="header.key" placeholder="Header Name" @input="syncToStore" />
                    <input v-model="header.value" placeholder="Header Value" @input="syncToStore" />
                    <button @click="removeHeader(i)" class="remove-btn">×</button>
                  </div>
                  <button @click="addHeader" class="add-btn">+ Add Header</button>
                </div>
                <div v-show="activeRequestTab === 'Body'" class="body-section">
                  <!-- Content-Type 选择器 -->
                  <div class="content-type-selector">
                    <label>Content-Type:</label>
                    <select v-model="currentRequest.contentType" class="content-type-select" @change="syncToStore">
                      <option value="application/json">application/json</option>
                      <option value="application/x-www-form-urlencoded">x-www-form-urlencoded</option>
                      <option value="multipart/form-data">multipart/form-data</option>
                      <option value="none">none (no body)</option>
                    </select>
                  </div>
                  
                  <!-- JSON Body -->
                  <div v-if="currentRequest.contentType === 'application/json'" class="body-editor">
                    <div class="editor-wrapper">
                      <MonacoEditor
                        v-model="jsonBody"
                        language="json"
                        :theme="theme"
                      />
                      <div class="drag-overlay"></div>
                    </div>
                  </div>
                  
                  <!-- URL Encoded Body -->
                  <div v-else-if="currentRequest.contentType === 'application/x-www-form-urlencoded'" class="kv-editor body-kv">
                    <div v-for="(item, i) in formBody" :key="i" class="kv-row">
                      <input v-model="item.key" placeholder="Key" @input="syncBodyToStore" />
                      <input v-model="item.value" placeholder="Value" @input="syncBodyToStore" />
                      <button @click="removeFormItem(i)" class="remove-btn">×</button>
                    </div>
                    <button @click="addFormItem" class="add-btn">+ Add Field</button>
                  </div>
                  
                  <!-- Multipart Form Data -->
                  <div v-else-if="currentRequest.contentType === 'multipart/form-data'" class="kv-editor body-kv">
                    <div v-for="(item, i) in multipartBody" :key="i" class="kv-row multipart-row">
                      <input v-model="item.key" placeholder="Field Name" class="field-name" @input="syncBodyToStore" />
                      <select v-model="item.type" class="field-type" @change="syncBodyToStore">
                        <option value="text">Text</option>
                        <option value="file">File</option>
                      </select>
                      <input v-if="item.type === 'text'" v-model="item.value" placeholder="Value" class="field-value" @input="syncBodyToStore" />
                      <input v-else type="file" @change="handleFileSelect($event, i)" class="field-file" />
                      <button @click="removeMultipartItem(i)" class="remove-btn">×</button>
                    </div>
                    <button @click="addMultipartItem" class="add-btn">+ Add Field</button>
                  </div>
                  
                  <!-- No Body -->
                  <div v-else class="no-body-hint">
                    <span>No request body for this Content-Type</span>
                  </div>
                </div>
              </div>
            </div>
          </Pane>

          <!-- 响应区 -->
          <Pane :size="50" :min-size="20">
            <div class="response-section">
              <div class="response-header">
                <span class="response-title">Response</span>
                <template v-if="response.status">
                  <span :class="['status', statusClass]">{{ response.status }}</span>
                  <span class="time">{{ response.time }}ms</span>
                  <span class="size">{{ formatSize(response.size) }}</span>
                </template>
                <button 
                  v-if="response.status || response.body" 
                  @click="clearResponse" 
                  class="clear-btn"
                  title="Clear Response"
                >
                  Clear
                </button>
              </div>
              <div class="tabs">
                <button
                  v-for="tab in responseTabs"
                  :key="tab"
                  :class="['tab', { active: activeResponseTab === tab }]"
                  @click="activeResponseTab = tab"
                >
                  {{ tab }}
                </button>
              </div>
              <div class="tab-content response-content">
                <div v-show="activeResponseTab === 'Body'" class="response-body">
                  <div class="editor-wrapper">
                    <MonacoEditor
                      v-model="formattedResponseBody"
                      :language="responseBodyLanguage"
                      :theme="theme"
                      :readOnly="true"
                    />
                    <div class="drag-overlay"></div>
                  </div>
                </div>
                <div v-show="activeResponseTab === 'Headers'" class="response-headers">
                  <div v-if="response.headerList.length > 0" class="headers-table">
                    <div class="headers-row header-row-title">
                      <span class="header-key">Name</span>
                      <span class="header-value">Value</span>
                    </div>
                    <div v-for="(h, i) in response.headerList" :key="i" class="headers-row">
                      <span class="header-key">{{ h.key }}</span>
                      <span class="header-value">{{ h.value }}</span>
                    </div>
                  </div>
                  <div v-else class="no-headers">No headers</div>
                </div>
              </div>
            </div>
          </Pane>
        </Splitpanes>
      </Pane>

      <!-- 右侧：Contract JSON 编辑器 (可通过 View 菜单控制显示) -->
      <Pane v-if="contractEditorVisible" :size="35" :min-size="15">
        <div class="right-panel">
          <div class="panel-header clickable" @click="toggleContractEditorExpanded">
            <div class="panel-header-left">
              <span class="expand-icon" :class="{ expanded: contractEditorExpanded }">▶</span>
              <span>Contract JSON Editor</span>
            </div>
            <button v-if="contractEditorExpanded" @click.stop="formatContract" class="format-btn">Format</button>
          </div>
          <div v-if="contractEditorExpanded" :class="['validation-status', contractValid ? 'valid' : 'invalid']">
            {{ validationMessage }}
          </div>
          <div v-if="contractEditorExpanded" class="contract-editor">
            <div class="editor-wrapper">
              <MonacoEditor
                ref="contractEditorRef"
                v-model="contractJson"
                language="json"
                :theme="theme"
                @validate="onContractValidate"
              />
              <div class="drag-overlay"></div>
            </div>
          </div>
          <div v-else class="collapsed-hint">
            <span>Click header to expand</span>
          </div>
        </div>
      </Pane>
    </Splitpanes>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { Splitpanes, Pane } from 'splitpanes'
import MonacoEditor from './components/MonacoEditor.vue'
import DomainSidebar from './components/domain/DomainSidebar.vue'
import ClientTabs from './components/domain/ClientTabs.vue'
import ServerSelector from './components/domain/ServerSelector.vue'
import { workspaceStore } from './stores/workspaceStore'
import { viewStore } from './stores/viewStore'
import { EventsOn } from '../wailsjs/runtime/runtime'
import type { editor } from 'monaco-editor'
import type { KVPair } from './types/domain'
import { html as beautifyHtml, js as beautifyJs, css as beautifyCss } from 'js-beautify'

// 标志变量 - 必须在所有 watch 之前定义
let isUpdatingFromAddressBar = false
let isUpdatingFromRequest = false
let isUpdatingFromContract = false

// 视图控制 - 侧边栏和 Contract Editor 的显示
const sidebarVisible = computed(() => {
  const panels = viewStore.viewState.value.panels
  return panels.serverDomains.visible || panels.clientDomains.visible
})

const contractEditorVisible = computed(() => {
  return viewStore.viewState.value.panels.contractEditor.visible
})

const contractEditorExpanded = computed(() => {
  return viewStore.viewState.value.panels.contractEditor.expanded
})

function toggleContractEditorExpanded() {
  viewStore.togglePanelExpanded('contractEditor')
}

// 是否有活动的 Tab
const hasActiveTab = computed(() => {
  return workspaceStore.openedTabs.value.length > 0 && !!workspaceStore.activeTab.value
})

// 创建新 Tab
function createNewTab() {
  workspaceStore.addNewTab()
}

// 初始化 workspaceStore 和监听原生菜单事件
onMounted(async () => {
  await workspaceStore.init()
  
  // 监听原生菜单的视图切换事件 (legacy toggle)
  EventsOn('view:toggle', (panel: string) => {
    if (panel === 'serverDomains' || panel === 'clientDomains' || panel === 'contractEditor') {
      viewStore.togglePanelVisible(panel as 'serverDomains' | 'clientDomains' | 'contractEditor')
    }
  })

  // 监听原生菜单的视图设置事件 (直接设置状态，避免双重切换)
  EventsOn('view:set', (panel: string, visible: boolean) => {
    if (panel === 'serverDomains' || panel === 'clientDomains' || panel === 'contractEditor') {
      viewStore.setPanelVisible(panel as 'serverDomains' | 'clientDomains' | 'contractEditor', visible)
    }
  })
  
  EventsOn('view:action', (action: string) => {
    if (action === 'collapseAll') {
      viewStore.collapseAll()
    } else if (action === 'expandAll') {
      viewStore.expandAll()
    }
  })
})

// 拖动状态
const isDragging = ref(false)

// 面板大小调整时触发编辑器重新布局
const onPaneResize = () => {
  setTimeout(() => window.dispatchEvent(new Event('resize')), 50)
}

const onResizeStart = () => {
  isDragging.value = true
}

const onResizeEnd = () => {
  isDragging.value = false
  setTimeout(() => window.dispatchEvent(new Event('resize')), 50)
}

const theme = ref('vs-dark')
const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS']
const requestTabs = ['Params', 'Headers', 'Body']
const activeRequestTab = ref('Params')
const responseTabs = ['Body', 'Headers']
const activeResponseTab = ref('Body')

// 当前请求配置 - 从 workspaceStore 的活动 Tab 获取
const currentRequest = computed(() => {
  const tab = workspaceStore.activeTab.value
  if (!tab) {
    return {
      method: 'GET',
      serverDomainId: '',
      serverId: '',
      path: '',
      contentType: 'application/json',
      pathParams: [] as KVPair[],
      params: [{ key: '', value: '' }] as KVPair[],
      headers: [{ key: '', value: '' }] as KVPair[]
    }
  }
  return {
    method: tab.request.method,
    serverDomainId: tab.serverDomainId ?? '',
    serverId: tab.serverId ?? '',
    path: tab.request.path,
    contentType: tab.request.contentType,
    pathParams: tab.request.pathParams ?? [],
    params: tab.request.params.length > 0 ? tab.request.params : [{ key: '', value: '' }],
    headers: tab.request.headers.length > 0 ? tab.request.headers : [{ key: '', value: '' }]
  }
})

// 从 path 模板中检测路径参数（如 /classes/{schoolId} -> ['schoolId']）
const detectedPathParams = computed(() => {
  const path = currentRequest.value.path
  const regex = /\{(\w+)\}/g
  const params: string[] = []
  let match
  while ((match = regex.exec(path)) !== null) {
    params.push(match[1])
  }
  return params
})

// 获取路径参数值
function getPathParamValue(paramName: string): string {
  const pathParams = currentRequest.value.pathParams || []
  const found = pathParams.find(p => p.key === paramName)
  return found?.value ?? ''
}

// 设置路径参数值
function setPathParamValue(paramName: string, value: string) {
  const tab = workspaceStore.activeTab.value
  if (!tab) return
  
  const pathParams = [...(tab.request.pathParams || [])]
  const index = pathParams.findIndex(p => p.key === paramName)
  
  if (index >= 0) {
    pathParams[index] = { key: paramName, value }
  } else {
    pathParams.push({ key: paramName, value })
  }
  
  workspaceStore.updateTab({
    ...tab,
    request: { ...tab.request, pathParams }
  })
}

// Body 相关状态
const jsonBody = ref('{\n  \n}')
const formBody = ref<KVPair[]>([{ key: '', value: '' }])
const multipartBody = ref<Array<{ key: string; type: 'text' | 'file'; value: string; file?: File }>>([
  { key: '', type: 'text', value: '' }
])

// 当活动 Tab 变化时，同步 body 数据
watch(() => workspaceStore.activeTab.value, (tab) => {
  if (tab) {
    const body = tab.request.body
    if (tab.request.contentType === 'application/json') {
      if (body && typeof body === 'object') {
        jsonBody.value = JSON.stringify(body, null, 2)
      } else if (body && typeof body === 'string') {
        jsonBody.value = body
      } else {
        jsonBody.value = '{\n  \n}'
      }
    } else if (tab.request.contentType === 'application/x-www-form-urlencoded') {
      if (body && typeof body === 'object' && !Array.isArray(body)) {
        formBody.value = Object.entries(body).map(([key, value]) => ({ key, value: String(value) }))
      }
      if (formBody.value.length === 0) formBody.value = [{ key: '', value: '' }]
    } else if (tab.request.contentType === 'multipart/form-data') {
      if (body && typeof body === 'object' && !Array.isArray(body)) {
        multipartBody.value = Object.entries(body).map(([key, value]) => {
          if (typeof value === 'object' && value && '$file' in value) {
            return { key, type: 'file' as const, value: String((value as any).$file) }
          }
          return { key, type: 'text' as const, value: String(value) }
        })
      }
      if (multipartBody.value.length === 0) multipartBody.value = [{ key: '', type: 'text', value: '' }]
    }
  }
}, { immediate: true })

// 响应数据
const response = ref({
  status: '',
  time: 0,
  size: 0,
  body: '',
  headers: '',
  headerList: [] as Array<{ key: string; value: string }>,
  contentType: ''
})

// 检测内容的实际类型（基于内容而非 Content-Type）
function detectContentType(body: string, contentTypeHeader: string): 'json' | 'html' | 'xml' | 'javascript' | 'css' | 'plaintext' {
  const ct = contentTypeHeader.toLowerCase()
  const trimmedBody = body.trim()
  const lowerBody = trimmedBody.toLowerCase()
  
  // 先根据 Content-Type 判断
  if (ct.includes('json')) return 'json'
  if (ct.includes('xml') && !ct.includes('html')) return 'xml'
  if (ct.includes('html')) return 'html'
  if (ct.includes('javascript')) return 'javascript'
  if (ct.includes('css')) return 'css'
  
  // Content-Type 不明确时，根据内容判断
  // JSON 检测
  if ((trimmedBody.startsWith('{') && trimmedBody.endsWith('}')) ||
      (trimmedBody.startsWith('[') && trimmedBody.endsWith(']'))) {
    try {
      JSON.parse(trimmedBody)
      return 'json'
    } catch {}
  }
  
  // XML 检测 (<?xml 开头)
  if (lowerBody.startsWith('<?xml')) {
    return 'xml'
  }
  
  // HTML 检测 - 更宽松的匹配（大小写不敏感）
  if (lowerBody.startsWith('<!doctype') || 
      lowerBody.startsWith('<html') ||
      (trimmedBody.startsWith('<') && /<(head|body|div|span|p|a|script|style|meta|link|title|!--)\b/i.test(trimmedBody))) {
    return 'html'
  }
  
  return 'plaintext'
}

// 根据内容获取响应 Body 的语言类型
const responseBodyLanguage = computed(() => {
  const detectedType = detectContentType(response.value.body, response.value.contentType)
  console.log('[Language] Detected language:', detectedType, 'for Content-Type:', response.value.contentType)
  return detectedType
})

// 格式化后的响应 Body (响应式)
const formattedResponseBody = ref('')

// 异步格式化函数
async function formatBodyAsync(body: string, contentType: string): Promise<string> {
  if (!body) return ''
  
  const detectedType = detectContentType(body, contentType)
  console.log('[Format] Detected type:', detectedType, 'from Content-Type:', contentType)
  
  // JSON 格式化
  if (detectedType === 'json') {
    try {
      return JSON.stringify(JSON.parse(body), null, 2)
    } catch {
      return body
    }
  }
  
  // HTML/XML 格式化 - 使用 js-beautify（对非标准 HTML 容错性更好）
  if (detectedType === 'html' || detectedType === 'xml') {
    try {
      console.log('[Format] Calling js-beautify for HTML/XML...')
      const result = beautifyHtml(body, {
        indent_size: 2,
        indent_char: ' ',
        max_preserve_newlines: 1,
        preserve_newlines: true,
        indent_inner_html: true,
        wrap_line_length: 120,
        unformatted: ['code', 'pre', 'script'],
        content_unformatted: ['pre', 'code']
      })
      console.log('[Format] js-beautify success')
      return result
    } catch (err) {
      console.error('[Format] js-beautify error:', err)
      return body
    }
  }
  
  // JavaScript 格式化
  if (detectedType === 'javascript') {
    try {
      return beautifyJs(body, { indent_size: 2 })
    } catch {
      return body
    }
  }
  
  // CSS 格式化
  if (detectedType === 'css') {
    try {
      return beautifyCss(body, { indent_size: 2 })
    } catch {
      return body
    }
  }
  
  return body
}

// 监听响应变化，自动格式化
watch(() => [response.value.body, response.value.contentType], async ([body, contentType]) => {
  if (body) {
    // 先显示原始内容
    formattedResponseBody.value = body as string
    // 异步格式化
    const formatted = await formatBodyAsync(body as string, contentType as string)
    formattedResponseBody.value = formatted
  } else {
    formattedResponseBody.value = ''
  }
}, { immediate: true })

const loading = ref(false)

// Contract JSON
const contractJson = ref(JSON.stringify({
  name: 'NewRequest',
  request: {
    meta: { method: 'GET', path: '/' },
    example: {},
  },
  response: { example: null },
}, null, 2))

const contractValid = ref(true)
const validationMessage = ref('✓ Valid JSON')
const contractEditorRef = ref<InstanceType<typeof MonacoEditor>>()

const statusClass = computed(() => {
  const status = response.value.status
  if (!status) return ''
  const code = parseInt(status)
  if (code >= 200 && code < 300) return 'success'
  if (code >= 400) return 'error'
  return 'warning'
})

// 同步到 Store
function syncToStore() {
  const tab = workspaceStore.activeTab.value
  if (!tab) return
  
  // 保存当前地址栏状态
  const savedUrl = addressBarPath.value
  const savedBaseURL = currentBaseURL.value
  
  // 阻止 watchers 覆盖地址栏
  isUpdatingFromAddressBar = true
  
  workspaceStore.updateTab({
    ...tab,
    request: {
      ...tab.request,
      method: currentRequest.value.method,
      path: currentRequest.value.path,
      contentType: currentRequest.value.contentType,
      params: currentRequest.value.params,
      headers: currentRequest.value.headers
    },
    serverDomainId: currentRequest.value.serverDomainId,
    serverId: currentRequest.value.serverId
  })
  
  // 等待 Vue 完成所有更新后恢复地址栏
  nextTick(() => {
    addressBarPath.value = savedUrl
    currentBaseURL.value = savedBaseURL
    isUpdatingFromAddressBar = false
  })
}

function syncBodyToStore() {
  const tab = workspaceStore.activeTab.value
  if (!tab) return

  let body: any = null
  if (tab.request.contentType === 'application/json') {
    try {
      body = JSON.parse(jsonBody.value)
    } catch {}
  } else if (tab.request.contentType === 'application/x-www-form-urlencoded') {
    const data: Record<string, string> = {}
    formBody.value.forEach(item => { if (item.key) data[item.key] = item.value })
    body = Object.keys(data).length > 0 ? data : null
  } else if (tab.request.contentType === 'multipart/form-data') {
    const data: Record<string, any> = {}
    multipartBody.value.forEach(item => {
      if (item.key) {
        data[item.key] = item.type === 'file' ? { $file: item.value } : item.value
      }
    })
    body = Object.keys(data).length > 0 ? data : null
  }

  workspaceStore.updateTab({
    ...tab,
    request: { ...tab.request, body }
  })
}

// Watch jsonBody 变化同步
watch(jsonBody, () => {
  if (currentRequest.value.contentType === 'application/json') {
    syncBodyToStore()
  }
})

function updateServerBinding(binding: { serverDomainId: string; serverId: string }) {
  console.log('updateServerBinding called:', binding)
  let tab = workspaceStore.activeTab.value
  
  // 如果没有活动的 Tab，创建一个新的
  if (!tab) {
    console.log('No active tab, creating new tab...')
    tab = workspaceStore.addNewTab()
  }
  
  console.log('activeTab:', tab.id, 'current serverDomainId:', tab.serverDomainId, 'serverId:', tab.serverId)
  
  const updatedTab = {
    ...tab,
    serverDomainId: binding.serverDomainId,
    serverId: binding.serverId
  }
  console.log('Calling updateTab with:', updatedTab.serverDomainId, updatedTab.serverId)
  workspaceStore.updateTab(updatedTab)
  console.log('After updateTab, activeTab:', workspaceStore.activeTab.value?.serverDomainId, workspaceStore.activeTab.value?.serverId)
}

// 地址栏显示的路径（独立状态，不自动替换回模板）
const addressBarPath = ref('')

// 当前的 baseURL（独立追踪，避免时序问题）
const currentBaseURL = ref('')

// 构建完整的地址栏 URL（包含 baseURL、path、pathParams、queryParams）
function buildAddressBarUrl(tab: any): string {
  let path = tab.request.path
  
  // 替换路径参数
  const pathParams = tab.request.pathParams || []
  pathParams.forEach((p: any) => {
    if (p.key && p.value) {
      path = path.replace(`{${p.key}}`, p.value)
    }
  })
  
  // 添加查询参数
  const queryParams = tab.request.params?.filter((p: any) => p.key) || []
  if (queryParams.length > 0) {
    const searchParams = new URLSearchParams()
    queryParams.forEach((p: any) => searchParams.append(p.key, p.value))
    path += (path.includes('?') ? '&' : '?') + searchParams.toString()
  }
  
  // External URL 模式：添加 baseURL
  if (!tab.serverDomainId || !tab.serverId) {
    // 优先使用已追踪的 currentBaseURL（最稳定）
    if (currentBaseURL.value && !path.startsWith('http')) {
      return currentBaseURL.value + path
    }
    
    // 后备方案：从当前地址栏提取 baseURL
    if (!path.startsWith('http') && addressBarPath.value.startsWith('http')) {
      try {
        const currentUrl = new URL(addressBarPath.value)
        currentBaseURL.value = currentUrl.origin // 更新追踪
        return currentUrl.origin + path
      } catch {}
    }
    
    // 最后方案：从 Contract JSON 获取 baseURL
    try {
      const contract = JSON.parse(contractJson.value)
      const baseURL = contract.request?.meta?.baseURL
      if (baseURL && !path.startsWith('http')) {
        currentBaseURL.value = baseURL // 更新追踪
        return baseURL + path
      }
    } catch {}
  }
  
  return path
}

// 当活动 Tab 变化时，初始化地址栏显示
watch(() => workspaceStore.activeTab.value, (tab) => {
  // 如果是从地址栏更新触发的，不要覆盖用户输入
  if (isUpdatingFromAddressBar) return
  
  if (tab) {
    addressBarPath.value = buildAddressBarUrl(tab)
  }
}, { immediate: true })

// 当 Query Params 变化时，同步更新地址栏
watch(() => workspaceStore.activeTab.value?.request.params, () => {
  if (isUpdatingFromAddressBar) return
  const tab = workspaceStore.activeTab.value
  if (tab) {
    addressBarPath.value = buildAddressBarUrl(tab)
  }
}, { deep: true })

// 同步 pathParams 变化到地址栏（当用户在 Path Parameters 输入框中修改时）
watch(() => workspaceStore.activeTab.value?.request.pathParams, (pathParams) => {
  // 如果是从地址栏更新的，不要再触发地址栏更新
  if (isUpdatingFromAddressBar) return
  
  const tab = workspaceStore.activeTab.value
  if (!tab) return
  
  // 从 Contract JSON 获取模板
  let templatePath = tab.request.path
  try {
    const contract = JSON.parse(contractJson.value)
    const contractPath = contract.request?.meta?.path
    if (contractPath && hasPathParams(contractPath)) {
      templatePath = contractPath
    }
  } catch {}
  
  if (hasPathParams(templatePath) && pathParams) {
    let path = templatePath
    pathParams.forEach(p => {
      if (p.key && p.value) {
        path = path.replace(`{${p.key}}`, p.value)
      }
    })
    addressBarPath.value = path
  }
}, { deep: true })

// 检查路径是否包含模板参数
function hasPathParams(path: string): boolean {
  return /\{(\w+)\}/.test(path)
}

// 根据模板和用户输入解析路径参数
function parsePathParamsFromInput(templatePath: string, inputPath: string): Record<string, string> | null {
  // 检查模板中是否有参数
  if (!hasPathParams(templatePath)) {
    return null
  }
  
  // 将模板转换为正则表达式，捕获参数值
  // 例如 /classes/{schoolId} -> /classes/([^/]*)
  // 使用 * 而非 + 允许空值匹配
  const paramNames: string[] = []
  // 需要转义正则特殊字符，但保留 {param} 的位置
  const escapedPattern = templatePath
    .replace(/[.*+?^${}()|[\]\\]/g, '\\$&') // 转义所有特殊字符
    .replace(/\\\{(\w+)\\\}/g, (_, name) => { // 把转义后的 \{param\} 换回捕获组
      paramNames.push(name)
      return '([^/]*)' // 使用 * 允许空值
    })
  
  try {
    const regex = new RegExp(`^${escapedPattern}$`)
    const match = inputPath.match(regex)
    
    if (match && paramNames.length > 0) {
      const result: Record<string, string> = {}
      paramNames.forEach((name, i) => {
        result[name] = match[i + 1]
      })
      return result
    }
  } catch (e) {
    console.error('[parsePathParams] Regex error:', e)
  }
  return null
}

function updatePath(inputPath: string) {
  console.log('>>> [updatePath] CALLED <<<', inputPath)
  
  const tab = workspaceStore.activeTab.value
  if (!tab) {
    console.log('[updatePath] No active tab!')
    return
  }
  
  // 解析 URL：分离 baseURL、pathname 和 query
  let pathname = inputPath
  let queryString = ''
  let baseURL = ''
  
  // 尝试解析为完整 URL
  try {
    const url = new URL(inputPath)
    baseURL = url.origin
    pathname = url.pathname
    queryString = url.search.slice(1) // 去掉 ?
    // 持久化 baseURL，防止后续 watcher 丢失
    currentBaseURL.value = baseURL
  } catch {
    // 不是完整 URL，检查是否有 query string
    const queryIndex = inputPath.indexOf('?')
    if (queryIndex >= 0) {
      pathname = inputPath.slice(0, queryIndex)
      queryString = inputPath.slice(queryIndex + 1)
    }
  }
  
  // 解析 query parameters
  const newParams: KVPair[] = []
  if (queryString) {
    const searchParams = new URLSearchParams(queryString)
    searchParams.forEach((value, key) => {
      newParams.push({ key, value })
    })
  }
  if (newParams.length === 0) {
    newParams.push({ key: '', value: '' })
  }
  
  // 优先从 Contract JSON 获取原始模板
  let templatePath = tab.request.path
  try {
    const contract = JSON.parse(contractJson.value)
    const contractPath = contract.request?.meta?.path
    if (contractPath && hasPathParams(contractPath)) {
      templatePath = contractPath
    }
  } catch {}
  
  // 如果模板中有参数，尝试解析路径参数
  let newPathParams = tab.request.pathParams || []
  if (hasPathParams(templatePath)) {
    const parsedParams = parsePathParamsFromInput(templatePath, pathname)
    if (parsedParams) {
      newPathParams = [...(tab.request.pathParams || [])]
      Object.entries(parsedParams).forEach(([key, value]) => {
        const index = newPathParams.findIndex(p => p.key === key)
        if (index >= 0) {
          newPathParams[index] = { key, value }
        } else {
          newPathParams.push({ key, value })
        }
      })
      // 有模板且解析成功，不更新 path（保持模板）
      pathname = templatePath
    }
  }
  
  // 更新地址栏显示
  isUpdatingFromAddressBar = true
  addressBarPath.value = inputPath
  
  // 更新 Tab 数据
  workspaceStore.updateTab({
    ...tab,
    request: { 
      ...tab.request, 
      path: pathname,
      params: newParams,
      pathParams: newPathParams
    }
  })
  
  setTimeout(() => { isUpdatingFromAddressBar = false }, 0)
}

// Params 操作
const addParam = () => {
  const tab = workspaceStore.activeTab.value
  if (tab) {
    const params = [...tab.request.params, { key: '', value: '' }]
    workspaceStore.updateTab({ ...tab, request: { ...tab.request, params } })
  }
}

const removeParam = (i: number) => {
  const tab = workspaceStore.activeTab.value
  if (!tab) return
  
  const params = [...tab.request.params]
  if (params.length > 1) {
    params.splice(i, 1)
  } else {
    params[0] = { key: '', value: '' }
  }
  workspaceStore.updateTab({ ...tab, request: { ...tab.request, params } })
}

// Headers 操作
const addHeader = () => {
  const tab = workspaceStore.activeTab.value
  if (tab) {
    const headers = [...tab.request.headers, { key: '', value: '' }]
    workspaceStore.updateTab({ ...tab, request: { ...tab.request, headers } })
  }
}

const removeHeader = (i: number) => {
  const tab = workspaceStore.activeTab.value
  if (!tab) return
  
  const headers = [...tab.request.headers]
  if (headers.length > 1) {
    headers.splice(i, 1)
  } else {
    headers[0] = { key: '', value: '' }
  }
  workspaceStore.updateTab({ ...tab, request: { ...tab.request, headers } })
}

// Form Body 操作
const addFormItem = () => formBody.value.push({ key: '', value: '' })
const removeFormItem = (i: number) => {
  if (formBody.value.length > 1) formBody.value.splice(i, 1)
  else formBody.value[0] = { key: '', value: '' }
  syncBodyToStore()
}

// Multipart Body 操作
const addMultipartItem = () => multipartBody.value.push({ key: '', type: 'text', value: '' })
const removeMultipartItem = (i: number) => {
  if (multipartBody.value.length > 1) multipartBody.value.splice(i, 1)
  else multipartBody.value[0] = { key: '', type: 'text', value: '' }
  syncBodyToStore()
}

const handleFileSelect = (event: Event, index: number) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (file) {
    multipartBody.value[index].file = file
    multipartBody.value[index].value = file.name
    syncBodyToStore()
  }
}

const formatSize = (bytes: number) => {
  if (bytes < 1024) return bytes + ' B'
  return (bytes / 1024).toFixed(1) + ' KB'
}

const onContractValidate = (markers: editor.IMarker[]) => {
  const errors = markers.filter(m => m.severity === 8)
  contractValid.value = errors.length === 0
  if (errors.length > 0) {
    const e = errors[0]
    validationMessage.value = `✗ Line ${e.startLineNumber}: ${e.message}`
  } else {
    validationMessage.value = '✓ Valid JSON'
  }
}

const formatContract = () => {
  contractEditorRef.value?.format()
}

// Contract JSON → Request 同步（反向绑定）
watch(contractJson, (json) => {
  if (isUpdatingFromRequest) return
  try {
    const contract = JSON.parse(json)
    isUpdatingFromContract = true
    
    // 提前设置 currentBaseURL（确保 Tab watcher 触发时已经有值）
    const baseURL = contract.request?.meta?.baseURL
    if (baseURL) {
      currentBaseURL.value = baseURL
    }
    
    const tab = workspaceStore.activeTab.value
    if (tab) {
      if (contract.request?.meta?.method) {
        tab.request.method = contract.request.meta.method
      }
      
      if (contract.request?.meta?.path) {
        tab.request.path = contract.request.meta.path
      }
      
      if (contract.request?.meta?.contentType) {
        tab.request.contentType = contract.request.meta.contentType
      }
      
      const headers = contract.request?.example?.headers
      if (headers && typeof headers === 'object') {
        tab.request.headers = Object.entries(headers).map(([key, value]) => ({ key, value: String(value) }))
        if (tab.request.headers.length === 0) tab.request.headers = [{ key: '', value: '' }]
      }
      
      const params = contract.request?.example?.query
      if (params && typeof params === 'object') {
        tab.request.params = Object.entries(params).map(([key, value]) => ({ key, value: String(value) }))
        if (tab.request.params.length === 0) tab.request.params = [{ key: '', value: '' }]
      }
      
      // pathParams 同步
      const pathParams = contract.request?.example?.pathParams
      if (pathParams && typeof pathParams === 'object') {
        tab.request.pathParams = Object.entries(pathParams).map(([key, value]) => ({ key, value: String(value) }))
      }
      
      const body = contract.request?.example?.body
      if (body !== undefined) {
        tab.request.body = body
        if (tab.request.contentType === 'application/json' && body) {
          jsonBody.value = typeof body === 'object' ? JSON.stringify(body, null, 2) : String(body)
        }
      }
      
      workspaceStore.updateTab(tab)
    }
    
    setTimeout(() => { isUpdatingFromContract = false }, 0)
  } catch {
    // JSON 无效时不更新
  }
})

// Request → Contract JSON 同步（正向绑定）
watch([() => workspaceStore.activeTab.value, jsonBody], () => {
  if (isUpdatingFromContract) return
  isUpdatingFromRequest = true
  
  const tab = workspaceStore.activeTab.value
  if (!tab) return
  
  try {
    const contract = JSON.parse(contractJson.value)
    contract.request.meta.method = tab.request.method
    // 保持 path 模板不变：如果 Contract 中已有模板参数，不覆盖
    const existingPath = contract.request.meta.path || ''
    if (!hasPathParams(existingPath)) {
      // 原来没有模板参数，直接使用 tab 的 path
      contract.request.meta.path = tab.request.path
    }
    // 如果 existingPath 有模板参数（如 /classes/{schoolId}），保持不变
    
    if (tab.request.contentType && tab.request.contentType !== 'none') {
      contract.request.meta.contentType = tab.request.contentType
    } else {
      delete contract.request.meta.contentType
    }
    
    // baseURL 处理：只更新 baseURL，不覆盖 path 模板
    if (tab.serverDomainId && tab.serverId) {
      // 选择了服务器模式：baseURL 为服务器地址
      const server = workspaceStore.getServer(tab.serverDomainId, tab.serverId)
      if (server) {
        const actualPort = workspaceStore.serverActualPorts.value[tab.serverId] || server.port
        contract.request.meta.baseURL = `http://localhost:${actualPort}`
      }
    } else {
      // External URL 模式：尝试从 path 解析 baseURL
      try {
        const urlObj = new URL(tab.request.path)
        contract.request.meta.baseURL = urlObj.origin
        // 只有在 path 是完整 URL 时才提取 pathname
        if (!tab.request.path.includes('{')) {
          contract.request.meta.path = urlObj.pathname + urlObj.search
        }
      } catch {
        // path 不是完整 URL，保持原样
      }
    }
    
    const headers: Record<string, string> = {}
    tab.request.headers.forEach(h => { if (h.key) headers[h.key] = h.value })
    if (Object.keys(headers).length > 0) {
      contract.request.example.headers = headers
    } else {
      delete contract.request.example.headers
    }
    
    const params: Record<string, string> = {}
    tab.request.params.forEach(p => { if (p.key) params[p.key] = p.value })
    if (Object.keys(params).length > 0) {
      contract.request.example.query = params
    } else {
      delete contract.request.example.query
    }
    
    // pathParams 同步到 Contract（仅保存非空值）
    const pathParams: Record<string, any> = {}
    tab.request.pathParams?.forEach(p => { 
      if (p.key && p.value) {  // 只保存有值的参数
        // 尝试转换为数字（保留类型）
        const numValue = Number(p.value)
        pathParams[p.key] = !isNaN(numValue) ? numValue : p.value
      }
    })
    if (Object.keys(pathParams).length > 0) {
      contract.request.example.pathParams = pathParams
    } else {
      delete contract.request.example.pathParams
    }
    
    // Body 处理
    if (tab.request.contentType === 'application/json') {
      try {
        contract.request.example.body = JSON.parse(jsonBody.value)
      } catch {
        delete contract.request.example.body
      }
    } else if (tab.request.body) {
      contract.request.example.body = tab.request.body
    } else {
      delete contract.request.example.body
    }
    
    contractJson.value = JSON.stringify(contract, null, 2)
  } catch {}
  
  setTimeout(() => { isUpdatingFromRequest = false }, 0)
}, { deep: true })

// 根据 contentType 获取 body 字符串
const getBodyForRequest = (): { body: string; contentType: string } => {
  const ct = currentRequest.value.contentType
  
  if (ct === 'application/json') {
    const body = jsonBody.value.trim()
    if (body && body !== '{\n  \n}') {
      return { body, contentType: 'application/json' }
    }
    return { body: '', contentType: '' }
  }
  
  if (ct === 'application/x-www-form-urlencoded') {
    const data = formBody.value.filter(item => item.key)
    if (data.length > 0) {
      const params = new URLSearchParams()
      data.forEach(item => params.append(item.key, item.value))
      return { body: params.toString(), contentType: 'application/x-www-form-urlencoded' }
    }
    return { body: '', contentType: '' }
  }
  
  if (ct === 'multipart/form-data') {
    const data = multipartBody.value.filter(item => item.key)
    if (data.length > 0) {
      return { body: JSON.stringify(data), contentType: 'multipart/form-data' }
    }
    return { body: '', contentType: '' }
  }
  
  return { body: '', contentType: '' }
}

// 清除响应数据
const clearResponse = () => {
  response.value = {
    status: '',
    time: 0,
    size: 0,
    body: '',
    headers: '',
    headerList: [],
    contentType: ''
  }
}

const sendRequest = async () => {
  const tab = workspaceStore.activeTab.value
  if (!tab) return
  
  // 构建最终请求 URL
  let finalUrl = ''
  
  if (tab.serverDomainId && tab.serverId) {
    // Server 模式：baseURL + path（替换路径参数）
    let url = workspaceStore.getTabUrl(tab)
    if (!url) return
    const pathParams = tab.request.pathParams || []
    pathParams.forEach(p => {
      if (p.key && p.value) {
        url = url!.replace(`{${p.key}}`, encodeURIComponent(p.value))
      }
    })
    finalUrl = url
  } else {
    // External URL 模式：从地址栏获取完整 URL（去掉 query，后面单独加）
    const addrUrl = addressBarPath.value
    const qIdx = addrUrl.indexOf('?')
    finalUrl = qIdx >= 0 ? addrUrl.slice(0, qIdx) : addrUrl
  }
  
  if (!finalUrl) return
  
  loading.value = true
  response.value = { status: '', time: 0, size: 0, body: '', headers: '', headerList: [], contentType: '' }
  
  try {
    // 添加 query params
    const params = tab.request.params.filter(p => p.key)
    if (params.length > 0) {
      const searchParams = new URLSearchParams()
      params.forEach(p => searchParams.append(p.key, p.value))
      finalUrl += (finalUrl.includes('?') ? '&' : '?') + searchParams.toString()
    }
    
    const headers: Record<string, string> = {}
    tab.request.headers.forEach(h => { if (h.key) headers[h.key] = h.value })
    
    // Get body
    const { body, contentType } = getBodyForRequest()
    if (contentType && !headers['Content-Type']) {
      headers['Content-Type'] = contentType
    }
    
    const { SendRequest } = await import('../wailsjs/go/main/App')
    const result = await SendRequest(tab.request.method, finalUrl, headers, body)
    
    response.value = {
      status: result.status,
      time: result.time,
      size: result.size,
      body: result.body,
      headers: result.headers,
      headerList: result.headerList || [],
      contentType: result.contentType || ''
    }
    
    // Update response in contract
    if (result.body) {
      try {
        const contract = JSON.parse(contractJson.value)
        contract.response.example = JSON.parse(result.body)
        contractJson.value = JSON.stringify(contract, null, 2)
      } catch {}
    }
  } catch (err: any) {
    response.value.body = JSON.stringify({ error: err.message || String(err) }, null, 2)
  } finally {
    loading.value = false
  }
}
</script>

<style>
* { box-sizing: border-box; margin: 0; padding: 0; -webkit-user-select: none; user-select: none; }
.monaco-editor, .monaco-editor * { -webkit-user-select: text; user-select: text; }
.headers-table, .headers-table * { -webkit-user-select: text; user-select: text; cursor: text; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1e1e1e; color: #d4d4d4; }
.app-container { display: flex; flex-direction: column; height: 100vh; padding: 8px; gap: 8px; }
.request-bar { display: flex; gap: 8px; padding: 8px; background: #252526; border-radius: 4px; align-items: stretch; }
.method-select { padding: 8px 12px; background: #3c3c3c; border: 1px solid #555; border-radius: 4px; color: #d4d4d4; font-weight: bold; cursor: pointer; flex-shrink: 0; height: 36px; box-sizing: border-box; }
.send-btn { padding: 8px 24px; background: #007acc; border: none; border-radius: 4px; color: white; font-weight: bold; cursor: pointer; flex-shrink: 0; height: 36px; box-sizing: border-box; }
.send-btn:hover { background: #0098ff; }
.send-btn:disabled { background: #555; cursor: not-allowed; }
.main-content { display: flex; flex: 1; gap: 8px; overflow: hidden; }
.left-panel { flex: 2; display: flex; flex-direction: column; gap: 8px; }
.right-panel { display: flex; flex-direction: column; background: #252526; border-radius: 4px; overflow: hidden; height: 100%; }
.request-section, .response-section { flex: 1; display: flex; flex-direction: column; background: #252526; border-radius: 4px; overflow: hidden; }
.tabs { display: flex; background: #2d2d30; border-bottom: 1px solid #3c3c3c; }
.tab { padding: 8px 16px; background: transparent; border: none; color: #969696; cursor: pointer; }
.tab:hover { color: #d4d4d4; }
.tab.active { color: #d4d4d4; border-bottom: 2px solid #007acc; }
.tab-content { flex: 1; padding: 8px; overflow: hidden; }
.kv-editor { display: flex; flex-direction: column; gap: 4px; }
.kv-row { display: flex; gap: 4px; }
.kv-row input { flex: 1; padding: 6px 10px; background: #3c3c3c; border: 1px solid #555; border-radius: 4px; color: #d4d4d4; }
.remove-btn { padding: 6px 10px; background: #5a1d1d; border: none; border-radius: 4px; color: #d4d4d4; cursor: pointer; }
.add-btn { padding: 6px 12px; background: #3c3c3c; border: 1px dashed #555; border-radius: 4px; color: #969696; cursor: pointer; margin-top: 4px; }
.add-btn:hover { border-color: #007acc; color: #007acc; }

/* Params Section 样式 */
.params-section { display: flex; flex-direction: column; gap: 16px; overflow-y: auto; height: 100%; }
.params-group { }
.params-group-header { font-size: 12px; font-weight: 600; color: #888; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 8px; padding-left: 4px; }
.param-key-fixed { background: #2d2d30 !important; color: #569cd6 !important; cursor: not-allowed; font-family: monospace; }

/* Body Section 样式 */
.body-section { display: flex; flex-direction: column; height: 100%; gap: 8px; }
.content-type-selector { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.content-type-selector label { color: #969696; font-size: 13px; }
.content-type-select { padding: 6px 10px; background: #3c3c3c; border: 1px solid #555; border-radius: 4px; color: #d4d4d4; cursor: pointer; }
.content-type-select:focus { outline: none; border-color: #007acc; }
.body-kv { flex: 1; overflow-y: auto; }
.body-editor { flex: 1; min-height: 150px; }
.no-body-hint { display: flex; align-items: center; justify-content: center; flex: 1; color: #969696; font-style: italic; }

/* Multipart 样式 */
.multipart-row { display: flex; align-items: center; }
.multipart-row .field-name { flex: 1; max-width: 150px; }
.multipart-row .field-type { width: 80px; padding: 6px 8px; background: #3c3c3c; border: 1px solid #555; border-radius: 4px; color: #d4d4d4; cursor: pointer; }
.multipart-row .field-value { flex: 2; }
.multipart-row .field-file { flex: 2; padding: 4px; background: #3c3c3c; border: 1px solid #555; border-radius: 4px; color: #d4d4d4; }
.multipart-row .field-file::file-selector-button { padding: 4px 8px; background: #505050; border: none; border-radius: 3px; color: #d4d4d4; cursor: pointer; margin-right: 8px; }
.multipart-row .field-file::file-selector-button:hover { background: #606060; }

.response-header { display: flex; align-items: center; gap: 16px; padding: 8px 12px; background: #2d2d30; }
.response-title { font-weight: bold; }
.status { padding: 2px 8px; border-radius: 4px; font-weight: bold; }
.status.success { background: #2d5a2d; color: #4caf50; }
.status.error { background: #5a2d2d; color: #f44336; }
.status.warning { background: #5a4d2d; color: #ff9800; }
.time, .size { color: #969696; font-size: 12px; }
.clear-btn { margin-left: auto; padding: 4px 12px; background: #3c3c3c; border: 1px solid #555; border-radius: 4px; color: #969696; font-size: 12px; cursor: pointer; transition: all 0.15s; }
.clear-btn:hover { border-color: #f44336; color: #f44336; background: #5a2d2d; }
.response-content { flex: 1; }
.response-body { height: 100%; }
.response-headers { height: 100%; overflow-y: auto; }
.headers-table { display: flex; flex-direction: column; }
.headers-row { display: flex; border-bottom: 1px solid #3c3c3c; }
.headers-row:last-child { border-bottom: none; }
.header-row-title { background: #2d2d30; font-weight: 600; font-size: 12px; color: #888; text-transform: uppercase; }
.header-key { flex: 0 0 200px; padding: 8px 12px; color: #569cd6; font-family: monospace; font-size: 13px; word-break: break-all; border-right: 1px solid #3c3c3c; }
.header-value { flex: 1; padding: 8px 12px; color: #ce9178; font-family: monospace; font-size: 13px; word-break: break-all; }
.header-row-title .header-key, .header-row-title .header-value { color: #888; }
.no-headers { display: flex; align-items: center; justify-content: center; height: 100%; color: #666; font-style: italic; }
.panel-header { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; background: #2d2d30; border-bottom: 1px solid #3c3c3c; }
.panel-header.clickable { cursor: pointer; user-select: none; }
.panel-header.clickable:hover { background: #353535; }
.panel-header-left { display: flex; align-items: center; gap: 8px; }
.panel-header .expand-icon { font-size: 10px; color: #888; transition: transform 0.15s; }
.panel-header .expand-icon.expanded { transform: rotate(90deg); }
.format-btn { padding: 4px 12px; background: #3c3c3c; border: 1px solid #555; border-radius: 4px; color: #d4d4d4; cursor: pointer; }
.format-btn:hover { border-color: #007acc; }
.validation-status { padding: 4px 12px; font-size: 12px; }
.validation-status.valid { color: #4caf50; }
.validation-status.invalid { color: #f44336; }
.contract-editor { flex: 1; overflow: hidden; position: relative; }
.body-editor, .response-body, .contract-editor { display: block; }

/* Splitpanes 基础布局样式 */
.splitpanes { display: flex; width: 100%; height: 100%; }
.splitpanes--vertical { flex-direction: row; }
.splitpanes--horizontal { flex-direction: column; }
.splitpanes--dragging { user-select: none; }
.splitpanes__pane { width: 100%; height: 100%; overflow: hidden; }
.splitpanes--vertical > .splitpanes__pane { transition: none; }
.splitpanes--horizontal > .splitpanes__pane { transition: none; }
.splitpanes__splitter { touch-action: none; background-color: #3c3c3c; box-sizing: border-box; position: relative; flex-shrink: 0; }
.splitpanes--vertical > .splitpanes__splitter { width: 6px; min-width: 6px; cursor: col-resize; }
.splitpanes--horizontal > .splitpanes__splitter { height: 6px; min-height: 6px; cursor: row-resize; }
.splitpanes__splitter:hover { background-color: #007acc; }
.splitpanes, .splitpanes__pane { background: #1e1e1e; }

/* Monaco Editor 保护样式 */
.monaco-editor-container { position: relative; z-index: 1; isolation: isolate; }
.splitpanes--dragging .monaco-editor-container { pointer-events: none !important; }
.monaco-editor, .monaco-editor .margin, .monaco-editor .margin-view-overlays, .monaco-editor .view-overlays, .monaco-editor .glyph-margin { background-color: #1e1e1e !important; }
.monaco-editor .line-numbers { color: #858585 !important; background-color: #1e1e1e !important; }
.splitpanes--dragging .monaco-editor *::selection, .splitpanes--dragging .monaco-editor *::-moz-selection { background: transparent !important; }
.splitpanes--dragging * { user-select: none !important; -webkit-user-select: none !important; }
.splitpanes--dragging .monaco-editor *, .splitpanes--dragging .monaco-editor .view-line, .splitpanes--dragging .monaco-editor .view-lines { background: transparent !important; background-color: transparent !important; }
.splitpanes--dragging .monaco-editor .margin, .splitpanes--dragging .monaco-editor .margin-view-overlays { background-color: #1e1e1e !important; }
.splitpanes--dragging { -webkit-user-select: none !important; user-select: none !important; }
.splitpanes--dragging::selection, .splitpanes--dragging *::selection { background: transparent !important; color: inherit !important; }
.splitpanes--dragging::-moz-selection, .splitpanes--dragging *::-moz-selection { background: transparent !important; color: inherit !important; }
.splitpanes--dragging * { -webkit-tap-highlight-color: transparent !important; -webkit-touch-callout: none !important; }

.request-section, .response-section, .right-panel { height: 100%; }

.editor-wrapper { position: relative; width: 100%; height: 100%; }
.drag-overlay { position: absolute; top: 0; left: 0; right: 0; bottom: 0; background: transparent; z-index: 9999; pointer-events: none; }
.splitpanes--dragging .drag-overlay { pointer-events: auto !important; background: rgba(30, 30, 30, 0.01) !important; }

/* ServerSelector 在请求栏中的样式 */
.request-bar .server-selector { flex: 1; }

/* 折叠状态提示 */
.collapsed-hint { 
  flex: 1; 
  display: flex; 
  align-items: center; 
  justify-content: center; 
  color: #666; 
  font-size: 13px; 
  font-style: italic;
  background: #252526;
}

/* 空状态视图 */
.empty-state,
.empty-state-center {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #252526;
  border-radius: 4px;
  height: 100%;
}

.empty-state-content {
  text-align: center;
  max-width: 400px;
  padding: 40px;
}

.empty-state-icon {
  font-size: 64px;
  margin-bottom: 24px;
  opacity: 0.6;
}

.empty-state-title {
  font-size: 24px;
  font-weight: 500;
  color: #d4d4d4;
  margin-bottom: 12px;
}

.empty-state-desc {
  font-size: 14px;
  color: #888;
  margin-bottom: 24px;
  line-height: 1.5;
}

.empty-state-btn {
  padding: 12px 32px;
  background: #007acc;
  border: none;
  border-radius: 6px;
  color: white;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}

.empty-state-btn:hover {
  background: #0098ff;
}

.empty-state-hint {
  margin-top: 20px;
  font-size: 13px;
  color: #666;
}
</style>