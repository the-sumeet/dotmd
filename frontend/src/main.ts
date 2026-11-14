import { mount } from 'svelte'
// Loads the Wails runtime, which handles `--wails-draggable` window dragging
// and the frontend <-> Go bridge.
import '@wailsio/runtime'
import './app.css'
import App from './App.svelte'

mount(App, { target: document.getElementById('app')! })
