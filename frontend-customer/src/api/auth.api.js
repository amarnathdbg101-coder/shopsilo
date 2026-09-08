/**
 * Authentication API Service
 * 
 * Hinglish Hint:
 * Backend ke /auth routes se baat karta hai:
 * - Login (Dukaandar ya Customer)
 * - Register (Naya account banana)
 * - Forgot/Reset Password
 */

import client from './client';

export const authApi = {
  // User login (returns access_token, user object)
  login: async (email, password) => {
    const res = await client.post('/auth/login', { email, password });
    return res.data; // { access_token, token_type, expires_in, user }
  },

  // New user registration
  register: async ({ full_name, email, password, phone }) => {
    const res = await client.post('/auth/register', {
      full_name,
      email,
      password,
      phone,
    });
    return res.data; // { user, access_token }
  },

  // Request password reset token via email
  forgotPassword: async (email) => {
    const res = await client.post('/auth/forgot-password', { email });
    return res.data;
  },

  // Reset password using token
  resetPassword: async (token, newPassword) => {
    const res = await client.post('/auth/reset-password', {
      token,
      new_password: newPassword,
    });
    return res.data;
  },
};
