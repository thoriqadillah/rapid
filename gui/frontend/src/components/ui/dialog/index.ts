export { default as Dialog } from './Dialog.vue'
export { default as DialogTrigger } from './DialogTrigger.vue'
export { default as DialogHeader } from './DialogHeader.vue'
export { default as DialogTitle } from './DialogTitle.vue'
export { default as DialogDescription } from './DialogDescription.vue'
export { default as DialogContent } from './DialogContent.vue'
export { default as DialogFooter } from './DialogFooter.vue'

export type Action = () => void
export interface Actionable {
    type: 'destructive' | 'warning' | 'default'
    label: string
    immediatelyClose?: boolean
    action: Action
}