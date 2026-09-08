import React, { createContext, useContext, useState, useEffect } from 'react';

const SavedContext = createContext(null);

export const SavedProvider = ({ children }) => {
  const [savedProducts, setSavedProducts] = useState(() => {
    try {
      const saved = localStorage.getItem('shopme_saved_products');
      return saved ? JSON.parse(saved) : [];
    } catch (e) {
      return [];
    }
  });

  const [savedShops, setSavedShops] = useState(() => {
    try {
      const saved = localStorage.getItem('shopme_saved_shops');
      return saved ? JSON.parse(saved) : [];
    } catch (e) {
      return [];
    }
  });

  useEffect(() => {
    localStorage.setItem('shopme_saved_products', JSON.stringify(savedProducts));
  }, [savedProducts]);

  useEffect(() => {
    localStorage.setItem('shopme_saved_shops', JSON.stringify(savedShops));
  }, [savedShops]);

  const toggleSaveProduct = (product) => {
    if (!product || !product.id) return;
    setSavedProducts((prev) => {
      const exists = prev.some((p) => p.id === product.id);
      if (exists) {
        return prev.filter((p) => p.id !== product.id);
      } else {
        return [...prev, product];
      }
    });
  };

  const isProductSaved = (productId) => {
    return savedProducts.some((p) => p.id === productId);
  };

  const toggleSaveShop = (shop) => {
    if (!shop || !shop.id) return;
    setSavedShops((prev) => {
      const exists = prev.some((s) => s.id === shop.id);
      if (exists) {
        return prev.filter((s) => s.id !== shop.id);
      } else {
        return [...prev, shop];
      }
    });
  };

  const isShopSaved = (shopId) => {
    return savedShops.some((s) => s.id === shopId);
  };

  return (
    <SavedContext.Provider
      value={{
        savedProducts,
        savedShops,
        toggleSaveProduct,
        isProductSaved,
        toggleSaveShop,
        isShopSaved,
      }}
    >
      {children}
    </SavedContext.Provider>
  );
};

export const useSaved = () => {
  const context = useContext(SavedContext);
  if (!context) {
    throw new Error('useSaved must be used within a SavedProvider');
  }
  return context;
};
