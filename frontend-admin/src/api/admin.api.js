import client from './client';

export const adminApi = {
  // Auth
  login: async (email, password) => {
    const res = await client.post('/auth/login', { email, password });
    return res.data; // { access_token, user }
  },

  // Dashboard Metrics
  getStats: async () => {
    const res = await client.get('/admin/stats');
    return res.data;
  },

  // Shop Management
  getShops: async (params = {}) => {
    const res = await client.get('/admin/shops', { params });
    return res.data; // { shops: [], total: number }
  },

  updateShopStatus: async (shopId, status, reason = '') => {
    const res = await client.patch(`/admin/shops/${shopId}/status`, { status, reason });
    return res.data;
  },

  banShop: async (shopId, { reason, blacklist_ip = true, blacklist_device = true, blacklist_images = true }) => {
    const res = await client.post(`/admin/shops/${shopId}/ban`, {
      reason,
      blacklist_ip,
      blacklist_device,
      blacklist_images,
    });
    return res.data;
  },

  // Reports & Grievance Desk
  getReports: async (params = {}) => {
    const res = await client.get('/admin/reports', { params });
    return res.data;
  },

  resolveReport: async (reportId, status, actionTaken) => {
    const res = await client.post(`/admin/reports/${reportId}/resolve`, {
      status,
      action_taken: actionTaken,
    });
    return res.data;
  },

  // Blacklist & Ban Evasion
  getBannedEntities: async (params = {}) => {
    const res = await client.get('/admin/banned-entities', { params });
    return res.data;
  },

  addBannedEntity: async ({ entity_type, entity_value, reason }) => {
    const res = await client.post('/admin/banned-entities', {
      entity_type,
      entity_value,
      reason,
    });
    return res.data;
  },

  // Categories Management
  getCategories: async () => {
    const res = await client.get('/admin/categories');
    return res.data; // array of categories
  },

  createCategory: async (payload) => {
    const res = await client.post('/admin/categories', payload);
    return res.data;
  },

  updateCategory: async (id, payload) => {
    const res = await client.put(`/admin/categories/${id}`, payload);
    return res.data;
  },

  deleteCategory: async (id) => {
    const res = await client.delete(`/admin/categories/${id}`);
    return res.data;
  },

  // User & Merchant Directory
  getUsers: async (params = {}) => {
    const res = await client.get('/admin/users', { params });
    return res.data; // { users: [], total: number }
  },

  updateUserStatus: async (userId, { is_active, role }) => {
    const res = await client.patch(`/admin/users/${userId}/status`, { is_active, role });
    return res.data;
  },
};

