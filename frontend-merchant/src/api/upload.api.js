/**
 * Cloud Image Upload API Service
 * 
 * Hinglish Hint:
 * Cloudflare R2 image storage service:
 * 1. User Avatar upload (/user/avatar)
 * 2. Shop Logo aur Promotional Banners upload (/shops/me/images)
 * 3. Product Catalog Images upload (/products/images - Max 4 images)
 */

import client from './client';

export const uploadApi = {
  // 1. User Avatar Upload
  uploadUserAvatar: async (file) => {
    const formData = new FormData();
    formData.append('avatar', file);
    const res = await client.post('/user/avatar', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    const data = res?.data || res;
    return {
      avatar_url: data?.avatar_url || (typeof data === 'string' ? data : ''),
    };
  },

  // 2. Shop Logo & Banners Upload
  uploadShopImages: async ({ logo, banners = [] }) => {
    const formData = new FormData();
    if (logo) {
      formData.append('logo', logo);
    }
    if (banners && banners.length > 0) {
      banners.slice(0, 2).forEach((b) => {
        formData.append('banners', b);
      });
    }
    const res = await client.post('/shops/me/images', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    const data = res?.data || res;
    return {
      logo_url: data?.logo_url || '',
      banners: Array.isArray(data?.banners) ? data.banners : [],
    };
  },

  // 3. Product Images Upload (Up to 4 images)
  uploadProductImages: async (files) => {
    const formData = new FormData();
    const fileList = Array.from(files || []).slice(0, 4);
    fileList.forEach((file) => {
      formData.append('images', file);
    });
    const res = await client.post('/products/images', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    const data = res?.data || res;
    const images = Array.isArray(data?.images)
      ? data.images
      : Array.isArray(data)
      ? data
      : [];
    return { images };
  },
};
