/**
 * Inventory & Stock Alerts API Service
 * 
 * Hinglish Hint:
 * Dukan ka stock control aur alerts:
 * - Stock kam ya zyada adjust karna (Stock In / Stock Out)
 * - Low-stock alerts (jin samano ka stock khatam hone wala hai)
 * - Wholesale vendor ke liye 1-Click PDF reorder sheet link
 */

import client, { API_BASE_URL } from './client';

export const inventoryApi = {
  // Stock adjust karna (+50 naya maal aaya ya -5 damage hua)
  // payload: { product_id, adjustment: number, low_stock_threshold: number, notes: string }
  adjustStock: async (adjustmentData) => {
    const res = await client.post('/shops/me/inventory/adjust', adjustmentData);
    return res.data;
  },

  // Khatam hone wale products ki list
  getLowStockAlerts: async () => {
    const res = await client.get('/shops/me/inventory/low-stock');
    return res.data; // { total_low_stock_items, items: [...] }
  },

  // Wholesale Reorder Sheet PDF download URL
  getReorderSheetUrl: () => {
    return `${API_BASE_URL}/shops/me/inventory/reorder-sheet.pdf`;
  },
};
