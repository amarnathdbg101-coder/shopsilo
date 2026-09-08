import client from './client';

export const offersApi = {
  // Merchant: Create a new promotional deal
  createOffer: async (offerData) => {
    const res = await client.post('/shops/me/offers', offerData);
    return res.data;
  },

  // Merchant / Public: Get offers for current shop
  getShopOffers: async (slug) => {
    const res = await client.get(`/shops/${slug}/offers`);
    return res.data;
  },
};
