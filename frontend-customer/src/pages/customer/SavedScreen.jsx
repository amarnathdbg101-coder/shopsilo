import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Heart, Store, Package, MapPin, Phone, Navigation, Trash2 } from 'lucide-react';
import { AppLayout } from '../../components/layout/AppLayout';
import { useSaved } from '../../context/SavedContext';
import { getImageUrl } from '../../utils/imageUrl';
import { ProductDetailModal } from '../../components/common/ProductDetailModal';
import { reservationApi } from '../../api/reservation.api';
import { useAuth } from '../../context/AuthContext';

export const SavedScreen = () => {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();
  const { savedProducts, savedShops, toggleSaveProduct, toggleSaveShop } = useSaved();
  const [activeTab, setActiveTab] = useState('products'); // 'products' | 'shops'
  const [inspectedProduct, setInspectedProduct] = useState(null);

  return (
    <AppLayout title="Saved Items" subtitle="Aapke Pasandeeda Products & Dukaanein">
      {/* Subtabs */}
      <div
        style={{
          display: 'flex',
          background: 'var(--bg-card)',
          borderRadius: '12px',
          padding: '4px',
          marginBottom: '16px',
          border: '1px solid var(--border-subtle)',
        }}
      >
        <button
          onClick={() => setActiveTab('products')}
          style={{
            flex: 1,
            padding: '8px 12px',
            borderRadius: '8px',
            border: 'none',
            fontSize: '0.85rem',
            fontWeight: 600,
            cursor: 'pointer',
            background: activeTab === 'products' ? 'var(--color-primary)' : 'transparent',
            color: activeTab === 'products' ? '#fff' : 'var(--text-secondary)',
            transition: 'all 0.2s',
          }}
        >
          Products ({savedProducts.length})
        </button>
        <button
          onClick={() => setActiveTab('shops')}
          style={{
            flex: 1,
            padding: '8px 12px',
            borderRadius: '8px',
            border: 'none',
            fontSize: '0.85rem',
            fontWeight: 600,
            cursor: 'pointer',
            background: activeTab === 'shops' ? 'var(--color-primary)' : 'transparent',
            color: activeTab === 'shops' ? '#fff' : 'var(--text-secondary)',
            transition: 'all 0.2s',
          }}
        >
          Dukaanein ({savedShops.length})
        </button>
      </div>

      {/* Content */}
      {activeTab === 'products' ? (
        savedProducts.length === 0 ? (
          <div className="card" style={{ textAlign: 'center', padding: '40px 20px' }}>
            <Heart size={44} color="var(--text-muted)" style={{ margin: '0 auto 12px auto' }} />
            <div style={{ fontWeight: 700, fontSize: '1.05rem', color: 'var(--text-primary)' }}>
              No saved products
            </div>
            <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginTop: '6px' }}>
              Tap the heart on any product to keep it here for quick price check and availability.
            </p>
            <button
              onClick={() => navigate('/')}
              style={{
                marginTop: '16px',
                background: 'var(--color-primary)',
                color: '#fff',
                border: 'none',
                padding: '8px 16px',
                borderRadius: '8px',
                fontWeight: 600,
                fontSize: '0.85rem',
                cursor: 'pointer',
              }}
            >
              Browse Products
            </button>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
            {savedProducts.map((p) => (
              <div
                key={p.id}
                className="card card-clickable"
                onClick={() => setInspectedProduct(p)}
                style={{
                  margin: 0,
                  padding: '16px',
                  borderRadius: '16px',
                  border: '1.5px solid var(--border-subtle)',
                  boxShadow: '0 4px 14px rgba(0,0,0,0.04)',
                  display: 'flex',
                  gap: '14px',
                  alignItems: 'center',
                  cursor: 'pointer',
                  position: 'relative',
                  transition: 'transform 0.15s ease, box-shadow 0.15s ease',
                }}
              >
                <div
                  style={{
                    width: '80px',
                    height: '80px',
                    borderRadius: '12px',
                    backgroundColor: '#ffffff',
                    overflow: 'hidden',
                    flexShrink: 0,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    border: '1px solid var(--border-subtle)',
                  }}
                >
                  {p.images && p.images[0] ? (
                    <img
                      src={getImageUrl(p.images[0])}
                      alt={p.name}
                      style={{ width: '100%', height: '100%', objectFit: 'contain', padding: '4px' }}
                    />
                  ) : (
                    <Package size={32} color="var(--text-muted)" style={{ opacity: 0.5 }} />
                  )}
                </div>

                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontWeight: 800, fontSize: '1rem', color: 'var(--text-primary)', lineHeight: 1.25 }}>
                    {p.name}
                  </div>
                  <div style={{ display: 'flex', alignItems: 'baseline', gap: '8px', marginTop: '4px' }}>
                    <span style={{ fontWeight: 900, fontSize: '1.1rem', color: 'var(--color-primary)' }}>
                      ₹{p.price}
                    </span>
                    {(p.compare_price || p.mrp) && (p.compare_price || p.mrp) > p.price && (
                      <span style={{ fontSize: '0.78rem', color: 'var(--text-muted)', textDecoration: 'line-through' }}>
                        ₹{p.compare_price || p.mrp}
                      </span>
                    )}
                  </div>
                  {p.shop_name && (
                    <div style={{ fontSize: '0.76rem', color: 'var(--text-secondary)', marginTop: '4px', display: 'flex', alignItems: 'center', gap: '4px' }}>
                      <Store size={12} color="var(--color-primary)" />
                      <span>{p.shop_name}</span>
                    </div>
                  )}
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', alignItems: 'flex-end' }} onClick={(e) => e.stopPropagation()}>
                  <button
                    onClick={() => toggleSaveProduct(p)}
                    style={{
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      color: '#ef4444',
                      padding: '4px',
                    }}
                    title="Remove from saved"
                  >
                    <Heart size={20} fill="#ef4444" />
                  </button>
                  <button
                    onClick={() => setInspectedProduct(p)}
                    className="btn btn-primary btn-sm"
                    style={{
                      padding: '6px 10px',
                      borderRadius: '6px',
                      fontSize: '0.74rem',
                      fontWeight: 700,
                    }}
                  >
                    View Details
                  </button>
                </div>
              </div>
            ))}
          </div>
        )
      ) : savedShops.length === 0 ? (
        <div className="card" style={{ textAlign: 'center', padding: '40px 20px' }}>
          <Store size={44} color="var(--text-muted)" style={{ margin: '0 auto 12px auto' }} />
          <div style={{ fontWeight: 700, fontSize: '1.05rem', color: 'var(--text-primary)' }}>
            No saved shops
          </div>
          <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginTop: '6px' }}>
            Save your favourite local shops for quick access and direct calls.
          </p>
          <button
            onClick={() => navigate('/')}
            style={{
              marginTop: '16px',
              background: 'var(--color-primary)',
              color: '#fff',
              border: 'none',
              padding: '8px 16px',
              borderRadius: '8px',
              fontWeight: 600,
              fontSize: '0.85rem',
              cursor: 'pointer',
            }}
          >
            Explore Shops
          </button>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {savedShops.map((s) => (
            <div
              key={s.id}
              className="card"
              style={{
                margin: 0,
                padding: '14px',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                  <div
                    style={{
                      width: '46px',
                      height: '46px',
                      borderRadius: '10px',
                      backgroundColor: 'var(--color-primary-light)',
                      overflow: 'hidden',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexShrink: 0,
                    }}
                  >
                    {s.logo_url ? (
                      <img
                        src={getImageUrl(s.logo_url)}
                        alt={s.name}
                        style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                      />
                    ) : (
                      <Store size={22} color="var(--color-primary)" />
                    )}
                  </div>
                  <div>
                    <div style={{ fontWeight: 700, fontSize: '0.95rem' }}>{s.name}</div>
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
                      {s.category || 'General Store'}
                    </div>
                    <div style={{ fontSize: '0.72rem', color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '4px', marginTop: '2px' }}>
                      <MapPin size={11} /> {s.address || s.city || 'Local Area'}
                    </div>
                  </div>
                </div>

                <button
                  onClick={() => toggleSaveShop(s)}
                  style={{
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    color: '#ef4444',
                    padding: '4px',
                  }}
                  title="Remove from saved"
                >
                  <Heart size={18} fill="#ef4444" />
                </button>
              </div>

              {/* Actions */}
              <div style={{ display: 'flex', gap: '8px', marginTop: '12px', paddingTop: '10px', borderTop: '1px solid var(--border-subtle)' }}>
                {s.phone && (
                  <a
                    href={`tel:${s.phone}`}
                    style={{
                      flex: 1,
                      background: 'var(--bg-card)',
                      color: 'var(--text-primary)',
                      border: '1px solid var(--border-subtle)',
                      padding: '6px',
                      borderRadius: '8px',
                      fontSize: '0.75rem',
                      fontWeight: 600,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      gap: '4px',
                      textDecoration: 'none',
                    }}
                  >
                    <Phone size={12} /> Call
                  </a>
                )}
                {s.latitude && s.longitude && (
                  <a
                    href={`https://maps.google.com/?q=${s.latitude},${s.longitude}`}
                    target="_blank"
                    rel="noreferrer"
                    style={{
                      flex: 1,
                      background: 'rgba(59, 130, 246, 0.1)',
                      color: '#3b82f6',
                      padding: '6px',
                      borderRadius: '8px',
                      fontSize: '0.75rem',
                      fontWeight: 600,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      gap: '4px',
                      textDecoration: 'none',
                    }}
                  >
                    <Navigation size={12} /> Directions
                  </a>
                )}
                <button
                  onClick={() => navigate(`/shop/${s.slug}`)}
                  style={{
                    flex: 1.2,
                    background: 'var(--color-primary)',
                    color: '#fff',
                    border: 'none',
                    padding: '6px',
                    borderRadius: '8px',
                    fontSize: '0.75rem',
                    fontWeight: 600,
                    cursor: 'pointer',
                  }}
                >
                  Storefront
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Product Detail Modal */}
      {inspectedProduct && (
        <ProductDetailModal
          product={inspectedProduct}
          onClose={() => setInspectedProduct(null)}
          onReserve={async ({ product, quantity, hold_hours, notes }) => {
            if (!isAuthenticated) {
              alert('Item reserve karne ke liye pehle Login karein');
              navigate('/login');
              return;
            }
            try {
              const res = await reservationApi.createReservation({
                product_id: product.id,
                quantity,
                hold_hours,
                notes,
              });
              alert(`Item Hold Ho Gaya! Pickup Code: ${res.pickup_code || res.reservation_number || 'OK'}`);
              setInspectedProduct(null);
            } catch (err) {
              alert(err.message || 'Reservation fail ho gaya');
            }
          }}
          isMerchant={false}
        />
      )}
    </AppLayout>
  );
};
