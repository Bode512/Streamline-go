import './style.css'
import App from './App.svelte'
import { mount } from 'svelte'

const target = document.getElementById('app')
if (!target) {
  throw new Error('Streamline Desktop mount target was not found')
}

export default mount(App, { target })
