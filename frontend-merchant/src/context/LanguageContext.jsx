/**
 * Language Context (Multi-language Support: Hindi, English, Hinglish)
 * 
 * Hinglish Hint:
 * App ki bhasha badalne ke liye:
 * - Hindi (हिंदी)
 * - English
 * - Hinglish (बोलचाल - Mix)
 */

import React, { createContext, useContext, useState, useEffect } from 'react';

const LanguageContext = createContext();

export const TRANSLATIONS = {
  hi: {
    // Nav & General
    app_name: 'ShopMe Dukan OS',
    dashboard: 'डैशबोर्ड',
    billing_pos: 'फास्ट POS बिलिंग',
    inventory: 'स्टॉक व कैटलॉग',
    khata: 'ग्राहक खाता',
    profit: 'मुनाफा व बिक्री',
    expenses: 'दुकान के खर्चे',
    pickups: 'काउंटर पिकअप',
    profile: 'दुकान सेटिंग्स व प्रोफाइल',
    logout: 'लॉगआउट करें',
    login: 'लॉगिन करें',
    online: 'ऑनलाइन (खुली है)',
    offline: 'ऑफलाइन (बंद है)',
    save: 'सेव करें',
    cancel: 'रद्द करें',
    delete: 'हटाएं',
    edit: 'एडिट करें',
    search: 'खोजें...',
    details: 'पूरी जानकारी',
    theme: 'थीम',
    dark_mode: 'डार्क मोड',
    light_mode: 'नॉर्मल (लाइट)',
    language: 'भाषा (Language)',
    settings: 'सेटिंग्स',
    
    // Inventory & Stock
    add_product: 'नया प्रोडक्ट जोड़ें',
    edit_product: 'प्रोडक्ट अपडेट करें',
    product_name: 'सामान का नाम',
    selling_price: 'बिक्री मूल्य (₹)',
    cost_price: 'खरीद मूल्य (₹)',
    stock_qty: 'स्टॉक मात्रा',
    min_stock: 'न्यूनतम स्टॉक (अलर्ट सीमा)',
    min_stock_hint: 'जब स्टॉक इससे कम होगा, तो अलर्ट दिखेगा (डिफ़ॉल्ट: 1)',
    low_stock_alert: 'कम स्टॉक चेतावनी',
    bulk_restock: '1-क्लिक बल्क रीस्टॉक (+10)',
    all_products: 'सारे प्रोडक्ट्स',
    low_stock: 'कम स्टॉक',
    stock_value: 'कुल स्टॉक वैल्यू',
    
    // Shop & Timing
    shop_info: 'दुकान की जानकारी',
    opening_time: 'खुलने का समय',
    closing_time: 'बंद होने का समय',
    weekly_off: 'साप्ताहिक छुट्टी',
    address: 'दुकान का पता',
    city: 'शहर',
    pincode: 'पिनकोड',
    phone: 'फ़ोन नंबर',
    whatsapp: 'व्हाट्सएप',
  },
  en: {
    // Nav & General
    app_name: 'ShopMe Merchant OS',
    dashboard: 'Dashboard',
    billing_pos: 'Fast POS Billing',
    inventory: 'Stock & Catalog',
    khata: 'Customer Ledger',
    profit: 'Profit & Analytics',
    expenses: 'Shop Expenses',
    pickups: 'Counter Pickups',
    profile: 'Shop Settings & Profile',
    logout: 'Logout',
    login: 'Login',
    online: 'Online (Open)',
    offline: 'Offline (Closed)',
    save: 'Save Changes',
    cancel: 'Cancel',
    delete: 'Delete',
    edit: 'Edit',
    search: 'Search...',
    details: 'Full Details',
    theme: 'Theme',
    dark_mode: 'Dark Mode',
    light_mode: 'Normal (Light)',
    language: 'Language',
    settings: 'Settings',

    // Inventory & Stock
    add_product: 'Add New Product',
    edit_product: 'Edit Product',
    product_name: 'Product Name',
    selling_price: 'Selling Price (₹)',
    cost_price: 'Cost Price (₹)',
    stock_qty: 'Stock Quantity',
    min_stock: 'Minimum Stock (Alert Limit)',
    min_stock_hint: 'Triggers low stock alert when inventory reaches this level (Default: 1)',
    low_stock_alert: 'Low Stock Alert',
    bulk_restock: '1-Click Bulk Restock (+10)',
    all_products: 'All Products',
    low_stock: 'Low Stock',
    stock_value: 'Total Stock Retail Value',

    // Shop & Timing
    shop_info: 'Shop Information',
    opening_time: 'Opening Time',
    closing_time: 'Closing Time',
    weekly_off: 'Weekly Off',
    address: 'Shop Address',
    city: 'City',
    pincode: 'Pincode',
    phone: 'Phone Number',
    whatsapp: 'WhatsApp',
  },
  hinglish: {
    // Nav & General
    app_name: 'ShopMe Dukan OS',
    dashboard: 'Dashboard Overview',
    billing_pos: 'Fast POS Billing',
    inventory: 'Stock & Catalog',
    khata: 'Customer Khata',
    profit: 'Munafa & Sales',
    expenses: 'Dukan Ke Roz Ke Kharche',
    pickups: 'Counter Pickup Verify',
    profile: 'Shop Settings & Profile',
    logout: 'Dukan OS Se Logout Karein',
    login: 'Merchant Login',
    online: 'Online (Khuli Hai)',
    offline: 'Offline (Band Hai)',
    save: 'Save Karein',
    cancel: 'Cancel Karein',
    delete: 'Delete Karein',
    edit: 'Edit Karein',
    search: 'Khojein...',
    details: 'Puri Jankari Dekhein',
    theme: 'Theme Option',
    dark_mode: 'Dark Mode',
    light_mode: 'Normal (Light Mode)',
    language: 'Bhasha / Language',
    settings: 'Settings',

    // Inventory & Stock
    add_product: 'Naya Product Jodein',
    edit_product: 'Product Update Karein',
    product_name: 'Saman Ka Naam',
    selling_price: 'Bikri Price (₹)',
    cost_price: 'Kharid Price (₹)',
    stock_qty: 'Stock Units',
    min_stock: 'Minimum Stock Limit',
    min_stock_hint: 'Jab maal is sankhya ya isse kam bachega toh alert aayega (Default: 1)',
    low_stock_alert: 'Kam Stock Alert',
    bulk_restock: '1-Click Bulk Restock (+10)',
    all_products: 'Sare Products',
    low_stock: 'Kam Stock',
    stock_value: 'Stock Retail Value',

    // Shop & Timing
    shop_info: 'Dukan Ki Details',
    opening_time: 'Khulne Ka Time',
    closing_time: 'Band Hone Ka Time',
    weekly_off: 'Weekly Chhutti',
    address: 'Dukan Ka Pata',
    city: 'Shahar',
    pincode: 'Pincode',
    phone: 'Phone Number',
    whatsapp: 'WhatsApp',
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
