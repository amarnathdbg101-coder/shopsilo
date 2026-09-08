/**
 * POS (Counter Billing) Cart Context
 * 
 * Hinglish Hint:
 * Dukan ke counter par grahak ke jhole/thele (Cart) ko manage karta hai:
 * - Product add/remove karna
 * - Quantity badhana/ghatana (+ / -)
 * - Discount lagana
 * - Total bill amount calculate karna
 * - Payment method select karna ('cash', 'upi', 'credit')
 */

import React, { createContext, useContext, useState, useMemo } from 'react';

const POSContext = createContext(null);

export const POSProvider = ({ children }) => {
  // Cart items array: [{ product, quantity, customPrice }]
  const [cart, setCart] = useState([]);
  const [customerPhone, setCustomerPhone] = useState('');
  const [discountAmount, setDiscountAmount] = useState(0);
  const [paymentMethod, setPaymentMethod] = useState('cash'); // 'cash' | 'upi' | 'credit'

  // Cart me product add karna ya quantity increment karna
  const addToCart = (product, quantity = 1, customPrice = null) => {
    setCart((prev) => {
      const existingIndex = prev.findIndex((item) => item.product.id === product.id);
      if (existingIndex > -1) {
        const updated = [...prev];
        updated[existingIndex].quantity += quantity;
        return updated;
      }
      return [...prev, { product, quantity, customPrice: customPrice ?? product.price }];
    });
  };

  // Quantity update (+1 ya -1)
  const updateQuantity = (productId, newQuantity) => {
    if (newQuantity <= 0) {
      removeFromCart(productId);
      return;
    }
    setCart((prev) =>
      prev.map((item) =>
        item.product.id === productId ? { ...item, quantity: newQuantity } : item
      )
    );
  };

  // Cart se item nikalna
  const removeFromCart = (productId) => {
    setCart((prev) => prev.filter((item) => item.product.id !== productId));
  };

  // Cart khali karna (bill banne ke baad)
  const clearCart = () => {
    setCart([]);
    setCustomerPhone('');
    setDiscountAmount(0);
    setPaymentMethod('cash');
  };

  // Bill calculations
  const subtotal = useMemo(() => {
    return cart.reduce((sum, item) => {
      const price = item.customPrice !== null ? item.customPrice : item.product.price;
      return sum + price * item.quantity;
    }, 0);
  }, [cart]);

  const total = useMemo(() => {
    const finalAmt = subtotal - (Number(discountAmount) || 0);
    return Math.max(0, finalAmt);
  }, [subtotal, discountAmount]);

  const itemCount = useMemo(() => {
    return cart.reduce((count, item) => count + item.quantity, 0);
  }, [cart]);

  return (
    <POSContext.Provider
      value={{
        cart,
        customerPhone,
        setCustomerPhone,
        discountAmount,
        setDiscountAmount,
        paymentMethod,
        setPaymentMethod,
        addToCart,
        updateQuantity,
        removeFromCart,
        clearCart,
        subtotal,
        total,
        itemCount,
      }}
    >
      {children}
    </POSContext.Provider>
  );
};

export const usePOS = () => {
  const context = useContext(POSContext);
  if (!context) {
    throw new Error('usePOS must be used within a POSProvider');
  }
  return context;
};
