/**
 * Shop Storefront Screen (Customer View)
 * 
 * Hinglish Hint:
 * Kisi specific dukan ka online digital panna (Storefront):
 * - Dukan ki details, timing, address aur WhatsApp contact
 * - Dukan ke sare products dekhna
 * - Item Reserve / Hold karna taaki dukan par jakar pickup kar sakein
 */

import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Store,
  MapPin,
  Clock,
  Phone,
  Search,
  ShoppingBag,
  CheckCircle,
  AlertCircle,
  Package,
  MessageCircle,
  Eye,
} from 'lucide-react';
import { shopApi } from '../../api/shop.api';
import { productApi } from '../../api/product.api';
import { reservationApi } from '../../api/reservation.api';
import { useAuth } from '../../context/AuthContext';
import { AppLayout } from '../../components/layout/AppLayout';
import { ProductDetailModal } from '../../components/common/ProductDetailModal';
import { ShopDetailModal } from '../../components/common/ShopDetailModal';
import { getImageUrl } from '../../utils/imageUrl';

export const StorefrontScreen = () => {
  const { slug } = useParams();
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  const [shop, setShop] = useState(null);
  const [products, setProducts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');

  // Selected product for detail inspection modal
  const [inspectedProduct, setInspectedProduct] = useState(null);
  const [showShopDetailModal, setShowShopDetailModal] = useState(false);
  const [reservationSuccess, setReservationSuccess] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    if (slug) {
      loadShopAndProducts();
    }
  }, [slug]);

  const loadShopAndProducts = async () => {
    try {
      setLoading(true);
      const [shopData, prodData] = await Promise.all([
        shopApi.getShopBySlug(slug),
        productApi.listByShopSlug(slug),
      ]);
      setShop(shopData);
      const list = Array.isArray(prodData?.products) ? prodData.products : (Array.isArray(prodData) ? prodData : []);
      setProducts(list);
    } catch (err) {
      console.error('Failed to load shop:', err);
      setProducts([]);
    } finally {
      setLoading(false);
    }
  };

  const handleReserveFromModal = async ({ product, quantity, hold_hours, notes }) => {
    if (!isAuthenticated) {
      alert('Item reserve karne ke liye pehle Login karein');
      navigate('/login');
      return;
    }

    try {
      setError('');
      const res = await reservationApi.createReservation({
        product_id: product.id,
        quantity,
        hold_hours,
        notes,
      });
      setReservationSuccess(res);
      setInspectedProduct(null);
    } catch (err) {
      alert(err.message || 'Reservation fail ho gaya');
    }
  };

  const filteredProducts = (Array.isArray(products) ? products : []).filter((p) =>
    (p.name || '').toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <AppLayout title={shop?.name || 'Dukan Storefront'} showBack={true}>
      {/* Shop Profile Banner & Photos */}
      {shop && (
        <div
          className="card"
          style={{
            background: 'linear-gradient(135deg, #1e1b4b 0%, #312e81 100%)',
            color: '#ffffff',
            border: 'none',
            padding: 0,
            overflow: 'hidden',
            marginBottom: '14px',
          }}
        >
          {/* Shop Promotional Banner Image if available */}
          {shop.banners && shop.banners.length > 0 && (
            <div style={{ width: '100%', height: '120px', overflow: 'hidden', position: 'relative' }}>
              <img
                src={getImageUrl(shop.banners[0])}
                alt={shop.name}
                style={{ width: '100%', height: '100%', objectFit: 'cover' }}
              />
              <div
                style={{
                  position: 'absolute',
                  inset: 0,
                  background: 'linear-gradient(to bottom, rgba(0,0,0,0.1) 0%, rgba(30,27,75,0.85) 100%)',
                }}
              />
            </div>
          )}

          <div style={{ padding: '16px' }}>
            <div style={{ display: 'flex', gap: '14px', alignItems: 'center' }}>
              {/* Shop Logo Avatar */}
              <div
                style={{
                  width: '56px',
                  height: '56px',
                  borderRadius: 'var(--radius-md)',
                  backgroundColor: 'rgba(255, 255, 255, 0.15)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                  overflow: 'hidden',
                  border: '2px solid rgba(255, 255, 255, 0.3)',
                  boxShadow: '0 4px 10px rgba(0,0,0,0.3)',
                }}
              >
                {shop.logo_url ? (
                  <img
                    src={getImageUrl(shop.logo_url)}
                    alt={shop.name}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                  />
                ) : (
                  <Store size={30} />
                )}
              </div>

              <div style={{ flex: 1, minWidth: 0 }}>
                <h1 style={{ fontSize: '1.25rem', fontWeight: 800, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {shop.name}
                </h1>
                <div style={{ fontSize: '0.78rem', opacity: 0.85 }}>{shop.category}</div>
                <div
                  style={{
                    fontSize: '0.74rem',
                    opacity: 0.8,
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    marginTop: '4px',
                  }}
                >
                  <MapPin size={12} /> {shop.address}
                </div>
              </div>
            </div>

            {/* Quick Contact & Status Bar */}
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginTop: '14px',
                paddingTop: '10px',
                borderTop: '1px solid rgba(255, 255, 255, 0.15)',
                fontSize: '0.75rem',
              }}
            >
              <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                <Clock size={12} /> {shop.timing || '9:00 AM - 9:00 PM'}
              </span>

              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                {shop.phone && (
                  <a
                    href={`https://wa.me/91${shop.phone.replace(/[^0-9]/g, '')}?text=Namaste%20${encodeURIComponent(shop.name)}`}
                    target="_blank"
                    rel="noreferrer"
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '4px',
                      backgroundColor: '#25d366',
                      color: '#ffffff',
                      padding: '3px 8px',
                      borderRadius: 'var(--radius-full)',
                      fontWeight: 700,
                      textDecoration: 'none',
                      fontSize: '0.72rem',
                    }}
                  >
                    <MessageCircle size={12} /> WhatsApp
                  </a>
                )}

                <span
                  style={{
                    backgroundColor: shop.is_active ? '#10b981' : '#ef4444',
                    color: '#ffffff',
                    padding: '2px 8px',
                    borderRadius: 'var(--radius-full)',
                    fontWeight: 700,
                  }}
                >
                  {shop.is_active ? 'Open Abhi' : 'Closed'}
                </span>
              </div>
            </div>

            {/* View Full Shop Info Button */}
            <button
              onClick={() => setShowShopDetailModal(true)}
              style={{
                marginTop: '12px',
                width: '100%',
                background: 'rgba(255, 255, 255, 0.15)',
                border: '1px solid rgba(255, 255, 255, 0.25)',
                color: '#ffffff',
                padding: '8px 12px',
                borderRadius: '10px',
                fontSize: '0.8rem',
                fontWeight: 700,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: '6px',
                cursor: 'pointer',
              }}
            >
              <Eye size={15} />
              <span>Dukan Ki Puri Jankari, Timing & Photos Dekhein</span>
            </button>
          </div>
        </div>
      )}

      {/* Search Bar */}
      <div className="search-box">
        <Search size={18} />
        <input
          type="text"
          placeholder="Is dukan me saman khojein..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
      </div>

      {/* Products Grid (Spacious Professional E-commerce Cards) */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: '16px' }}>
        {filteredProducts.map((p) => {
          const stock = Number(p.stock_quantity ?? p.inventory?.available_quantity ?? p.inventory?.quantity ?? 0);
          const inStock = stock > 0;
          const hasDiscount = p.compare_price && p.compare_price > p.price;
          const discountPct = hasDiscount ? Math.round(((p.compare_price - p.price) / p.compare_price) * 100) : 0;
          const brand = p.attributes?.brand || p.attributes?.company;

          const handleOpenModal = () => {
            const enriched = {
              ...p,
              shop_name: p.shop_name || shop?.name,
              shop_slug: p.shop_slug || shop?.slug,
              shop_address: p.shop_address || shop?.address,
              shop_city: p.shop_city || shop?.city,
              shop_phone: p.shop_phone || shop?.phone,
              shop_latitude: p.shop_latitude || shop?.latitude,
              shop_longitude: p.shop_longitude || shop?.longitude,
            };
            setInspectedProduct(enriched);
          };

          return (
            <div
              key={p.id}
              className="card card-clickable"
              onClick={handleOpenModal}
              style={{
                margin: 0,
                padding: '16px',
                borderRadius: '18px',
                border: '1.5px solid var(--border-subtle)',
                boxShadow: '0 6px 18px rgba(0,0,0,0.04)',
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between',
                cursor: 'pointer',
                transition: 'transform 0.15s ease, box-shadow 0.15s ease',
              }}
            >
              <div>
                {/* Big High-Res Product Photo */}
                <div
                  style={{
                    width: '100%',
                    height: '160px',
                    borderRadius: '14px',
                    backgroundColor: '#ffffff',
                    overflow: 'hidden',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    marginBottom: '12px',
                    border: '1.5px solid var(--border-subtle)',
                    position: 'relative',
                    boxShadow: 'inset 0 0 8px rgba(0,0,0,0.02)',
                  }}
                >
                  {p.images && p.images.length > 0 ? (
                    <img
                      src={getImageUrl(p.images[0])}
                      alt={p.name}
                      style={{ width: '100%', height: '100%', objectFit: 'contain', padding: '6px' }}
                      onError={(e) => { e.currentTarget.style.display = 'none'; }}
                    />
                  ) : (
                    <Package size={44} color="var(--text-muted)" style={{ opacity: 0.4 }} />
                  )}

                  {/* Stock Pill on image */}
                  <span
                    style={{
                      position: 'absolute',
                      bottom: '6px',
                      left: '6px',
                      right: '6px',
                      textAlign: 'center',
                      fontSize: '0.66rem',
                      fontWeight: 800,
                      padding: '3px 6px',
                      borderRadius: '6px',
                      backgroundColor: inStock ? 'rgba(16, 185, 129, 0.95)' : 'rgba(239, 68, 68, 0.95)',
                      color: '#ffffff',
                      boxShadow: '0 2px 4px rgba(0,0,0,0.15)',
                    }}
                  >
                    {inStock ? `✓ ${stock} in store` : 'Out of stock'}
                  </span>
                </div>

                {/* Product Title */}
                <div style={{ fontWeight: 800, fontSize: '1.05rem', lineHeight: 1.3, color: 'var(--text-primary)' }}>
                  {p.name}
                </div>

                {/* Brand & Attribute Chips Upfront */}
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '5px', marginTop: '6px' }}>
                  {brand && (
                    <span style={{ fontSize: '0.72rem', background: 'rgba(37, 99, 235, 0.09)', color: '#2563eb', padding: '2px 7px', borderRadius: '6px', fontWeight: 700 }}>
                      {brand}
                    </span>
                  )}
                  {p.attributes?.size && (
                    <span style={{ fontSize: '0.7rem', background: '#f1f5f9', color: 'var(--text-secondary)', padding: '2px 6px', borderRadius: '4px', fontWeight: 600 }}>
                      Size: {p.attributes.size}
                    </span>
                  )}
                  {p.attributes?.weight && (
                    <span style={{ fontSize: '0.7rem', background: '#f1f5f9', color: 'var(--text-secondary)', padding: '2px 6px', borderRadius: '4px', fontWeight: 600 }}>
                      {p.attributes.weight}
                    </span>
                  )}
                </div>
              </div>

              {/* Bottom: Pricing & CTA Button */}
              <div style={{ marginTop: '14px', paddingTop: '10px', borderTop: '1px solid var(--border-subtle)' }}>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: '8px', marginBottom: '10px' }}>
                  <span style={{ fontWeight: 900, fontSize: '1.25rem', color: 'var(--color-primary)' }}>
                    ₹{p.price}
                  </span>
                  {hasDiscount && (
                    <>
                      <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)', textDecoration: 'line-through' }}>
                        ₹{p.compare_price}
                      </span>
                      <span style={{ fontSize: '0.7rem', color: '#15803d', fontWeight: 800, background: '#dcfce7', padding: '1px 6px', borderRadius: '4px' }}>
                        Save ₹{p.compare_price - p.price} ({discountPct}% OFF)
                      </span>
                    </>
                  )}
                </div>

                <button
                  className="btn btn-primary btn-block"
                  style={{
                    padding: '8px 12px',
                    fontSize: '0.82rem',
                    fontWeight: 700,
                    borderRadius: '8px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: '6px',
                  }}
                  disabled={!inStock}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleOpenModal();
                  }}
                >
                  <Eye size={14} />
                  <span>{inStock ? 'View Details & Hold' : 'Out of Stock'}</span>
                </button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Product Detail Modal */}
      {inspectedProduct && (
        <ProductDetailModal
          product={inspectedProduct}
          onClose={() => setInspectedProduct(null)}
          onReserve={handleReserveFromModal}
          isMerchant={false}
        />
      )}

      {/* Reservation Success Bottom Sheet with OTP/Pickup Code */}
      {reservationSuccess && (
        <div className="modal-backdrop" onClick={() => setReservationSuccess(null)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <div style={{ textAlign: 'center', padding: '10px' }}>
              <CheckCircle size={48} color="var(--color-success)" style={{ margin: '0 auto 8px auto' }} />
              <h2 style={{ fontSize: '1.2rem', fontWeight: 800 }}>Item Hold Ho Gaya!</h2>
              <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                Dukan counter par jakar yeh Pickup Code dikhayein:
              </p>

              <div
                style={{
                  backgroundColor: 'var(--color-primary-light)',
                  color: 'var(--color-primary)',
                  fontSize: '1.8rem',
                  fontWeight: 900,
                  letterSpacing: '4px',
                  padding: '12px',
                  borderRadius: 'var(--radius-md)',
                  margin: '16px 0',
                }}
              >
                {reservationSuccess.pickup_code || reservationSuccess.reservation_number || 'HOLD-OK'}
              </div>

              <button
                className="btn btn-secondary btn-block"
                onClick={() => setReservationSuccess(null)}
              >
                Theek Hai
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Shop Detail Modal */}
      {showShopDetailModal && shop && (
        <ShopDetailModal
          shop={shop}
          onClose={() => setShowShopDetailModal(false)}
        />
      )}
    </AppLayout>
  );
};
