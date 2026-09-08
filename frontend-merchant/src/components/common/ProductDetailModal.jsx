/**
 * ProductDetailModal Component
 * 
 * Hinglish Hint:
 * Kisi bhi product par tap karne par ek modern native-app style bottom sheet khulti hai:
 * - High-resolution image gallery (multiple images swipe/preview)
 * - Complete product specifications & flexible attributes (Brand, Size, Color, etc.)
 * - SKU, Price, Stock indicator
 * - Customer ke liye instant "Hold / Reserve" action
 * - Merchant ke liye "Adjust Stock" action
 */

import React, { useState } from 'react';
import {
  X,
  ShoppingBag,
  Package,
  Layers,
  Tag,
  CheckCircle,
  AlertCircle,
  Copy,
  Clock,
  ChevronLeft,
  ChevronRight,
  Edit3,
  Plus,
} from 'lucide-react';
import { getCategoryEmoji } from '../../utils/categoryMeta';
import { getImageUrl } from '../../utils/imageUrl';

export const ProductDetailModal = ({
  product,
  onClose,
  onReserve,        // Customer action
  onAdjustStock,    // Merchant action
  onEditProduct,    // Merchant edit action
  isMerchant = false,
}) => {
  if (!product) return null;

  const images = product.images && product.images.length > 0 ? product.images : [];
  const [selectedImageIdx, setSelectedImageIdx] = useState(0);
  const [copiedSKU, setCopiedSKU] = useState(false);

  // Reserve form state for customers
  const [reserveQty, setReserveQty] = useState(1);
  const [reserveHours, setReserveHours] = useState(4);
  const [reserveNotes, setReserveNotes] = useState('');
  const [reserving, setReserving] = useState(false);

  const currentStock = Number(
    product.stock_quantity ??
    product.inventory?.available_quantity ??
    product.inventory?.quantity ??
    0
  );
  const inStock = currentStock > 0;

  const handleCopySKU = () => {
    if (product.sku) {
      navigator.clipboard.writeText(product.sku);
      setCopiedSKU(true);
      setTimeout(() => setCopiedSKU(false), 2000);
    }
  };

  const handleReserveSubmit = async (e) => {
    e.preventDefault();
    if (onReserve) {
      setReserving(true);
      try {
        await onReserve({
          product,
          quantity: Number(reserveQty),
          hold_hours: Number(reserveHours),
          notes: reserveNotes,
        });
      } finally {
        setReserving(false);
      }
    }
  };

  const hasAttributes = product.attributes && Object.keys(product.attributes).length > 0;

  return (
    <div className="modal-backdrop" onClick={onClose} style={{ zIndex: 120 }}>
      <div
        className="bottom-sheet"
        onClick={(e) => e.stopPropagation()}
        style={{
          maxHeight: '90vh',
          display: 'flex',
          flexDirection: 'column',
          padding: 0,
          overflow: 'hidden',
        }}
      >
        {/* Modal Handle & Close Button */}
        <div
          style={{
            padding: '12px 16px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            borderBottom: '1px solid var(--border-subtle)',
            backgroundColor: '#ffffff',
          }}
        >
          <div className="sheet-handle" style={{ margin: 0 }} />
          <button
            onClick={onClose}
            style={{
              background: '#f1f5f9',
              border: 'none',
              borderRadius: '50%',
              width: '32px',
              height: '32px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              color: 'var(--text-secondary)',
            }}
            title="Band Karein"
          >
            <X size={18} />
          </button>
        </div>

        {/* Scrollable Content */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '16px' }}>
          {/* Main Image Gallery */}
          <div style={{ marginBottom: '16px' }}>
            <div
              style={{
                width: '100%',
                height: '240px',
                borderRadius: 'var(--radius-lg)',
                backgroundColor: '#f8fafc',
                overflow: 'hidden',
                position: 'relative',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                border: '1px solid var(--border-subtle)',
                boxShadow: 'inset 0 0 10px rgba(0,0,0,0.03)',
              }}
            >
              {images.length > 0 ? (
                <img
                  src={getImageUrl(images[selectedImageIdx] || images[0])}
                  alt={product.name}
                  style={{
                    width: '100%',
                    height: '100%',
                    objectFit: 'contain',
                    transition: 'all 0.2s ease',
                  }}
                />
              ) : (
                <div style={{ textAlign: 'center', color: 'var(--text-muted)' }}>
                  <Package size={64} style={{ opacity: 0.4, margin: '0 auto 8px auto' }} />
                  <div style={{ fontSize: '0.82rem', fontWeight: 600 }}>Koi Photo Available Nahi Hai</div>
                </div>
              )}

              {/* Multiple Images Navigation Arrows */}
              {images.length > 1 && (
                <>
                  <button
                    onClick={() => setSelectedImageIdx((prev) => (prev > 0 ? prev - 1 : images.length - 1))}
                    style={{
                      position: 'absolute',
                      left: '8px',
                      top: '50%',
                      transform: 'translateY(-50%)',
                      backgroundColor: 'rgba(255,255,255,0.9)',
                      border: 'none',
                      borderRadius: '50%',
                      width: '32px',
                      height: '32px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      cursor: 'pointer',
                      boxShadow: '0 2px 6px rgba(0,0,0,0.15)',
                    }}
                  >
                    <ChevronLeft size={18} />
                  </button>
                  <button
                    onClick={() => setSelectedImageIdx((prev) => (prev < images.length - 1 ? prev + 1 : 0))}
                    style={{
                      position: 'absolute',
                      right: '8px',
                      top: '50%',
                      transform: 'translateY(-50%)',
                      backgroundColor: 'rgba(255,255,255,0.9)',
                      border: 'none',
                      borderRadius: '50%',
                      width: '32px',
                      height: '32px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      cursor: 'pointer',
                      boxShadow: '0 2px 6px rgba(0,0,0,0.15)',
                    }}
                  >
                    <ChevronRight size={18} />
                  </button>
                </>
              )}
            </div>

            {/* Thumbnail Strip */}
            {images.length > 1 && (
              <div style={{ display: 'flex', gap: '8px', marginTop: '10px', overflowX: 'auto', paddingBottom: '4px' }}>
                {images.map((img, idx) => (
                  <img
                    key={idx}
                    src={getImageUrl(img)}
                    alt={`Thumb ${idx + 1}`}
                    onClick={() => setSelectedImageIdx(idx)}
                    style={{
                      width: '52px',
                      height: '52px',
                      borderRadius: 'var(--radius-sm)',
                      objectFit: 'cover',
                      cursor: 'pointer',
                      border: selectedImageIdx === idx ? '2px solid var(--color-primary)' : '1px solid var(--border-subtle)',
                      opacity: selectedImageIdx === idx ? 1 : 0.65,
                      transition: 'all 0.15s ease',
                      flexShrink: 0,
                    }}
                  />
                ))}
              </div>
            )}
          </div>

          {/* Product Header & Pricing */}
          <div style={{ marginBottom: '16px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '10px' }}>
              <div>
                <h2 style={{ fontSize: '1.25rem', fontWeight: 800, color: 'var(--text-primary)', lineHeight: 1.25 }}>
                  {product.name}
                </h2>
                {Boolean(product.category) && (
                  <div style={{ marginTop: '4px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <span style={{ fontSize: '0.85rem' }}>
                      {getCategoryEmoji(
                        typeof product.category === 'object'
                          ? (product.category?.slug || product.category?.name || '')
                          : String(product.category || '')
                      )}
                    </span>
                    <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', fontWeight: 600 }}>
                      {typeof product.category === 'object'
                        ? (product.category?.name || product.category?.slug || 'Category')
                        : String(product.category || 'Category')}
                    </span>
                  </div>
                )}
              </div>

              <div style={{ textAlign: 'right', flexShrink: 0 }}>
                <div style={{ fontSize: '1.4rem', fontWeight: 900, color: 'var(--color-primary)' }}>
                  ₹{product.price}
                </div>
                {isMerchant && product.cost_price > 0 && (
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                    Kharid: ₹{product.cost_price}
                  </div>
                )}
              </div>
            </div>

            {/* Badges Bar: Stock & SKU */}
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginTop: '10px', alignItems: 'center' }}>
              <span
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '5px',
                  padding: '3px 10px',
                  borderRadius: 'var(--radius-full)',
                  fontSize: '0.78rem',
                  fontWeight: 700,
                  backgroundColor: inStock ? 'rgba(16, 185, 129, 0.12)' : 'rgba(239, 68, 68, 0.12)',
                  color: inStock ? '#065f46' : '#991b1b',
                }}
              >
                <span
                  style={{
                    width: '6px',
                    height: '6px',
                    borderRadius: '50%',
                    backgroundColor: inStock ? '#10b981' : '#ef4444',
                  }}
                />
                {inStock ? `Stock: ${currentStock} pcs available` : 'Out of Stock'}
              </span>

              {product.sku && (
                <button
                  onClick={handleCopySKU}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '4px',
                    padding: '3px 10px',
                    borderRadius: 'var(--radius-full)',
                    fontSize: '0.75rem',
                    fontWeight: 600,
                    backgroundColor: '#f1f5f9',
                    color: 'var(--text-secondary)',
                    border: 'none',
                    cursor: 'pointer',
                  }}
                  title="SKU Copy Karein"
                >
                  <Copy size={12} />
                  <span>SKU: {product.sku}</span>
                  {copiedSKU && <span style={{ color: '#15803d', fontWeight: 700 }}>✓ Copied</span>}
                </button>
              )}
            </div>
          </div>

          {/* Flexible Specifications & Attributes Card */}
          {hasAttributes && (
            <div
              style={{
                backgroundColor: '#f8fafc',
                border: '1px solid var(--border-subtle)',
                borderRadius: 'var(--radius-md)',
                padding: '12px 14px',
                marginBottom: '16px',
              }}
            >
              <div style={{ fontSize: '0.78rem', fontWeight: 800, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '10px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Layers size={14} color="var(--color-primary)" />
                SPECIFICATIONS & ATTRIBUTES
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '8px' }}>
                {Object.entries(product.attributes).map(([key, val]) => {
                  if (!val) return null;
                  const label = key.replace(/_/g, ' ').toUpperCase();
                  return (
                    <div key={key} style={{ backgroundColor: '#ffffff', padding: '8px 10px', borderRadius: 'var(--radius-sm)', border: '1px solid #e2e8f0' }}>
                      <div style={{ fontSize: '0.68rem', color: 'var(--text-muted)', fontWeight: 700 }}>
                        {label}
                      </div>
                      <div style={{ fontSize: '0.82rem', fontWeight: 700, color: 'var(--text-primary)', marginTop: '2px' }}>
                        {String(val)}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Customer Action: Reserve / Hold Item */}
          {!isMerchant && (
            <div style={{ marginTop: '14px', borderTop: '1px solid var(--border-subtle)', paddingTop: '16px' }}>
              <div style={{ fontSize: '0.85rem', fontWeight: 800, marginBottom: '10px' }}>
                🛍️ DUKAN SE PICKUP KE LIYE HOLD / RESERVE KAREIN
              </div>

              {!inStock ? (
                <div style={{ backgroundColor: '#fee2e2', color: '#b91c1c', padding: '10px', borderRadius: 'var(--radius-md)', fontSize: '0.82rem', textAlign: 'center', fontWeight: 600 }}>
                  Yeh product abhi out of stock hai.
                </div>
              ) : (
                <form onSubmit={handleReserveSubmit}>
                  <div style={{ display: 'flex', gap: '10px', marginBottom: '10px' }}>
                    <div style={{ flex: 1 }}>
                      <label className="form-label" style={{ fontSize: '0.75rem' }}>Kitne Pieces (Qty)</label>
                      <input
                        type="number"
                        min="1"
                        max={currentStock || 10}
                        required
                        className="form-input"
                        value={reserveQty}
                        onChange={(e) => setReserveQty(e.target.value)}
                      />
                    </div>
                    <div style={{ flex: 1 }}>
                      <label className="form-label" style={{ fontSize: '0.75rem' }}>Hold Time (Hours)</label>
                      <select
                        className="form-input"
                        value={reserveHours}
                        onChange={(e) => setReserveHours(e.target.value)}
                      >
                        <option value="2">2 Ghante</option>
                        <option value="4">4 Ghante</option>
                        <option value="8">8 Ghante</option>
                        <option value="24">24 Ghante</option>
                      </select>
                    </div>
                  </div>

                  <div style={{ marginBottom: '12px' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Koi Special Note (Optional)</label>
                    <input
                      type="text"
                      className="form-input"
                      placeholder="e.g. Mai 2 baje counter par aaunga"
                      value={reserveNotes}
                      onChange={(e) => setReserveNotes(e.target.value)}
                    />
                  </div>

                  <button
                    type="submit"
                    className="btn btn-primary btn-block btn-lg"
                    disabled={reserving}
                  >
                    <ShoppingBag size={18} />
                    <span>{reserving ? 'Hold Ho Raha Hai...' : `Item Hold Karein (₹${product.price * reserveQty})`}</span>
                  </button>
                </form>
              )}
            </div>
          )}

          {/* Merchant Actions: Edit Details/Photos & Stock Adjust */}
          {isMerchant && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '16px' }}>
              {onEditProduct && (
                <button
                  type="button"
                  className="btn btn-secondary btn-block"
                  style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px' }}
                  onClick={() => {
                    onClose();
                    onEditProduct(product);
                  }}
                >
                  <Edit3 size={16} /> Product & Photos Edit Karein
                </button>
              )}
              {onAdjustStock && (
                <button
                  type="button"
                  className="btn btn-primary btn-block"
                  style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px' }}
                  onClick={() => {
                    onClose();
                    onAdjustStock(product);
                  }}
                >
                  <Plus size={16} /> Stock In / Stock Out Adjust Karein
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
