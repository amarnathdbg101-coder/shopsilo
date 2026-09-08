/**
 * Customer Khata (Udhar / Credit Book) API Service
 * 
 * Hinglish Hint:
 * Grahako ke udhar aur jama (payments) ka pura hisab-kitab:
 * - Market me total kitna udhar baki hai (/shops/me/khata/summary)
 * - Sare udhar wale grahako ki list (/shops/me/khata)
 * - Kisi specific grahak ki passbook aur len-den history
 * - Naya udhar likhna (Record Credit)
 * - Pura ya aadha paisa aane par jama karna (Record Payment)
 */

import client from './client';

export const khataApi = {
  // Total market udhar aur count summary
  getSummary: async () => {
    const res = await client.get('/shops/me/khata/summary');
    return res.data;
  },

  // Udhar wale grahako ki list (search by name/mobile supported)
  listCustomers: async (search = '') => {
    const params = search ? { search } : {};
    const res = await client.get('/shops/me/khata', { params });
    return res.data;
  },

  // Kisi grahak ki complete passbook / transaction history
  getCustomerHistory: async (mobile) => {
    const res = await client.get(`/shops/me/khata/${mobile}`);
    return res.data; // { customer, transactions }
  },

  // Naya udhar likhna (Grahak ko udhar samaan diya)
  // payload: { customer_name, customer_mobile, amount, notes, bill_number }
  recordCredit: async (creditData) => {
    const res = await client.post('/shops/me/khata', creditData);
    return res.data;
  },

  // Grahak ne paise jama kiye (Settlement / Payment received)
  // payload: { customer_name, amount, payment_mode: 'cash'|'upi'|'card'|'other', notes }
  recordPayment: async (mobile, paymentData) => {
    const res = await client.post(`/shops/me/khata/${mobile}/payment`, paymentData);
    return res.data;
  },
};
