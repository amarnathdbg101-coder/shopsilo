/**
 * Analytics & Profit Intelligence API Service
 * 
 * Hinglish Hint:
 * Dukan ki asal kamayi aur munafa (Profit Intelligence):
 * - Monthly Gross Sales vs Total Expenses vs Real "Pocket Profit"
 * - Sabse zyada bikne wale vs Dead-stock products ki matrix
 */

import client from './client';

export const analyticsApi = {
  // Real Net Pocket Profit summary
  getMonthlyProfit: async () => {
    const res = await client.get('/shops/me/analytics/profit');
    return res.data;
  },

  // Best profitable vs Dead stock products matrix
  getProductMatrix: async () => {
    const res = await client.get('/shops/me/analytics/products');
    return res.data;
  },
};
