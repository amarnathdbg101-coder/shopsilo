/**
 * Shop Expenses (Dukan ke Roz ke Kharche) API Service
 * 
 * Hinglish Hint:
 * Dukan ke daily kharche log karne ke liye:
 * - Kharche add karna (Chai-nashta, Bijli bill, Dukan rent, Staff salary, Packaging)
 * - Mahine bhar ke kharche list karna summary totals ke sath
 * - Galat entry ko delete karna
 */

import client from './client';

export const expenseApi = {
  // Dukan ka naya kharcha add karna
  // payload: { category, amount, notes, payment_method, expense_date }
  createExpense: async (expenseData) => {
    const res = await client.post('/shops/me/expenses', expenseData);
    return res.data;
  },

  // Mahine ya date range ke hisab se kharche fetch karna
  // params: { month, year }
  listExpenses: async (params = {}) => {
    const res = await client.get('/shops/me/expenses', { params });
    return res.data; // { expenses, total_amount, category_total, month, year }
  },

  // Kharcha delete karna
  deleteExpense: async (id) => {
    const res = await client.delete(`/shops/me/expenses/${id}`);
    return res.data;
  },
};
