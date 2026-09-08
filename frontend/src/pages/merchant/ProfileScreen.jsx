/**
 * Dedicated User Profile & Account Management Screen
 * 
 * Hinglish Hint:
 * Dukaandar ya Grahak ka personal profile page:
 * - User Details (Naam, Phone, Email, Role)
 * - Dukan Status & Settings (Online/Offline toggle, QR Code)
 * - Loyalty Points & Passbook
 * - Logout & Account management
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  User,
  Phone,
  Mail,
  Store,
  QrCode,
  Power,
  Shield,
  Award,
  ExternalLink,
  LogOut,
  ChevronRight,
  Sparkles,
  HelpCircle,
  Camera,
  Image,
  Upload,
} from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { shopApi } from '../../api/shop.api';
import { uploadApi } from '../../api/upload.api';
import { AppLayout } from '../../components/layout/AppLayout';

export const ProfileScreen = () => {
  const navigate = useNavigate();
  const { user, shop, refreshShop, logout, isMerchant, updateUser, updateShopState } = useAuth();
  const [showQR, setShowQR] = useState(false);

  // Avatar upload state
  const [avatarUploading, setAvatarUploading] = useState(false);
  const [avatarError, setAvatarError] = useState('');

  // Shop images upload state
  const [shopLogoFile, setShopLogoFile] = useState(null);
  const [shopLogoPreview, setShopLogoPreview] = useState('');
  const [shopBannerFiles, setShopBannerFiles] = useState([]);
  const [shopBannerPreviews, setShopBannerPreviews] = useState([]);
  const [shopImagesUploading, setShopImagesUploading] = useState(false);
  const [shopImagesSuccess, setShopImagesSuccess] = useState('');

  // Handle avatar upload
  const handleAvatarChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate size (max 2MB)
    if (file.size > 2 * 1024 * 1024) {
      alert('Photo ka size 2MB se kam hona chahiye');
      return;
    }

    try {
      setAvatarUploading(true);
      setAvatarError('');
      const data = await uploadApi.uploadUserAvatar(file);
      updateUser({ avatar_url: data.avatar_url });
      alert('Profile photo safaltapoorvak update ho gayi!');
    } catch (err) {
      console.error('Avatar upload error:', err);
      alert('Avatar upload nahi ho saka: ' + (err.message || 'Error'));
    } finally {
      setAvatarUploading(false);
    }
  };

  // Handle shop logo selection
  const handleLogoSelect = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 2 * 1024 * 1024) {
      alert('Logo size 2MB se kam hona chahiye');
      return;
    }
    setShopLogoFile(file);
    setShopLogoPreview(URL.createObjectURL(file));
  };

  // Handle shop banners selection (up to 2)
  const handleBannersSelect = (e) => {
    const files = Array.from(e.target.files || []).slice(0, 2);
    if (files.length === 0) return;
    setShopBannerFiles(files);
    setShopBannerPreviews(files.map((f) => URL.createObjectURL(f)));
  };

  // Upload shop branding
  const handleUploadShopBranding = async (e) => {
    e.preventDefault();
    if (!shopLogoFile && shopBannerFiles.length === 0) {
      alert('Kripya kam se kam ek Logo ya Banner select karein');
      return;
    }

    try {
      setShopImagesUploading(true);
      setShopImagesSuccess('');
      const res = await uploadApi.uploadShopImages({
        logo: shopLogoFile,
        banners: shopBannerFiles,
      });

      updateShopState({
        logo_url: res.logo_url || shop.logo_url,
        banners: res.banners || shop.banners,
      });

      setShopImagesSuccess('Dukan ki photos update ho gayi!');
      setShopLogoFile(null);
      setShopBannerFiles([]);
      setTimeout(() => setShopImagesSuccess(''), 4000);
    } catch (err) {
      alert('Dukan photos upload error: ' + (err.message || 'Error'));
    } finally {
      setShopImagesUploading(false);
    }
  };

  // Toggle shop status
  const handleToggleShopStatus = async () => {
    try {
      await shopApi.toggleShopStatus();
      await refreshShop();
    } catch (err) {
      alert('Status change error: ' + err.message);
    }
  };

  return (
    <AppLayout title="Aapka Profile" subtitle="Account & Dukan Settings">
      {/* User Hero Card */}
      <div
        className="card"
        style={{
          background: 'linear-gradient(135deg, #1e1b4b 0%, #312e81 100%)',
          color: '#ffffff',
          border: 'none',
          padding: '20px',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          {/* Avatar with Camera Button */}
          <div style={{ position: 'relative' }}>
            <div
              style={{
                width: '64px',
                height: '64px',
                borderRadius: '50%',
                background: user?.avatar_url
                  ? 'transparent'
                  : 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)',
                color: '#ffffff',
                fontSize: '1.5rem',
                fontWeight: 900,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                boxShadow: '0 4px 14px rgba(0, 0, 0, 0.3)',
                flexShrink: 0,
                overflow: 'hidden',
                border: '2px solid rgba(255, 255, 255, 0.3)',
              }}
            >
              {user?.avatar_url ? (
                <img
                  src={user.avatar_url}
                  alt={user.full_name}
                  style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                />
              ) : (
                user?.full_name?.charAt(0)?.toUpperCase() || 'U'
              )}
            </div>

            {/* Camera Upload Button */}
            <label
              htmlFor="avatar-file-input"
              style={{
                position: 'absolute',
                bottom: '-4px',
                right: '-4px',
                width: '26px',
                height: '26px',
                borderRadius: '50%',
                backgroundColor: 'var(--color-primary)',
                color: '#ffffff',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: 'pointer',
                boxShadow: '0 2px 6px rgba(0,0,0,0.4)',
                border: '2px solid #ffffff',
              }}
              title="Profile Photo Badlein"
            >
              <Camera size={13} />
            </label>
            <input
              id="avatar-file-input"
              type="file"
              accept="image/jpeg,image/png,image/webp"
              onChange={handleAvatarChange}
              style={{ display: 'none' }}
              disabled={avatarUploading}
            />
          </div>

          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <h2 style={{ fontSize: '1.2rem', fontWeight: 800, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {user?.full_name || 'Dukaandar'}
              </h2>
            </div>
            <div style={{ fontSize: '0.8rem', opacity: 0.85, marginTop: '2px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {user?.email}
            </div>
            {avatarUploading && (
              <div style={{ fontSize: '0.72rem', color: '#fde047', marginTop: '3px' }}>
                Photo upload ho rahi hai...
              </div>
            )}
            <div style={{ marginTop: '6px' }}>
              <span
                style={{
                  backgroundColor: 'rgba(255, 255, 255, 0.2)',
                  color: '#ffffff',
                  fontSize: '0.72rem',
                  fontWeight: 700,
                  padding: '3px 8px',
                  borderRadius: 'var(--radius-full)',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '4px',
                }}
              >
                <Shield size={12} />
                {isMerchant ? 'Verified Merchant (Dukaandar)' : 'Customer (Grahak)'}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Shop Branding & Photos Card (If Merchant) */}
      {shop && (
        <div className="card" style={{ padding: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '12px' }}>
            <Image size={18} color="var(--color-primary)" />
            <span style={{ fontWeight: 800, fontSize: '0.95rem' }}>Dukan Ki Photos & Branding</span>
          </div>

          {shopImagesSuccess && (
            <div style={{ backgroundColor: '#dcfce7', color: '#15803d', padding: '8px 12px', borderRadius: 'var(--radius-sm)', fontSize: '0.82rem', marginBottom: '12px', fontWeight: 600 }}>
              ✓ {shopImagesSuccess}
            </div>
          )}

          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
            {/* Logo Picker */}
            <div>
              <div style={{ fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-secondary)', marginBottom: '6px' }}>
                DUKAN KA LOGO
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <div
                  style={{
                    width: '54px',
                    height: '54px',
                    borderRadius: 'var(--radius-md)',
                    border: '1.5px dashed var(--border-strong)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    overflow: 'hidden',
                    backgroundColor: '#f8fafc',
                    flexShrink: 0,
                  }}
                >
                  {shopLogoPreview ? (
                    <img src={shopLogoPreview} alt="New Logo" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                  ) : shop.logo_url ? (
                    <img src={shop.logo_url} alt={shop.name} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                  ) : (
                    <Store size={24} color="var(--text-muted)" />
                  )}
                </div>

                <div style={{ flex: 1 }}>
                  <label
                    htmlFor="shop-logo-input"
                    className="btn btn-secondary btn-sm"
                    style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
                  >
                    <Upload size={14} /> {shop.logo_url || shopLogoFile ? 'Logo Badlein' : 'Logo Upload Karein'}
                  </label>
                  <input
                    id="shop-logo-input"
                    type="file"
                    accept="image/jpeg,image/png,image/webp"
                    onChange={handleLogoSelect}
                    style={{ display: 'none' }}
                  />
                  <div style={{ fontSize: '0.72rem', color: 'var(--text-muted)', marginTop: '4px' }}>
                    Square size (JPEG/PNG, max 2MB)
                  </div>
                </div>
              </div>
            </div>

            {/* Banners Picker */}
            <div>
              <div style={{ fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-secondary)', marginBottom: '6px' }}>
                PROMOTIONAL BANNERS (MAX 2)
              </div>
              <div style={{ display: 'flex', gap: '8px', marginBottom: '8px', flexWrap: 'wrap' }}>
                {shopBannerPreviews.length > 0 ? (
                  shopBannerPreviews.map((bp, idx) => (
                    <img
                      key={idx}
                      src={bp}
                      alt="Banner Preview"
                      style={{ width: '120px', height: '60px', objectFit: 'cover', borderRadius: 'var(--radius-sm)', border: '1px solid var(--border-subtle)' }}
                    />
                  ))
                ) : shop.banners && shop.banners.length > 0 ? (
                  shop.banners.map((b, idx) => (
                    <img
                      key={idx}
                      src={b}
                      alt={`Shop Banner ${idx + 1}`}
                      style={{ width: '120px', height: '60px', objectFit: 'cover', borderRadius: 'var(--radius-sm)', border: '1px solid var(--border-subtle)' }}
                    />
                  ))
                ) : (
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                    Koi promotional banner upload nahi kiya gaya hai.
                  </div>
                )}
              </div>

              <label
                htmlFor="shop-banners-input"
                className="btn btn-secondary btn-sm"
                style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
              >
                <Upload size={14} /> Naye Banners Chunein (Max 2)
              </label>
              <input
                id="shop-banners-input"
                type="file"
                multiple
                accept="image/jpeg,image/png,image/webp"
                onChange={handleBannersSelect}
                style={{ display: 'none' }}
              />
            </div>

            {/* Save Button if files selected */}
            {(shopLogoFile || shopBannerFiles.length > 0) && (
              <button
                onClick={handleUploadShopBranding}
                className="btn btn-primary btn-sm btn-block"
                disabled={shopImagesUploading}
                style={{ marginTop: '6px' }}
              >
                {shopImagesUploading ? 'Photos Upload Ho Rahi Hain...' : '✓ Nayi Photos Save Karein'}
              </button>
            )}
          </div>
        </div>
      )}

      {/* Dukan Details (If Merchant) */}
      {shop && (
        <div className="card" style={{ padding: '16px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '14px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <Store size={20} color="var(--color-primary)" />
              <span style={{ fontWeight: 800, fontSize: '0.95rem' }}>Dukan Ki Jankari</span>
            </div>

            <button
              onClick={handleToggleShopStatus}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '5px',
                backgroundColor: shop.is_active ? 'var(--color-success-light)' : 'var(--color-danger-light)',
                border: `1px solid ${shop.is_active ? '#10b981' : '#ef4444'}`,
                color: shop.is_active ? '#065f46' : '#991b1b',
                padding: '4px 10px',
                borderRadius: 'var(--radius-full)',
                fontSize: '0.75rem',
                fontWeight: 700,
                cursor: 'pointer',
              }}
            >
              <Power size={13} />
              <span>{shop.is_active ? 'Online' : 'Offline'}</span>
            </button>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', fontSize: '0.85rem' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Dukan Ka Naam:</span>
              <span style={{ fontWeight: 700 }}>{shop.name}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Category:</span>
              <span style={{ fontWeight: 600 }}>{shop.category}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Pata (Address):</span>
              <span style={{ fontWeight: 600, textAlign: 'right', maxWidth: '60%' }}>{shop.address}</span>
            </div>
          </div>

          <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
            <a
              href={`/shop/${shop.slug}`}
              target="_blank"
              rel="noreferrer"
              className="btn btn-secondary btn-sm"
              style={{ flex: 1, textDecoration: 'none' }}
            >
              <ExternalLink size={14} /> Storefront
            </a>
            <button
              onClick={() => setShowQR(true)}
              className="btn btn-secondary btn-sm"
              style={{ flex: 1 }}
            >
              <QrCode size={14} /> Shop QR
            </button>
          </div>
        </div>
      )}

      {/* Loyalty & Rewards Card */}
      <div
        className="card"
        style={{
          background: 'linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%)',
          border: '1px solid #fde68a',
          padding: '14px',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div
              style={{
                width: '40px',
                height: '40px',
                borderRadius: '50%',
                backgroundColor: '#f59e0b',
                color: 'white',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Award size={22} />
            </div>
            <div>
              <div style={{ fontWeight: 800, fontSize: '0.9rem', color: '#92400e' }}>
                Loyalty Rewards & Badges
              </div>
              <div style={{ fontSize: '0.75rem', color: '#b45309' }}>
                Counter POS aur Grahak billing rewards
              </div>
            </div>
          </div>
          <span
            style={{
              backgroundColor: '#92400e',
              color: '#ffffff',
              fontWeight: 800,
              fontSize: '0.8rem',
              padding: '4px 8px',
              borderRadius: 'var(--radius-sm)',
            }}
          >
            Active
          </span>
        </div>
      </div>

      {/* Account Actions */}
      <div className="card" style={{ padding: '8px 12px' }}>
        <div
          className="list-item card-clickable"
          onClick={() => alert('Support Helpline: +91 9876543210 (WhatsApp available)')}
          style={{ cursor: 'pointer' }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <HelpCircle size={20} color="var(--color-primary)" />
            <span style={{ fontWeight: 600, fontSize: '0.9rem' }}>Madad & Support (Help)</span>
          </div>
          <ChevronRight size={18} color="var(--text-muted)" />
        </div>

        <div
          className="list-item card-clickable"
          onClick={logout}
          style={{ cursor: 'pointer', borderBottom: 'none' }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <LogOut size={20} color="var(--color-danger)" />
            <span style={{ fontWeight: 700, fontSize: '0.9rem', color: 'var(--color-danger)' }}>
              Account Se Logout Karein
            </span>
          </div>
          <ChevronRight size={18} color="var(--color-danger)" />
        </div>
      </div>

      {/* Shop QR Modal */}
      {showQR && (
        <div className="modal-backdrop" onClick={() => setShowQR(false)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <div style={{ textAlign: 'center', padding: '10px' }}>
              <h3 style={{ fontSize: '1.2rem', fontWeight: 800 }}>{shop?.name} QR</h3>
              <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
                Grahak isko scan karke aapki dukan dekh sakte hain
              </p>
              <div
                style={{
                  width: '180px',
                  height: '180px',
                  margin: '0 auto',
                  backgroundColor: 'var(--bg-surface-subtle)',
                  borderRadius: '16px',
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '8px',
                  border: '2px dashed var(--color-primary)',
                }}
              >
                <QrCode size={90} color="var(--color-primary)" />
                <span style={{ fontSize: '0.8rem', fontWeight: 700 }}>/{shop?.slug}</span>
              </div>
              <button
                className="btn btn-secondary btn-block"
                style={{ marginTop: '20px' }}
                onClick={() => setShowQR(false)}
              >
                Band Karein
              </button>
            </div>
          </div>
        </div>
      )}
    </AppLayout>
  );
};
