/**
 * Shop Management API Service
 * 
 * Hinglish Hint:
 * Dukan ki profile aur settings handle karta hai:
 * - Apni dukan ka data lena (/shops/me)
 * - Nayi dukan create karna (/shops)
 * - Dukan status toggle (Online/Offline)
 * - Shop QR code download/view
 * - Public marketplace me shops browse karna
 */

import client from './client';

export const shopApi = {
  // Merchant: Apni shop ki details lana
  getMyShop: async () => {
    const res = await client.get('/shops/me');
    return res.data;
  },

  // Merchant: Nayi shop register karna
  createShop: async (shopData) => {
    const res = await client.post('/shops', shopData);
    return res.data;
  },

  // Merchant: Shop details update karna (name, address, etc.)
  updateMyShop: async (updateData) => {
    const res = await client.put('/shops/me', updateData);
    return res.data;
  },

  // Merchant: Dukan ko live (Open) ya close (Offline) karna
  toggleShopStatus: async () => {
    const res = await client.patch('/shops/me/status');
    return res.data;
  },

  // Merchant: Apni dukan ka payment / storefront QR Code lena
  getMyShopQR: async () => {
    const res = await client.get('/shops/me/qr');
    return res.data;
  },

  // Public: Marketplace ke sare active shops dekhna
  listPublicShops: async (params = {}) => {
    const res = await client.get('/shops', { params });
    return res.data;
  },

  // Public: Slug se kisi bhi dukan ka storefront dekhna (e.g. /shops/slug/gupta-general-store)
  getShopBySlug: async (slug) => {
    const res = await client.get(`/shops/slug/${slug}`);
    return res.data;
  },
};
