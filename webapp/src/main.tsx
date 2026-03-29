import React from 'react';
import ReactDOM from 'react-dom/client';
import '@fontsource/iosevka/400.css';
import '@fontsource/iosevka/700.css';
import { App } from './App';

const rootElement = document.getElementById('root');
if (!rootElement) {
  throw new Error('Missing #root element');
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
