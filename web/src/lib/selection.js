import { writable } from 'svelte/store'

/** The plugin name currently selected for the reference panel, or null. */
export const selectedPlugin = writable(null)
