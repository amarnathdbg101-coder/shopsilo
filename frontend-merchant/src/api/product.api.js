/**
 * Product & Catalog API Service
 * 
 * Hinglish Hint:
 * Dukan ke samano (Products) ko manage karne ke liye:
 * - Products ki list lena
 * - Naya product add karna
 * - Barcode scan karke product dhundna (/products/scan/{code})
 * - Categories list lana
 */

import client from './client';

export const productApi = {
  // Public & Merchant: Shop ke products list karna
  listByShopSlug: async (slug, params = {}) => {
    const res = await client.get(`/shops/${slug}/products`, { params });
    return res.data;
  },

  // Public: Global products search
  listProducts: async (params = {}) => {
    const res = await client.get('/products', { params });
    return res.data;
  },
  listPublicProducts: async (params = {}) => {
    const res = await client.get('/products', { params });
    return res.data;
  },

  // Barcode / SKU Scan se product turant dhundna (POS Fast Billing ke liye)
  scanProduct: async (barcode) => {
    const res = await client.get(`/products/scan/${barcode}`);
    return res.data;
  },

  // Merchant: Naya product catalog me add karna
  createProduct: async (productData) => {
    const res = await client.post('/products', productData);
    return res.data;
  },

  // Merchant: Product edit karna
  updateProduct: async (id, updateData) => {
    const res = await client.put(`/products/${id}`, updateData);
    return res.data;
  },

  // Merchant: Product delete karna
  deleteProduct: async (id) => {
    const res = await client.delete(`/products/${id}`);
    return res.data;
  },

  // Public: Categories list
  getCategories: async () => {
    const res = await client.get('/categories');
    return res.data;
  },
};
