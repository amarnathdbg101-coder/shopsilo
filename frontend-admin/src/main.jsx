import React from 'react';
import ReactDOM from 'react-dom/client';
import { AdminAuthProvider } from './context/AdminAuthContext';
import { AppContent } from './App';
import './index.css';

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <AdminAuthProvider>
      <AppContent />
    </AdminAuthProvider>
  </React.StrictMode>
);
