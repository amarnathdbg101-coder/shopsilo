/**
 * ShopMe API Client (Axios Instance)
 * 
 * Hinglish Hint:
 * Yeh file backend se baat karne ka main gate hai. 
 * 1. Yeh har request me apne aap JWT token ('shopme_token') bhejta hai.
 * 2. Backend ke standard response format { success: true, data: ..., message: ... } ko parse karta hai.
 * 3. Agar token expire ho jaye ya 401 error aaye toh clean handling karta hai.
 */

import axios from 'axios';

// Backend server URL (Cleaned, supports production environment variables)
const rawUrl = (import.meta.env.VITE_API_BASE_URL || '').trim();
export const API_BASE_URL = rawUrl ? rawUrl.replace(/\/+$/, '') : 'http://localhost:8080';

const client = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 15000,
});

// Request Interceptor: Attach JWT Token automatically if user is logged in
client.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('shopme_token');
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response Interceptor: Standardize API responses & errors
client.interceptors.response.use(
  (response) => {
    // Backend returns { success: true, data: ..., message: ... }
    return response.data;
  },
  (error) => {
    // Extract error message returned by Go backend: { success: false, error: "..." }
    const backendMessage = error.response?.data?.error || error.response?.data?.message;
    const fallbackMessage = error.message || 'Kuch galat ho gaya, kripya dobara koshish karein';
    
    // Auto logout on 401 (Unauthorized) except when trying to login
    if (error.response?.status === 401 && !error.config.url.includes('/auth/login')) {
      localStorage.removeItem('shopme_token');
      localStorage.removeItem('shopme_user');
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
    }

    return Promise.reject(new Error(backendMessage || fallbackMessage));
  }
);

export default client;
