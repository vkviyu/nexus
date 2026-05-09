declare module 'splitpanes' {
  import { DefineComponent } from 'vue'

  export const Splitpanes: DefineComponent<{
    horizontal?: boolean
    pushOtherPanes?: boolean
    dblClickSplitter?: boolean
    firstSplitter?: boolean
  }, {}, {}, {}, {}, {}, {}, {
    resize: (panes: Array<{ min: number; max: number; size: number }>) => void
    resized: (panes: Array<{ min: number; max: number; size: number }>) => void
    'pane-click': (pane: { index: number; min: number; max: number; size: number }) => void
    'pane-maximize': (pane: { index: number; min: number; max: number; size: number }) => void
    'pane-add': (pane: { index: number; min: number; max: number; size: number }) => void
    'pane-remove': (pane: { index: number; min: number; max: number; size: number }) => void
    'splitter-click': (splitter: { index: number; left: number; right: number }) => void
    ready: () => void
  }>

  export const Pane: DefineComponent<{
    size?: number | string
    minSize?: number | string
    maxSize?: number | string
  }>
}