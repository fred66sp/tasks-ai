import './styles/tokens.css';
import './styles/layout.css';
import './styles/components.css';
import './styles/responsive.css';
import { AppShell } from './app-shell.js';
import { TkIcon } from './components/icon.js';

const app = globalThis.Vue.createApp(AppShell);
app.component('TkIcon', TkIcon);
app.mount('#app');
