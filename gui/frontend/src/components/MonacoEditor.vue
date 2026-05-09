<template>
  <div ref="editorContainer" class="monaco-editor-container"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'

// 配置 Monaco Worker
self.MonacoEnvironment = {
  getWorker(_: unknown, label: string) {
    if (label === 'json') {
      return new jsonWorker()
    }
    return new editorWorker()
  }
}

const props = defineProps<{
  modelValue: string
  language?: string
  readOnly?: boolean
  theme?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'validate', markers: monaco.editor.IMarker[]): void
}>()

const editorContainer = ref<HTMLElement>()
let editor: monaco.editor.IStandaloneCodeEditor | null = null
let handleResize: (() => void) | null = null

onMounted(() => {
  if (!editorContainer.value) return

  // 确保容器有明确尺寸
  editorContainer.value.style.width = '100%'
  
  editor = monaco.editor.create(editorContainer.value, {
    value: props.modelValue,
    language: props.language || 'json',
    theme: props.theme || 'vs-dark',
    readOnly: props.readOnly || false,
    automaticLayout: true,
    minimap: { enabled: false },
    fontSize: 14,
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    formatOnPaste: true,
    formatOnType: true,
    tabSize: 2,
    folding: true,
    foldingStrategy: 'indentation',
    bracketPairColorization: { enabled: true },
  })

  // 内容变化时通知父组件
  editor.onDidChangeModelContent(() => {
    const value = editor?.getValue() || ''
    emit('update:modelValue', value)
  })

  // 监听验证结果
  monaco.editor.onDidChangeMarkers((uris) => {
    const model = editor?.getModel()
    if (model && uris.some(uri => uri.toString() === model.uri.toString())) {
      const markers = monaco.editor.getModelMarkers({ resource: model.uri })
      emit('validate', markers)
    }
  })

  // 添加快捷键：Cmd+S / Ctrl+S 格式化文档
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    editor?.getAction('editor.action.formatDocument')?.run()
  })

  // 延迟触发布局重算，解决容器尺寸计算问题
  setTimeout(() => editor?.layout(), 100)

  // 监听窗口变化，重新计算布局
  handleResize = () => {
    editor?.layout()
  }
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  if (handleResize) {
    window.removeEventListener('resize', handleResize)
  }
  editor?.dispose()
})

// 监听外部值变化
watch(() => props.modelValue, (newValue) => {
  if (editor && editor.getValue() !== newValue) {
    editor.setValue(newValue)
  }
})

// 监听语言变化，动态更新编辑器语言
watch(() => props.language, (newLanguage) => {
  if (editor && newLanguage) {
    const model = editor.getModel()
    if (model) {
      monaco.editor.setModelLanguage(model, newLanguage)
    }
  }
})

// 格式化方法
const format = () => {
  editor?.getAction('editor.action.formatDocument')?.run()
}

// 手动触发布局更新（用于 v-show 切换后）
const layout = () => {
  editor?.layout()
}

defineExpose({ format, layout })
</script>

<style scoped>
.monaco-editor-container {
  width: 100%;
  height: 100%;
  min-height: 200px;
  position: relative;
}
</style>