/**
 * Counter POS (Point of Sale) API Service
 * 
 * Hinglish Hint:
 * Dukan ke counter billing terminal ke liye:
 * - Counter par naya bill/sale create karna (Cash, UPI, Khata/Credit)
 * - Aaj ki din bhar ki bikri (Daily Summary) dekhna
 * - Digital PDF receipt ka URL generate karna
 */

import client, { API_BASE_URL } from './client';

export const posApi = {
  // Counter Sale create karna
  // payload: { customer_phone, items: [{ product_id, quantity, custom_price }], discount_amount, payment_method }
  createSale: async (saleData) => {
    const res = await client.post('/shops/me/pos/sale', saleData);
    return res.data; // { bill, receipt_url, loyalty_points_credited }
  },

  // Aaj ki bikri ki summary (Total Bills, Cash Sales, UPI Sales, Credit Sales)
  getDailySummary: async (date) => {
    const params = date ? { date } : {};
    const res = await client.get('/shops/me/pos/daily-summary', { params });
    return res.data;
  },

  // Public Digital Receipt URL builder
  getReceiptUrl: (billNumber) => {
    return `${API_BASE_URL}/receipts/${billNumber}`;
  },
};
