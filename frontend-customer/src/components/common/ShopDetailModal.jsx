/**
 * ShopDetailModal Component (Customer Frontend)
 * 
 * Hinglish Hint:
 * Dukan par click karne par khulne wala modern, comprehensive details modal:
 * - High-res dukan photo / promotional banner gallery
 * - Dukan ka logo aur Verified Partner badge
 * - Address, City, Pincode aur Google Maps GPS Directions
 * - Opening/Closing timings, Weekly Off aur Realtime Open/Closed indicator
 * - Call aur WhatsApp Direct Chat buttons
 * - Storefront Browse Products CTA
 * - Private merchant data (e.g. profits, wholesale margins) puri tarah se excluded hai
 */

import React, { useState } from 'react';
import {
  X,
  Store,
  MapPin,
  Clock,
  Phone,
  MessageCircle,
  Navigation,
  ShieldCheck,
  Calendar,
  Sparkles,
  ArrowRight,
  CheckCircle2,
  Share2,
} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { getImageUrl } from '../../utils/imageUrl';

export const ShopDetailModal = ({ shop, onClose }) => {
  const navigate = useNavigate();
  const [copiedLink, setCopiedLink] = useState(false);
  const [selectedBannerIdx, setSelectedBannerIdx] = useState(0);

  if (!shop) return null;

  const banners = Array.isArray(shop.banners) && shop.banners.length > 0 ? shop.banners : [];
  const fullAddress = [shop.address, shop.city, shop.pincode].filter(Boolean).join(', ');

  const handleShareShop = () => {
    const shopUrl = `${window.location.origin}/shop/${shop.slug}`;
    if (navigator.share) {
      navigator
        .share({
          title: shop.name,
          text: `Check out ${shop.name} on ShopMe Local!`,
          url: shopUrl,
        })
        .catch(() => {});
    } else {
      navigator.clipboard.writeText(shopUrl);
      setCopiedLink(true);
      setTimeout(() => setCopiedLink(false), 2000);
    }
  };

  const handleOpenStorefront = () => {
    onClose();
    navigate(`/shop/${shop.slug}`);
  };

  return (
    <div className="modal-backdrop" onClick={onClose} style={{ zIndex: 120 }}>
      <div
        className="bottom-sheet"
        onClick={(e) => e.stopPropagation()}
        style={{
          maxHeight: '92vh',
          display: 'flex',
          flexDirection: 'column',
          padding: 0,
          overflow: 'hidden',
          borderRadius: '24px 24px 0 0',
          backgroundColor: '#ffffff',
          boxShadow: '0 -10px 40px rgba(0,0,0,0.18)',
        }}
      >
        {/* Modal Handle & Action Bar */}
        <div
          style={{
            padding: '12px 18px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            borderBottom: '1px solid var(--border-subtle)',
            backgroundColor: '#ffffff',
          }}
        >
          <div className="sheet-handle" style={{ margin: 0 }} />
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
            <button
              onClick={handleShareShop}
              style={{
                background: '#f1f5f9',
                border: 'none',
                borderRadius: '50%',
                width: '34px',
                height: '34px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: 'pointer',
                color: 'var(--text-secondary)',
              }}
              title="Dukan Share Karein"
            >
              <Share2 size={16} />
            </button>
            <button
              onClick={onClose}
              style={{
                background: '#f1f5f9',
                border: 'none',
                borderRadius: '50%',
                width: '34px',
                height: '34px',
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
        </div>

        {/* Scrollable Body */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '16px' }}>
          {copiedLink && (
            <div
              style={{
                backgroundColor: '#dcfce7',
                color: '#15803d',
                padding: '8px 12px',
                borderRadius: '8px',
                fontSize: '0.8rem',
                fontWeight: 700,
                textAlign: 'center',
                marginBottom: '12px',
              }}
            >
              ✓ Dukan ka link clipboard par copy ho gaya!
            </div>
          )}

          {/* Shop Hero Banner & Photos */}
          <div
            style={{
              position: 'relative',
              borderRadius: '16px',
              overflow: 'hidden',
              backgroundColor: '#1e1b4b',
              minHeight: '140px',
              marginBottom: '16px',
              boxShadow: '0 6px 20px rgba(0,0,0,0.08)',
            }}
          >
            {banners.length > 0 ? (
              <img
                src={getImageUrl(banners[selectedBannerIdx] || banners[0])}
                alt={shop.name}
                style={{ width: '100%', height: '180px', objectFit: 'cover' }}
                onError={(e) => {
                  e.currentTarget.style.display = 'none';
                }}
              />
            ) : (
              <div
                style={{
                  height: '140px',
                  background: 'linear-gradient(135deg, #1e1b4b 0%, #312e81 100%)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: 'rgba(255,255,255,0.2)',
                }}
              >
                <Store size={64} />
              </div>
            )}

            {/* Thumbnail dots if multiple banners */}
            {banners.length > 1 && (
              <div
                style={{
                  position: 'absolute',
                  bottom: '10px',
                  left: '50%',
                  transform: 'translateX(-50%)',
                  display: 'flex',
                  gap: '6px',
                  backgroundColor: 'rgba(0,0,0,0.5)',
                  padding: '4px 8px',
                  borderRadius: '12px',
                }}
              >
                {banners.map((_, i) => (
                  <button
                    key={i}
                    onClick={() => setSelectedBannerIdx(i)}
                    style={{
                      width: selectedBannerIdx === i ? '16px' : '6px',
                      height: '6px',
                      borderRadius: '3px',
                      backgroundColor: selectedBannerIdx === i ? '#ffffff' : 'rgba(255,255,255,0.4)',
                      border: 'none',
                      cursor: 'pointer',
                      padding: 0,
                      transition: 'all 0.2s ease',
                    }}
                  />
                ))}
              </div>
            )}
          </div>

          {/* Shop Profile Header */}
          <div style={{ display: 'flex', gap: '14px', alignItems: 'flex-start', marginBottom: '16px' }}>
            <div
              style={{
                width: '76px',
                height: '76px',
                borderRadius: '18px',
                backgroundColor: 'var(--color-primary-light)',
                color: 'var(--color-primary)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
                overflow: 'hidden',
                border: '2px solid var(--border-subtle)',
                boxShadow: '0 4px 14px rgba(0,0,0,0.08)',
              }}
            >
              {shop.logo_url ? (
                <img
                  src={getImageUrl(shop.logo_url)}
                  alt={shop.name}
                  style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                  onError={(e) => {
                    e.currentTarget.style.display = 'none';
                  }}
                />
              ) : (
                <Store size={36} />
              )}
            </div>

            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', alignItems: 'center' }}>
                <span
                  style={{
                    fontSize: '0.72rem',
                    fontWeight: 700,
                    color: '#2563eb',
                    backgroundColor: 'rgba(37, 99, 235, 0.1)',
                    padding: '2px 8px',
                    borderRadius: '6px',
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '4px',
                  }}
                >
                  <ShieldCheck size={12} /> Verified ShopMe Partner
                </span>
                <span
                  style={{
                    fontSize: '0.72rem',
                    fontWeight: 700,
                    backgroundColor: shop.is_active ? 'rgba(16, 185, 129, 0.15)' : 'rgba(239, 68, 68, 0.15)',
                    color: shop.is_active ? '#065f46' : '#991b1b',
                    padding: '2px 8px',
                    borderRadius: '6px',
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '5px',
                  }}
                >
                  <span
                    style={{
                      width: '6px',
                      height: '6px',
                      borderRadius: '50%',
                      backgroundColor: shop.is_active ? '#10b981' : '#ef4444',
                    }}
                  />
                  {shop.is_active ? 'OPEN ABHI' : 'BAND HAI'}
                </span>
              </div>

              <h2 style={{ fontSize: '1.35rem', fontWeight: 800, margin: '6px 0 2px 0', color: 'var(--text-primary)' }}>
                {shop.name}
              </h2>
              <div style={{ fontSize: '0.82rem', fontWeight: 600, color: 'var(--text-secondary)' }}>
                🏪 {shop.category || 'General Store'} • {shop.city || 'Local Area'}
              </div>
            </div>
          </div>

          {/* Action Buttons: Direct Call & WhatsApp */}
          <div style={{ display: 'flex', gap: '10px', marginBottom: '18px' }}>
            {shop.phone && (
              <a
                href={`tel:${shop.phone}`}
                className="btn btn-secondary"
                style={{
                  flex: 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '6px',
                  fontSize: '0.85rem',
                  fontWeight: 700,
                  textDecoration: 'none',
                  padding: '10px',
                  borderRadius: '12px',
                }}
              >
                <Phone size={16} />
                <span>Call Shop</span>
              </a>
            )}

            {shop.phone && (
              <a
                href={`https://wa.me/91${shop.phone.replace(/[^0-9]/g, '')}?text=Namaste%20${encodeURIComponent(shop.name)}%2C%20ShopMe%20se%20sampark%20kar%20raha%20hu.`}
                target="_blank"
                rel="noreferrer"
                className="btn"
                style={{
                  flex: 1.2,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '6px',
                  fontSize: '0.85rem',
                  fontWeight: 700,
                  textDecoration: 'none',
                  padding: '10px',
                  borderRadius: '12px',
                  backgroundColor: '#25d366',
                  color: '#ffffff',
                }}
              >
                <MessageCircle size={16} />
                <span>WhatsApp Chat</span>
              </a>
            )}

            {shop.latitude && shop.longitude && (
              <a
                href={`https://maps.google.com/?q=${shop.latitude},${shop.longitude}`}
                target="_blank"
                rel="noreferrer"
                className="btn btn-secondary"
                style={{
                  flex: 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '6px',
                  fontSize: '0.85rem',
                  fontWeight: 700,
                  textDecoration: 'none',
                  padding: '10px',
                  borderRadius: '12px',
                }}
              >
                <Navigation size={16} />
                <span>Directions</span>
              </a>
            )}
          </div>

          {/* Description Section if available */}
          {shop.description && (
            <div
              style={{
                backgroundColor: '#f8fafc',
                border: '1px solid var(--border-subtle)',
                borderRadius: '14px',
                padding: '14px',
                marginBottom: '16px',
              }}
            >
              <div
                style={{
                  fontSize: '0.74rem',
                  fontWeight: 800,
                  color: 'var(--text-secondary)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.5px',
                  marginBottom: '6px',
                }}
              >
                ABOUT DUKAN
              </div>
              <p style={{ margin: 0, fontSize: '0.86rem', color: 'var(--text-primary)', lineHeight: 1.5 }}>
                {shop.description}
              </p>
            </div>
          )}

          {/* Timings & Working Days Card */}
          <div
            style={{
              backgroundColor: '#f8fafc',
              border: '1px solid var(--border-subtle)',
              borderRadius: '14px',
              padding: '14px',
              marginBottom: '16px',
            }}
          >
            <div
              style={{
                fontSize: '0.74rem',
                fontWeight: 800,
                color: 'var(--text-secondary)',
                textTransform: 'uppercase',
                letterSpacing: '0.5px',
                marginBottom: '10px',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
              }}
            >
              <Clock size={14} color="var(--color-primary)" />
              DUKAN TIMINGS & SCHEDULE
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '10px' }}>
              <div style={{ backgroundColor: '#ffffff', padding: '10px', borderRadius: '10px', border: '1px solid #e2e8f0' }}>
                <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)', fontWeight: 700 }}>
                  OPENING TIME
                </div>
                <div style={{ fontSize: '0.92rem', fontWeight: 800, color: 'var(--text-primary)', marginTop: '2px' }}>
                  {shop.opening_time || '09:00 AM'}
                </div>
              </div>

              <div style={{ backgroundColor: '#ffffff', padding: '10px', borderRadius: '10px', border: '1px solid #e2e8f0' }}>
                <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)', fontWeight: 700 }}>
                  CLOSING TIME
                </div>
                <div style={{ fontSize: '0.92rem', fontWeight: 800, color: 'var(--text-primary)', marginTop: '2px' }}>
                  {shop.closing_time || '09:00 PM'}
                </div>
              </div>

              <div style={{ gridColumn: '1 / -1', backgroundColor: '#ffffff', padding: '10px', borderRadius: '10px', border: '1px solid #e2e8f0' }}>
                <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)', fontWeight: 700 }}>
                  WEEKLY OFF / CHHUTTI
                </div>
                <div style={{ fontSize: '0.85rem', fontWeight: 700, color: shop.weekly_off ? '#b91c1c' : '#15803d', marginTop: '2px' }}>
                  {shop.weekly_off ? `Band Rehta Hai: ${shop.weekly_off}` : 'Saare 7 Din Khuli Rehti Hai (Open All Days)'}
                </div>
              </div>
            </div>
          </div>

          {/* Physical Address & Location Card */}
          <div
            style={{
              backgroundColor: '#f8fafc',
              border: '1px solid var(--border-subtle)',
              borderRadius: '14px',
              padding: '14px',
              marginBottom: '16px',
            }}
          >
            <div
              style={{
                fontSize: '0.74rem',
                fontWeight: 800,
                color: 'var(--text-secondary)',
                textTransform: 'uppercase',
                letterSpacing: '0.5px',
                marginBottom: '8px',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
              }}
            >
              <MapPin size={14} color="var(--color-primary)" />
              DUKAN KA PATA & LOCATION
            </div>

            <div style={{ fontSize: '0.88rem', fontWeight: 700, color: 'var(--text-primary)', lineHeight: 1.4 }}>
              {fullAddress || 'Address details currently not listed.'}
            </div>

            {shop.latitude && shop.longitude && (
              <div style={{ marginTop: '8px', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                GPS: {shop.latitude.toFixed(4)}, {shop.longitude.toFixed(4)}
              </div>
            )}
          </div>

          {/* Customer Guarantees / Shopping Benefits */}
          <div
            style={{
              backgroundColor: '#eff6ff',
              border: '1px solid #bfdbfe',
              borderRadius: '14px',
              padding: '14px',
              marginBottom: '16px',
            }}
          >
            <div style={{ fontSize: '0.75rem', fontWeight: 800, color: '#1e40af', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '6px' }}>
              <Sparkles size={14} /> SHOPME CUSTOMER ASSURANCE
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', fontSize: '0.78rem', color: '#1e3a8a' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <CheckCircle2 size={14} color="#2563eb" />
                <span>30-minute Counter Pickup: Hold item and pick directly</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <CheckCircle2 size={14} color="#2563eb" />
                <span>In-Store Inspection: Check product before payment</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <CheckCircle2 size={14} color="#2563eb" />
                <span>Multiple Payments: Cash, UPI, GooglePay, PhonePe accepted</span>
              </div>
            </div>
          </div>
        </div>

        {/* Sticky Bottom CTA: Visit Storefront */}
        <div
          style={{
            padding: '14px 18px',
            borderTop: '1px solid var(--border-subtle)',
            backgroundColor: '#ffffff',
            boxShadow: '0 -4px 12px rgba(0,0,0,0.05)',
          }}
        >
          <button
            onClick={handleOpenStorefront}
            className="btn btn-primary btn-block btn-lg"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              fontWeight: 800,
              fontSize: '0.98rem',
              borderRadius: '14px',
            }}
          >
            <span>Dukan Ke Sare Products Dekhein</span>
            <ArrowRight size={18} />
          </button>
        </div>
      </div>
    </div>
  );
};
