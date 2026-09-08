/**
 * Language Context (Multi-language Support for Customer App: Hindi, English, Hinglish)
 * 
 * Hinglish Hint:
 * Grahak app ki bhasha badalne ke liye:
 * - Hindi (हिंदी)
 * - English
 * - Hinglish (बोलचाल - Mix)
 */

import React, { createContext, useContext, useState, useEffect } from 'react';

const LanguageContext = createContext();

export const TRANSLATIONS = {
  hi: {
    // Nav & General
    app_name: 'ShopMe ग्राहक बाज़ार',
    explore: 'दुकानें खोजें',
    deals: 'ऑफ़र्स व डील्स',
    saved: 'पसंदीदा',
    reservations: 'पिकअप टोकन',
    profile: 'प्रोफ़ाइल',
    login: 'लॉगिन करें',
    logout: 'लॉगआउट करें',
    search: 'सामान या दुकान खोजें...',
    near_you: 'आपके नज़दीक',
    distance: 'दूरी',
    open_now: 'खुली है',
    closed: 'बंद है',
    directions: 'रास्ता देखें',
    call: 'कॉल करें',
    whatsapp: 'व्हाट्सएप',
    
    // Product & Cart
    in_stock: 'स्टॉक में उपलब्ध',
    out_of_stock: 'स्टॉक समाप्त',
    low_stock: 'सीमित स्टॉक बाकी',
    view_details: 'पूरी जानकारी',
    reserve_now: 'काउंटर पिकअप होल्ड करें',
    price: 'मूल्य',
    mrp: 'MRP',
    save_amount: 'बचत',
    description: 'विवरण',
    shop: 'दुकान',
    category: 'श्रेणी',
    
    // Settings & Theme
    theme: 'थीम',
    dark_mode: 'डार्क मोड',
    light_mode: 'नॉर्मल (लाइट)',
    language: 'भाषा (Language)',
    settings: 'सेटिंग्स',
  },
  en: {
    // Nav & General
    app_name: 'ShopMe Hyperlocal Market',
    explore: 'Explore Shops',
    deals: 'Live Deals',
    saved: 'Saved Items',
    reservations: 'Pickups',
    profile: 'Profile',
    login: 'Login',
    logout: 'Logout',
    search: 'Search items or shops...',
    near_you: 'Near You',
    distance: 'Distance',
    open_now: 'Open Now',
    closed: 'Closed',
    directions: 'Directions',
    call: 'Call',
    whatsapp: 'WhatsApp',
    
    // Product & Cart
    in_stock: 'In Stock',
    out_of_stock: 'Out of Stock',
    low_stock: 'Limited Stock Left',
    view_details: 'View Details',
    reserve_now: 'Reserve for Pickup',
    price: 'Price',
    mrp: 'MRP',
    save_amount: 'You Save',
    description: 'Description',
    shop: 'Shop',
    category: 'Category',
    
    // Settings & Theme
    theme: 'Theme',
    dark_mode: 'Dark Mode',
    light_mode: 'Normal (Light)',
    language: 'Language',
    settings: 'Settings',
  },
  hinglish: {
    // Nav & General
    app_name: 'ShopMe Local Market',
    explore: 'Aas Paas Ki Dukaanein',
    deals: 'Offers & Discounts',
    saved: 'Saved / Favorite',
    reservations: 'Pickup Tokens',
    profile: 'Customer Profile',
    login: 'Login Karein',
    logout: 'Logout Karein',
    search: 'Saman ya dukan khojein...',
    near_you: 'Aapke Paas',
    distance: 'Doori',
    open_now: 'Khuli Hai',
    closed: 'Band Hai',
    directions: 'Rasta Dekhein',
    call: 'Call Karein',
    whatsapp: 'WhatsApp Karein',
    
    // Product & Cart
    in_stock: 'Stock Me Hai',
    out_of_stock: 'Stock Khatam',
    low_stock: 'Kam Maal Bacha Hai',
    view_details: 'Puri Jankari Dekhein',
    reserve_now: 'Counter Pickup Hold Karein',
    price: 'Price',
    mrp: 'MRP',
    save_amount: 'Aapki Bachat',
    description: 'Details',
    shop: 'Dukan',
    category: 'Category',
    
    // Settings & Theme
    theme: 'Theme Option',
    dark_mode: 'Dark Mode',
    light_mode: 'Normal (Light Mode)',
    language: 'Bhasha / Language',
    settings: 'Settings',
  },
};

export const LanguageProvider = ({ children }) => {
  const [language, setLanguageState] = useState(() => {
    return localStorage.getItem('shopme_language') || 'hi';
  });

  useEffect(() => {
    localStorage.setItem('shopme_language', language);
  }, [language]);

  const setLanguage = (lang) => {
    if (TRANSLATIONS[lang]) {
      setLanguageState(lang);
    }
  };

  const t = (key, fallback = '') => {
    return TRANSLATIONS[language]?.[key] || TRANSLATIONS.en?.[key] || fallback || key;
  };

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t, availableLanguages: Object.keys(TRANSLATIONS) }}>
      {children}
    </LanguageContext.Provider>
  );
};

export const useLanguage = () => {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error('useLanguage must be used within a LanguageProvider');
  }
  return context;
};
