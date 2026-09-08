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
} from 'lucide-react';
import { shopApi } from '../../api/shop.api';
import { productApi } from '../../api/product.api';
import { reservationApi } from '../../api/reservation.api';
import { useAuth } from '../../context/AuthContext';
import { AppLayout } from '../../components/layout/AppLayout';
import { ProductDetailModal } from '../../components/common/ProductDetailModal';

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
                src={shop.banners[0]}
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
                    src={shop.logo_url}
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

      {/* Products Grid (Real e-commerce cards with photos) */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '10px' }}>
        {filteredProducts.map((p) => (
          <div
            key={p.id}
            className="card card-clickable"
            onClick={() => setInspectedProduct(p)}
            style={{
              margin: 0,
              padding: '10px',
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'space-between',
              cursor: 'pointer',
              transition: 'transform 0.15s ease, box-shadow 0.15s ease',
            }}
          >
            <div>
              {/* Product Photo or Placeholder */}
              <div
                style={{
                  width: '100%',
                  height: '110px',
                  borderRadius: 'var(--radius-md)',
                  backgroundColor: '#f8fafc',
                  overflow: 'hidden',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  marginBottom: '8px',
                  border: '1px solid var(--border-subtle)',
                }}
              >
                {p.images && p.images.length > 0 ? (
                  <img
                    src={p.images[0]}
                    alt={p.name}
                    style={{ width: '100%', height: '100%', objectFit: 'contain' }}
                  />
                ) : (
                  <Package size={32} color="var(--text-muted)" style={{ opacity: 0.5 }} />
                )}
              </div>

              <div style={{ fontWeight: 700, fontSize: '0.88rem', lineHeight: 1.25 }}>
                {p.name}
              </div>
              <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginTop: '2px' }}>
                {p.stock_quantity > 0 ? `${p.stock_quantity} available` : 'Out of stock'}
              </div>

              {/* Attributes Chips */}
              {p.attributes && (p.attributes.company || p.attributes.size) && (
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '3px', marginTop: '4px' }}>
                  {p.attributes.company && (
                    <span className="badge badge-info" style={{ fontSize: '0.62rem', padding: '1px 5px' }}>
                      {p.attributes.company}
                    </span>
                  )}
                  {p.attributes.size && (
                    <span className="badge badge-muted" style={{ fontSize: '0.62rem', padding: '1px 5px' }}>
                      {p.attributes.size}
                    </span>
                  )}
                </div>
              )}
            </div>

            <div style={{ marginTop: '10px' }}>
              <div style={{ fontWeight: 800, fontSize: '1rem', color: 'var(--color-primary)', marginBottom: '6px' }}>
                ₹{p.price}
              </div>
              <button
                className="btn btn-primary btn-sm btn-block"
                style={{ padding: '6px 8px', fontSize: '0.78rem' }}
                disabled={p.stock_quantity <= 0}
                onClick={(e) => {
                  e.stopPropagation();
                  setInspectedProduct(p);
                }}
              >
                <ShoppingBag size={13} /> Hold / View
              </button>
            </div>
          </div>
        ))}
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
    </AppLayout>
  );
};
