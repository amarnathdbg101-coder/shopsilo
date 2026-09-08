/**
 * EditShopModal Component
 * 
 * Hinglish Hint:
 * Dukaandar apni dukan ki saari details (naam, category, pata, shahar, timing,
 * phone, WhatsApp, GPS location, logo aur promotional banners) yahan se update kar sakta hai.
 */

import React, { useState } from 'react';
import {
  X,
  Store,
  MapPin,
  Clock,
  Phone,
  MessageCircle,
  Upload,
  Crosshair,
  CheckCircle,
  Layers,
  Image as ImageIcon,
} from 'lucide-react';
import { shopApi } from '../../api/shop.api';
import { uploadApi } from '../../api/upload.api';
import { useAuth } from '../../context/AuthContext';
import { getImageUrl } from '../../utils/imageUrl';

const SHOP_CATEGORIES = [
  'General Store / Kirana',
  'Kirana & Grocery',
  'Electronics & Mobile',
  'Clothing & Fashion',
  'Pharmacy / Medical',
  'Bakery & Dairy',
  'Hardware & Tools',
  'Beauty & Cosmetics',
  'Footwear & Bags',
  'Stationery & Books',
  'Restaurant & Cafe',
  'Automobile & Garage',
  'Other Local Business',
];

const WEEKLY_OFF_OPTIONS = [
  { label: 'Har Din Khula (No Off)', value: '' },
  { label: 'Sunday (Ravivar)', value: 'Sunday' },
  { label: 'Monday (Somvar)', value: 'Monday' },
  { label: 'Tuesday (Mangalvar)', value: 'Tuesday' },
  { label: 'Wednesday (Budhvar)', value: 'Wednesday' },
  { label: 'Thursday (Guruvar)', value: 'Thursday' },
  { label: 'Friday (Shukravar)', value: 'Friday' },
  { label: 'Saturday (Shanivar)', value: 'Saturday' },
];

export const EditShopModal = ({ isOpen, onClose, onUpdated }) => {
  const { shop, updateShopState, refreshShop } = useAuth();
  if (!isOpen || !shop) return null;

  const [activeTab, setActiveTab] = useState('basic'); // 'basic' | 'location' | 'timings' | 'branding'

  // Form State
  const [formData, setFormData] = useState({
    name: shop.name || '',
    category: shop.category || 'Kirana & Grocery',
    description: shop.description || '',
    address: shop.address || '',
    city: shop.city || '',
    pincode: shop.pincode || '',
    phone: shop.phone || '',
    whatsapp_number: shop.whatsapp_number || shop.phone || '',
    opening_time: shop.opening_time || '09:00',
    closing_time: shop.closing_time || '21:00',
    weekly_off: shop.weekly_off || '',
    latitude: shop.latitude != null ? shop.latitude : 26.1542,
    longitude: shop.longitude != null ? shop.longitude : 85.8918,
    is_active: shop.is_active !== undefined ? shop.is_active : true,
  });

  // Images state
  const [logoFile, setLogoFile] = useState(null);
  const [logoPreview, setLogoPreview] = useState('');
  const [bannerFiles, setBannerFiles] = useState([]);
  const [bannerPreviews, setBannerPreviews] = useState([]);

  // Loading & error
  const [saving, setSaving] = useState(false);
  const [detectingGps, setDetectingGps] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  // Handle Logo Select
  const handleLogoChange = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 2 * 1024 * 1024) {
      alert('Logo photo 2MB se choti honi chahiye');
      return;
    }
    setLogoFile(file);
    setLogoPreview(URL.createObjectURL(file));
  };

  // Handle Banners Select (up to 2)
  const handleBannersChange = (e) => {
    const files = Array.from(e.target.files || []).slice(0, 2);
    if (files.length === 0) return;
    for (const f of files) {
      if (f.size > 2 * 1024 * 1024) {
        alert(`"${f.name}" 2MB se zyada hai. Kripya 2MB se choti photo chunein.`);
        return;
      }
    }
    setBannerFiles(files);
    setBannerPreviews(files.map((f) => URL.createObjectURL(f)));
  };

  // GPS Auto-detection
  const handleDetectGPS = () => {
    if (!navigator.geolocation) {
      alert('Aapke browser me GPS support nahi hai');
      return;
    }
    setDetectingGps(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setFormData((prev) => ({
          ...prev,
          latitude: parseFloat(pos.coords.latitude.toFixed(6)),
          longitude: parseFloat(pos.coords.longitude.toFixed(6)),
        }));
        setDetectingGps(false);
        alert('GPS location detect ho gayi!');
      },
      (err) => {
        setDetectingGps(false);
        alert('GPS location nahi mil saki: ' + err.message);
      },
      { timeout: 8000, enableHighAccuracy: true }
    );
  };

  // Form Submit
  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setSaving(true);

    try {
      let finalLogoUrl = shop.logo_url;
      let finalBanners = shop.banners || [];

      // 1. Upload new branding images if selected
      if (logoFile || bannerFiles.length > 0) {
        try {
          const uploadRes = await uploadApi.uploadShopImages({
            logo: logoFile,
            banners: bannerFiles,
          });
          if (uploadRes.logo_url) finalLogoUrl = uploadRes.logo_url;
          if (uploadRes.banners && uploadRes.banners.length > 0) finalBanners = uploadRes.banners;
        } catch (uploadErr) {
          console.warn('Shop images upload warning:', uploadErr);
          // If upload fails, alert user but allow saving text details if they confirm
          if (!window.confirm('Photos upload nahi ho sakin (' + (uploadErr.message || 'Error') + '). Kya aap baaki details update karna chahte hain?')) {
            setSaving(false);
            return;
          }
        }
      }

      // 2. Prepare payload for PUT /shops/me
      const updatePayload = {
        name: formData.name.trim(),
        category: formData.category,
        description: formData.description.trim(),
        address: formData.address.trim(),
        city: formData.city.trim(),
        pincode: formData.pincode.trim(),
        phone: formData.phone.trim(),
        whatsapp_number: formData.whatsapp_number.trim(),
        opening_time: formData.opening_time,
        closing_time: formData.closing_time,
        weekly_off: formData.weekly_off,
        latitude: Number(formData.latitude),
        longitude: Number(formData.longitude),
        logo_url: finalLogoUrl,
        banners: finalBanners,
      };

      const updatedShop = await shopApi.updateMyShop(updatePayload);

      // 3. Update application auth state
      updateShopState(updatedShop);
      if (refreshShop) await refreshShop();
      if (onUpdated) onUpdated(updatedShop);

      setSuccess('Dukan ki details safaltapoorvak update ho gayi!');
      setTimeout(() => {
        onClose();
      }, 1200);
    } catch (err) {
      setError(err.message || 'Shop update karne me samasya aayi');
    } finally {
      setSaving(false);
    }
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
        }}
      >
        {/* Modal Header */}
        <div
          style={{
            padding: '14px 18px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            borderBottom: '1px solid var(--border-subtle)',
            backgroundColor: '#ffffff',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Store size={20} color="var(--color-primary)" />
            <h2 style={{ fontSize: '1.15rem', fontWeight: 800, margin: 0 }}>
              Dukan Details Update Karein
            </h2>
          </div>
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
            title="Band karein"
          >
            <X size={18} />
          </button>
        </div>

        {/* Tab Navigation */}
        <div
          style={{
            display: 'flex',
            borderBottom: '1px solid var(--border-subtle)',
            backgroundColor: '#f8fafc',
            overflowX: 'auto',
            scrollbarWidth: 'none',
          }}
        >
          {[
            { id: 'basic', label: '1. Dukan Info' },
            { id: 'location', label: '2. Pata & GPS' },
            { id: 'timings', label: '3. Timings' },
            { id: 'branding', label: '4. Logo & Photos' },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              style={{
                flex: 1,
                padding: '10px 12px',
                border: 'none',
                background: activeTab === tab.id ? '#ffffff' : 'transparent',
                color: activeTab === tab.id ? 'var(--color-primary)' : 'var(--text-secondary)',
                fontWeight: activeTab === tab.id ? 800 : 600,
                fontSize: '0.8rem',
                borderBottom: activeTab === tab.id ? '2.5px solid var(--color-primary)' : 'none',
                cursor: 'pointer',
                whiteSpace: 'nowrap',
              }}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Modal Body Form */}
        <form onSubmit={handleSubmit} style={{ flex: 1, overflowY: 'auto', padding: '16px' }}>
          {error && (
            <div style={{ backgroundColor: '#fee2e2', color: '#b91c1c', padding: '10px 14px', borderRadius: 'var(--radius-sm)', fontSize: '0.82rem', marginBottom: '14px', fontWeight: 600 }}>
              {error}
            </div>
          )}
          {success && (
            <div style={{ backgroundColor: '#dcfce7', color: '#15803d', padding: '10px 14px', borderRadius: 'var(--radius-sm)', fontSize: '0.82rem', marginBottom: '14px', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '6px' }}>
              <CheckCircle size={16} />
              <span>{success}</span>
            </div>
          )}

          {/* TAB 1: BASIC INFO */}
          {activeTab === 'basic' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
              <div className="form-group">
                <label className="form-label">Dukan Ka Naam (Shop Name) *</label>
                <input
                  type="text"
                  required
                  className="form-input"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="e.g. Amarnath Cosmetic Store"
                />
              </div>

              <div className="form-group">
                <label className="form-label">Category *</label>
                <select
                  className="form-select"
                  value={formData.category}
                  onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                >
                  {SHOP_CATEGORIES.map((cat) => (
                    <option key={cat} value={cat}>
                      {cat}
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">Dukan Ka Vivaran (Description)</label>
                <textarea
                  className="form-input"
                  rows={3}
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Aapki dukan me kya-kya milta hai, visheshtaayein..."
                />
              </div>

              <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '8px' }}>
                <button
                  type="button"
                  className="btn btn-secondary btn-sm"
                  onClick={() => setActiveTab('location')}
                >
                  Next: Pata & GPS →
                </button>
              </div>
            </div>
          )}

          {/* TAB 2: LOCATION & CONTACT */}
          {activeTab === 'location' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
              <div className="form-group">
                <label className="form-label">Dukan Ka Pura Pata (Full Address) *</label>
                <input
                  type="text"
                  required
                  className="form-input"
                  value={formData.address}
                  onChange={(e) => setFormData({ ...formData, address: e.target.value })}
                  placeholder="e.g. Shop #12, Tower Chowk, Main Market"
                />
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: '10px' }}>
                <div className="form-group">
                  <label className="form-label">Shahar (City)</label>
                  <input
                    type="text"
                    className="form-input"
                    value={formData.city}
                    onChange={(e) => setFormData({ ...formData, city: e.target.value })}
                    placeholder="e.g. Darbhanga"
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">PIN Code</label>
                  <input
                    type="text"
                    className="form-input"
                    value={formData.pincode}
                    onChange={(e) => setFormData({ ...formData, pincode: e.target.value })}
                    placeholder="846004"
                  />
                </div>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
                <div className="form-group">
                  <label className="form-label">Calling Phone</label>
                  <input
                    type="tel"
                    className="form-input"
                    value={formData.phone}
                    onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                    placeholder="9876543210"
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">WhatsApp Number</label>
                  <input
                    type="tel"
                    className="form-input"
                    value={formData.whatsapp_number}
                    onChange={(e) => setFormData({ ...formData, whatsapp_number: e.target.value })}
                    placeholder="9876543210"
                  />
                </div>
              </div>

              {/* GPS Coordinates Section */}
              <div
                style={{
                  background: 'var(--bg-app)',
                  padding: '12px',
                  borderRadius: 'var(--radius-md)',
                  border: '1px solid var(--border-subtle)',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                  <span style={{ fontSize: '0.8rem', fontWeight: 800 }}>📍 GPS Coordinates (Nearby Graahak khojne ke liye)</span>
                  <button
                    type="button"
                    onClick={handleDetectGPS}
                    disabled={detectingGps}
                    className="btn btn-secondary btn-sm"
                    style={{ fontSize: '0.72rem', padding: '3px 8px' }}
                  >
                    <Crosshair size={12} className={detectingGps ? 'spin' : ''} />
                    <span>{detectingGps ? 'Detecting...' : 'Detect GPS'}</span>
                  </button>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
                  <div>
                    <span style={{ fontSize: '0.7rem', color: 'var(--text-muted)' }}>Latitude</span>
                    <input
                      type="number"
                      step="any"
                      className="form-input"
                      value={formData.latitude}
                      onChange={(e) => setFormData({ ...formData, latitude: e.target.value })}
                    />
                  </div>
                  <div>
                    <span style={{ fontSize: '0.7rem', color: 'var(--text-muted)' }}>Longitude</span>
                    <input
                      type="number"
                      step="any"
                      className="form-input"
                      value={formData.longitude}
                      onChange={(e) => setFormData({ ...formData, longitude: e.target.value })}
                    />
                  </div>
                </div>
              </div>

              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '8px' }}>
                <button
                  type="button"
                  className="btn btn-secondary btn-sm"
                  onClick={() => setActiveTab('basic')}
                >
                  ← Back
                </button>
                <button
                  type="button"
                  className="btn btn-secondary btn-sm"
                  onClick={() => setActiveTab('timings')}
                >
                  Next: Timings →
                </button>
              </div>
            </div>
          )}

          {/* TAB 3: TIMINGS */}
          {activeTab === 'timings' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
                <div className="form-group">
                  <label className="form-label">Khulne Ka Samay (Opens At)</label>
                  <input
                    type="time"
                    className="form-input"
                    value={formData.opening_time}
                    onChange={(e) => setFormData({ ...formData, opening_time: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Band Hone Ka Samay (Closes At)</label>
                  <input
                    type="time"
                    className="form-input"
                    value={formData.closing_time}
                    onChange={(e) => setFormData({ ...formData, closing_time: e.target.value })}
                  />
                </div>
              </div>

              <div className="form-group">
                <label className="form-label">Saptahik Chutti (Weekly Off)</label>
                <select
                  className="form-select"
                  value={formData.weekly_off}
                  onChange={(e) => setFormData({ ...formData, weekly_off: e.target.value })}
                >
                  {WEEKLY_OFF_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              </div>

              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '8px' }}>
                <button
                  type="button"
                  className="btn btn-secondary btn-sm"
                  onClick={() => setActiveTab('location')}
                >
                  ← Back
                </button>
                <button
                  type="button"
                  className="btn btn-secondary btn-sm"
                  onClick={() => setActiveTab('branding')}
                >
                  Next: Logo & Photos →
                </button>
              </div>
            </div>
          )}

          {/* TAB 4: BRANDING & PHOTOS */}
          {activeTab === 'branding' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              {/* Shop Logo Section */}
              <div>
                <label className="form-label" style={{ marginBottom: '6px' }}>
                  Dukan Ka Logo (Square Avatar)
                </label>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <div
                    style={{
                      width: '64px',
                      height: '64px',
                      borderRadius: 'var(--radius-md)',
                      backgroundColor: '#f1f5f9',
                      overflow: 'hidden',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      border: '1.5px solid var(--border-subtle)',
                      flexShrink: 0,
                    }}
                  >
                    {logoPreview ? (
                      <img src={logoPreview} alt="Logo Preview" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                    ) : shop.logo_url ? (
                      <img src={getImageUrl(shop.logo_url)} alt={shop.name} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                    ) : (
                      <Store size={28} color="var(--text-muted)" />
                    )}
                  </div>

                  <div>
                    <label
                      htmlFor="modal-shop-logo"
                      className="btn btn-secondary btn-sm"
                      style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '5px' }}
                    >
                      <Upload size={14} /> Naya Logo Chunein
                    </label>
                    <input
                      id="modal-shop-logo"
                      type="file"
                      accept="image/jpeg,image/png,image/webp"
                      onChange={handleLogoChange}
                      style={{ display: 'none' }}
                    />
                    <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)', marginTop: '4px' }}>
                      JPEG/PNG, Max 2MB
                    </div>
                  </div>
                </div>
              </div>

              {/* Promotional Banners Section */}
              <div>
                <label className="form-label" style={{ marginBottom: '6px' }}>
                  Dukan Ke Banners (Max 2 Photos)
                </label>
                <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', marginBottom: '8px' }}>
                  {bannerPreviews.length > 0 ? (
                    bannerPreviews.map((p, i) => (
                      <img
                        key={i}
                        src={p}
                        alt="New Banner"
                        style={{ width: '130px', height: '65px', borderRadius: 'var(--radius-sm)', objectFit: 'cover', border: '1.5px solid var(--color-primary)' }}
                      />
                    ))
                  ) : shop.banners && shop.banners.length > 0 ? (
                    shop.banners.map((b, i) => (
                      <img
                        key={i}
                        src={getImageUrl(b)}
                        alt="Current Banner"
                        style={{ width: '130px', height: '65px', borderRadius: 'var(--radius-sm)', objectFit: 'cover', border: '1px solid var(--border-subtle)' }}
                      />
                    ))
                  ) : (
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-muted)' }}>
                      Koi banner photo nahi hai.
                    </div>
                  )}
                </div>

                <label
                  htmlFor="modal-shop-banners"
                  className="btn btn-secondary btn-sm"
                  style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '5px' }}
                >
                  <Upload size={14} /> Banners Chunein (Max 2)
                </label>
                <input
                  id="modal-shop-banners"
                  type="file"
                  multiple
                  accept="image/jpeg,image/png,image/webp"
                  onChange={handleBannersChange}
                  style={{ display: 'none' }}
                />
              </div>

              <div style={{ display: 'flex', justifyContent: 'flex-start', marginTop: '8px' }}>
                <button
                  type="button"
                  className="btn btn-secondary btn-sm"
                  onClick={() => setActiveTab('timings')}
                >
                  ← Back: Timings
                </button>
              </div>
            </div>
          )}

          {/* Submit Action Button */}
          <div style={{ marginTop: '20px', paddingTop: '14px', borderTop: '1px solid var(--border-subtle)' }}>
            <button
              type="submit"
              disabled={saving}
              className="btn btn-primary btn-block btn-lg"
              style={{ fontWeight: 800 }}
            >
              {saving ? 'Dukan Update Ho Rahi Hai...' : '✓ Dukan Ki Jankari Save Karein'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
