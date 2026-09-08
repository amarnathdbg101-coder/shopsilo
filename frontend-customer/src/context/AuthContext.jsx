/**
 * Authentication Context
 * 
 * Hinglish Hint:
 * Yeh pure app me user ke login status, JWT token aur uski dukan (shop)
 * ki details ko track karta hai.
 * Kisi bhi screen me useAuth() call karke pata chal jata hai ki user login hai ya nahi.
 */

import React, { createContext, useContext, useState, useEffect } from 'react';
import { authApi } from '../api/auth.api';
import { shopApi } from '../api/shop.api';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [shop, setShop] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('shopme_token') || null);
  const [loading, setLoading] = useState(true);

  // App shuru hote hi saved session check karna
  useEffect(() => {
    const savedUser = localStorage.getItem('shopme_user');
    if (savedUser && token) {
      try {
        const parsedUser = JSON.parse(savedUser);
        setUser(parsedUser);
        // Agar user dukaandar hai, toh uski shop fetch kar lo
        if (parsedUser.role === 'shop' || parsedUser.role === 'admin') {
          fetchShopDetails();
        }
      } catch (e) {
        console.error('Session restore failed:', e);
        logout();
      }
    }
    setLoading(false);
  }, [token]);

  // Merchant ki dukan ki details load karna
  const fetchShopDetails = async () => {
    try {
      const shopData = await shopApi.getMyShop();
      setShop(shopData);
    } catch (err) {
      // Ho sakta hai user ne abhi shop register na ki ho
      setShop(null);
    }
  };

  // Login handler
  const login = async (email, password) => {
    const res = await authApi.login(email, password);
    // res: { access_token, user }
    setToken(res.access_token);
    setUser(res.user);
    localStorage.setItem('shopme_token', res.access_token);
    localStorage.setItem('shopme_user', JSON.stringify(res.user));

    if (res.user?.role === 'shop' || res.user?.role === 'admin') {
      try {
        const shopData = await shopApi.getMyShop();
        setShop(shopData);
      } catch (e) {
        setShop(null);
      }
    }
    return res;
  };

  // Register handler
  const register = async (userData) => {
    const res = await authApi.register(userData);
    if (res.access_token) {
      setToken(res.access_token);
      setUser(res.user);
      localStorage.setItem('shopme_token', res.access_token);
      localStorage.setItem('shopme_user', JSON.stringify(res.user));
    }
    return res;
  };

  // Logout handler
  const logout = () => {
    setToken(null);
    setUser(null);
    setShop(null);
    localStorage.removeItem('shopme_token');
    localStorage.removeItem('shopme_user');
  };

  // Update local user state (e.g. new avatar)
  const updateUser = (updatedFields) => {
    setUser((prev) => {
      const updated = { ...prev, ...updatedFields };
      localStorage.setItem('shopme_user', JSON.stringify(updated));
      return updated;
    });
  };

  // Update local shop state (e.g. new logo or banners)
  const updateShopState = (updatedFields) => {
    setShop((prev) => (prev ? { ...prev, ...updatedFields } : updatedFields));
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        shop,
        token,
        loading,
        isAuthenticated: !!token,
        isMerchant: user?.role === 'shop' || user?.role === 'admin',
        login,
        register,
        logout,
        updateUser,
        updateShopState,
        refreshShop: fetchShopDetails,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook to use Auth easily
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
