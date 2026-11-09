import { mount } from 'svelte'
import '@fontsource-variable/inter'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'
import './lib/styles/global.css'
import App from './App.svelte'

const app = mount(App, { target: document.getElementById('app')! })

export default app