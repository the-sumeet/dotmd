import { mount } from 'svelte'
// Loads the Wails runtime, which handles `--wails-draggable` window dragging
// and the frontend <-> Go bridge.
import '@wailsio/runtime'
import './app.css'
import App from './App.svelte'
import { applyStoredTheme } from './lib/settings.svelte'

// index.html already set light or dark; this keeps "System" following the OS.
applyStoredTheme()

mount(App, { target: document.getElementById('app')! })
