import client from './client';

export const dealsApi = {
  // Public: Get all active deals across shops
  listDeals: async (params = {}) => {
    const res = await client.get('/deals', { params });
    return res.data;
  },

  // Public: Get offers for a specific shop
  getShopOffers: async (slug) => {
    const res = await client.get(`/shops/${slug}/offers`);
    return res.data;
  },
};
