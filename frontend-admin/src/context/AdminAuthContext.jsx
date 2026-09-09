import React, { createContext, useContext, useState, useEffect } from 'react';
import { adminApi } from '../api/admin.api';

const AdminAuthContext = createContext(null);

export const AdminAuthProvider = ({ children }) => {
  const [adminUser, setAdminUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('shopme_admin_token') || null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const savedUser = localStorage.getItem('shopme_admin_user');
    if (savedUser && token) {
      try {
        const parsed = JSON.parse(savedUser);
        if (parsed.role === 'admin') {
          setAdminUser(parsed);
        } else {
          logout();
        }
      } catch (e) {
        logout();
      }
    }
    setLoading(false);
  }, [token]);

  const login = async (email, password) => {
    const res = await adminApi.login(email, password);
    if (res.user?.role !== 'admin') {
      throw new Error('Access denied: Only users with Administrator role can access this console.');
    }

    setToken(res.access_token);
    setAdminUser(res.user);
    localStorage.setItem('shopme_admin_token', res.access_token);
    localStorage.setItem('shopme_admin_user', JSON.stringify(res.user));
    return res.user;
  };

  const logout = () => {
    setToken(null);
    setAdminUser(null);
    localStorage.removeItem('shopme_admin_token');
    localStorage.removeItem('shopme_admin_user');
  };

  return (
    <AdminAuthContext.Provider value={{ adminUser, token, loading, login, logout }}>
      {children}
    </AdminAuthContext.Provider>
  );
};

export const useAdminAuth = () => {
  const context = useContext(AdminAuthContext);
  if (!context) {
    throw new Error('useAdminAuth must be used within an AdminAuthProvider');
  }
  return context;
};
