/**
 * Counter POS (Point of Sale) Screen
 * 
 * Hinglish Hint:
 * Dukaandar ke counter par fast billing ke liye:
 * 1. Barcode scanner ya product search se turant item cart me add hota hai.
 * 2. Quantity adjust hoti hai (+ / -).
 * 3. Payment Mode select hota hai:
 *    - 'cash' (Nakad)
 *    - 'upi' (Online UPI)
 *    - 'credit' (Seedhe Khata me udhar chala jayega!)
 * 4. Bill banne ke baad turant Digital Receipt PDF dekhne/print karne ka option milta hai.
 */

import React, { useState, useEffect } from 'react';
import {
  Search,
  Barcode,
  Plus,
  Minus,
  Trash2,
  Receipt,
  CheckCircle,
  AlertCircle,
  Printer,
  X,
} from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { usePOS } from '../../context/POSContext';
import { productApi } from '../../api/product.api';
import { posApi } from '../../api/pos.api';
import { AppLayout } from '../../components/layout/AppLayout';
import { getImageUrl } from '../../utils/imageUrl';

export const POSScreen = () => {
  const { shop } = useAuth();
  const {
    cart,
    addToCart,
    updateQuantity,
    removeFromCart,
    clearCart,
    subtotal,
    total,
    discountAmount,
    setDiscountAmount,
    customerPhone,
    setCustomerPhone,
    paymentMethod,
    setPaymentMethod,
    itemCount,
  } = usePOS();

  const [products, setProducts] = useState([]);
  const [loadingProducts, setLoadingProducts] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [barcodeInput, setBarcodeInput] = useState('');
  const [billingLoading, setBillingLoading] = useState(false);
  const [billSuccess, setBillSuccess] = useState(null);
  const [errorMessage, setErrorMessage] = useState('');

  // Shop ke products load karna
  useEffect(() => {
    if (shop?.slug) {
      loadProducts();
    }
  }, [shop]);

  const loadProducts = async () => {
    try {
      setLoadingProducts(true);
      const data = await productApi.listByShopSlug(shop.slug);
      // Agar backend pagination format deta hai
      setProducts(data.products || data || []);
    } catch (err) {
      console.error('Failed to load products:', err);
    } finally {
      setLoadingProducts(false);
    }
  };

  // Barcode enter dabane par fast scan
  const handleBarcodeSubmit = async (e) => {
    e.preventDefault();
    if (!barcodeInput.trim()) return;

    try {
      setErrorMessage('');
      const scannedProduct = await productApi.scanProduct(barcodeInput.trim());
      if (scannedProduct) {
        addToCart(scannedProduct, 1);
        setBarcodeInput('');
      }
    } catch (err) {
      // Local fallback: search existing products by SKU or Name
      const match = products.find(
        (p) => p.sku?.toLowerCase() === barcodeInput.trim().toLowerCase()
      );
      if (match) {
        addToCart(match, 1);
        setBarcodeInput('');
      } else {
        setErrorMessage(`Barcode "${barcodeInput}" nahi mila`);
      }
    }
  };

  // Final Sale Complete karna
  const handleCheckout = async () => {
    if (cart.length === 0) {
      setErrorMessage('Cart khali hai, pehle koi saman chuniye');
      return;
    }

    if (paymentMethod === 'credit' && !customerPhone.trim()) {
      setErrorMessage('Khata me udhar likhne ke liye Customer Mobile Number zaroori hai!');
      return;
    }

    try {
      setBillingLoading(true);
      setErrorMessage('');

      // Backend DTO format: { customer_phone, items: [{ product_id, quantity, custom_price }], discount_amount, payment_method }
      const payload = {
        customer_phone: customerPhone.trim() || undefined,
        discount_amount: Number(discountAmount) || 0,
        payment_method: paymentMethod, // 'cash' | 'upi' | 'credit'
        items: cart.map((item) => ({
          product_id: item.product.id,
          quantity: item.quantity,
          custom_price: item.customPrice,
        })),
      };

      const result = await posApi.createSale(payload);
      setBillSuccess(result);
      clearCart();
    } catch (err) {
      setErrorMessage(err.message || 'Sale record karne me error aaya');
    } finally {
      setBillingLoading(false);
    }
  };

  // Filter products by search text
  const filteredProducts = products.filter(
    (p) =>
      p.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      p.sku?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <AppLayout title="Counter POS" subtitle="Tez Billing Terminal">
      {/* Fast Barcode Scanner Input */}
      <form onSubmit={handleBarcodeSubmit} style={{ marginBottom: '12px' }}>
        <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
          <Barcode size={20} style={{ position: 'absolute', left: '12px', color: 'var(--color-primary)' }} />
          <input
            type="text"
            className="form-input"
            style={{ paddingLeft: '40px', fontWeight: 600 }}
            placeholder="Barcode scan karein ya SKU likh ke Enter dabayein..."
            value={barcodeInput}
            onChange={(e) => setBarcodeInput(e.target.value)}
          />
        </div>
      </form>

      {/* Search Bar for manual product tap */}
      <div className="search-box">
        <Search size={18} />
        <input
          type="text"
          placeholder="Product ka naam dhundhein..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
      </div>

      {errorMessage && (
        <div
          style={{
            backgroundColor: 'var(--color-danger-light)',
            color: 'var(--color-danger)',
            padding: '10px',
            borderRadius: 'var(--radius-md)',
            fontSize: '0.82rem',
            marginBottom: '12px',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
          }}
        >
          <AlertCircle size={16} />
          <span>{errorMessage}</span>
        </div>
      )}

      {/* Product Quick-Tap Catalog */}
      <div style={{ marginBottom: '20px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
          <span style={{ fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-secondary)' }}>
            TAP KARKE ADD KAREIN ({filteredProducts.length})
          </span>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '8px', maxHeight: '220px', overflowY: 'auto' }}>
          {filteredProducts.slice(0, 20).map((prod) => (
            <div
              key={prod.id}
              className="card-clickable"
              onClick={() => addToCart(prod, 1)}
              style={{
                backgroundColor: 'var(--bg-surface)',
                border: '1px solid var(--border-subtle)',
                borderRadius: 'var(--radius-md)',
                padding: '8px 10px',
                display: 'flex',
                gap: '8px',
                alignItems: 'center',
                cursor: 'pointer',
              }}
            >
              {/* Product Thumbnail */}
              <div
                style={{
                  width: '38px',
                  height: '38px',
                  borderRadius: 'var(--radius-sm)',
                  backgroundColor: '#f8fafc',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  overflow: 'hidden',
                  flexShrink: 0,
                  border: '1px solid var(--border-subtle)',
                }}
              >
                {prod.images && prod.images.length > 0 ? (
                  <img
                    src={getImageUrl(prod.images[0])}
                    alt={prod.name}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                  />
                ) : (
                  <Package size={18} color="var(--text-muted)" style={{ opacity: 0.5 }} />
                )}
              </div>

              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: 700, fontSize: '0.82rem', lineHeight: 1.2, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {prod.name}
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '4px' }}>
                  <span style={{ fontWeight: 800, color: 'var(--color-primary)', fontSize: '0.88rem' }}>
                    ₹{prod.price}
                  </span>
                  <span
                    style={{
                      backgroundColor: 'var(--color-primary-light)',
                      color: 'var(--color-primary)',
                      borderRadius: 'var(--radius-full)',
                      padding: '2px 6px',
                      fontSize: '0.68rem',
                      fontWeight: 700,
                    }}
                  >
                    + Add
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Cart Summary & Checkout Box */}
      <div
        className="card"
        style={{
          border: '2px solid var(--color-primary-light)',
          padding: '14px',
          boxShadow: 'var(--shadow-md)',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
            <Receipt size={18} color="var(--color-primary)" />
            <span style={{ fontWeight: 700, fontSize: '0.95rem' }}>Counter Cart ({itemCount} items)</span>
          </div>
          {cart.length > 0 && (
            <button
              onClick={clearCart}
              style={{
                background: 'transparent',
                border: 'none',
                color: 'var(--color-danger)',
                fontSize: '0.75rem',
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              Khali Karein
            </button>
          )}
        </div>

        {/* Cart Items List */}
        {cart.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '16px', color: 'var(--text-muted)', fontSize: '0.82rem' }}>
            Cart abhi khali hai. Upar se product chuniye.
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '180px', overflowY: 'auto' }}>
            {cart.map((item) => (
              <div
                key={item.product.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '6px 0',
                  borderBottom: '1px solid var(--border-subtle)',
                }}
              >
                <div style={{ flex: 1, paddingRight: '8px' }}>
                  <div style={{ fontSize: '0.85rem', fontWeight: 600 }}>{item.product.name}</div>
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                    ₹{item.customPrice || item.product.price} x {item.quantity} = ₹{(item.customPrice || item.product.price) * item.quantity}
                  </div>
                </div>

                {/* Quantity Buttons */}
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <button
                    onClick={() => updateQuantity(item.product.id, item.quantity - 1)}
                    style={{
                      width: '26px',
                      height: '26px',
                      borderRadius: 'var(--radius-sm)',
                      border: '1px solid var(--border-strong)',
                      backgroundColor: 'var(--bg-surface)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      cursor: 'pointer',
                    }}
                  >
                    <Minus size={14} />
                  </button>
                  <span style={{ fontWeight: 700, minWidth: '18px', textAlign: 'center', fontSize: '0.85rem' }}>
                    {item.quantity}
                  </span>
                  <button
                    onClick={() => updateQuantity(item.product.id, item.quantity + 1)}
                    style={{
                      width: '26px',
                      height: '26px',
                      borderRadius: 'var(--radius-sm)',
                      border: '1px solid var(--border-strong)',
                      backgroundColor: 'var(--bg-surface)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      cursor: 'pointer',
                    }}
                  >
                    <Plus size={14} />
                  </button>
                  <button
                    onClick={() => removeFromCart(item.product.id)}
                    style={{
                      border: 'none',
                      background: 'transparent',
                      color: 'var(--color-danger)',
                      cursor: 'pointer',
                      padding: '4px',
                    }}
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Customer Phone & Discount */}
        {cart.length > 0 && (
          <div style={{ marginTop: '14px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <div>
              <label className="form-label" style={{ fontSize: '0.75rem' }}>
                Customer Mobile No. {paymentMethod === 'credit' && <span style={{ color: 'var(--color-danger)' }}>*</span>}
              </label>
              <input
                type="tel"
                placeholder="Grahak ka mobile number (optional)"
                className="form-input"
                style={{ padding: '8px 10px', fontSize: '0.85rem' }}
                value={customerPhone}
                onChange={(e) => setCustomerPhone(e.target.value)}
              />
            </div>

            {/* Payment Method Selector */}
            <div>
              <label className="form-label" style={{ fontSize: '0.75rem' }}>
                Payment Mode (Bhugtan Ka Tarika)
              </label>
              <div className="tab-pills">
                <button
                  type="button"
                  className={`tab-pill ${paymentMethod === 'cash' ? 'active' : ''}`}
                  onClick={() => setPaymentMethod('cash')}
                >
                  💵 Cash
                </button>
                <button
                  type="button"
                  className={`tab-pill ${paymentMethod === 'upi' ? 'active' : ''}`}
                  onClick={() => setPaymentMethod('upi')}
                >
                  📱 UPI
                </button>
                <button
                  type="button"
                  className={`tab-pill ${paymentMethod === 'credit' ? 'active' : ''}`}
                  onClick={() => setPaymentMethod('credit')}
                  style={{ color: paymentMethod === 'credit' ? 'var(--color-danger)' : undefined }}
                >
                  📖 Khata Udhar
                </button>
              </div>
            </div>

            {/* Bill Totals */}
            <div
              style={{
                backgroundColor: 'var(--bg-surface-subtle)',
                padding: '10px',
                borderRadius: 'var(--radius-md)',
                marginTop: '4px',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                <span>Subtotal:</span>
                <span>₹{subtotal.toFixed(2)}</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '4px' }}>
                <span style={{ fontSize: '0.95rem', fontWeight: 800 }}>Total Amount:</span>
                <span style={{ fontSize: '1.2rem', fontWeight: 900, color: 'var(--color-primary)' }}>
                  ₹{total.toFixed(2)}
                </span>
              </div>
            </div>

            {/* Checkout Button */}
            <button
              onClick={handleCheckout}
              disabled={billingLoading}
              className={`btn btn-block btn-lg ${paymentMethod === 'credit' ? 'btn-danger' : 'btn-success'}`}
              style={{ marginTop: '8px' }}
            >
              {billingLoading ? 'Parchi ban rahi hai...' : `₹${total.toFixed(2)} Ka Bill Banayein`}
            </button>
          </div>
        )}
      </div>

      {/* Bill Success Receipt Bottom Sheet */}
      {billSuccess && (
        <div className="modal-backdrop" onClick={() => setBillSuccess(null)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <div style={{ textAlign: 'center', padding: '10px' }}>
              <CheckCircle size={52} color="var(--color-success)" style={{ margin: '0 auto 8px auto' }} />
              <h2 style={{ fontSize: '1.3rem', fontWeight: 800 }}>Bill Ban Gaya!</h2>
              <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                Bill Number: <strong>{billSuccess.bill?.bill_number}</strong>
              </p>

              <div
                style={{
                  backgroundColor: 'var(--bg-surface-subtle)',
                  borderRadius: 'var(--radius-md)',
                  padding: '12px',
                  margin: '16px 0',
                  textAlign: 'left',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '6px' }}>
                  <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>Total Amount:</span>
                  <span style={{ fontWeight: 800 }}>₹{billSuccess.bill?.final_amount}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '6px' }}>
                  <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>Payment Mode:</span>
                  <span style={{ fontWeight: 700, textTransform: 'uppercase' }}>{billSuccess.bill?.payment_method}</span>
                </div>
                {billSuccess.loyalty_points_credited > 0 && (
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span style={{ fontSize: '0.8rem', color: 'var(--color-primary)' }}>Loyalty Points:</span>
                    <span style={{ fontWeight: 700, color: 'var(--color-primary)' }}>
                      +{billSuccess.loyalty_points_credited} pts
                    </span>
                  </div>
                )}
              </div>

              <div style={{ display: 'flex', gap: '8px' }}>
                <a
                  href={posApi.getReceiptUrl(billSuccess.bill?.bill_number)}
                  target="_blank"
                  rel="noreferrer"
                  className="btn btn-primary btn-block"
                  style={{ textDecoration: 'none' }}
                >
                  <Printer size={16} /> Digital PDF Receipt Dekhein
                </a>
                <button
                  className="btn btn-secondary"
                  onClick={() => setBillSuccess(null)}
                >
                  Band
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </AppLayout>
  );
};
