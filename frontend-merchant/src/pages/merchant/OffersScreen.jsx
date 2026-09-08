/**
 * Merchant Offers & Promotions Manager
 * 
 * Ported from Flutter APK (QuickPick):
 * - Create & publish deals (Flat %, Buy 1 Get 1, Buy X Get Y, Festive discounts)
 * - Live status in Customer Deals feed
 * - Offer views & engagement insights
 */

import React, { useState, useEffect } from 'react';
import {
  Tag,
  Plus,
  Sparkles,
  Calendar,
  Eye,
  CheckCircle2,
  AlertCircle,
  Clock,
  ArrowRight,
} from 'lucide-react';
import { offersApi } from '../../api/offers.api';
import { useAuth } from '../../context/AuthContext';
import { AppLayout } from '../../components/layout/AppLayout';

const TEMPLATES = [
  {
    type: 'FLAT_DISCOUNT',
    label: 'Flat % Discount',
    title: 'Flat Discount on All Items',
    discount_text: 'Flat 20% OFF',
    desc: 'Special discount on all items in-store. Walk in and claim this deal.',
  },
  {
    type: 'BOGO',
    label: 'Buy 1 Get 1 (BOGO)',
    title: 'Buy 1 Get 1 Free',
    discount_text: 'Buy 1 Get 1',
    desc: 'Buy one item, get another selected item absolutely free today.',
  },
  {
    type: 'BUY_X_GET_Y',
    label: 'Buy 2 Get 1 Free',
    title: 'Buy 2 Get 1 Free Special',
    discount_text: 'Buy 2 Get 1',
    desc: 'Buy any two selected items and get one free at our counter.',
  },
  {
    type: 'FESTIVE',
    label: 'Festive Deal',
    title: 'Festival Celebration In-Store Offer',
    discount_text: 'Festive Special',
    desc: 'Celebrate with special discounted prices in-store for a limited time.',
  },
];

export const OffersScreen = () => {
  const { shop } = useAuth();
  const [offers, setOffers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');

  const [formData, setFormData] = useState({
    title: '',
    discount_text: '',
    description: '',
    expires_in_days: 7,
    min_points_required: 0,
  });

  useEffect(() => {
    if (shop?.slug) {
      loadOffers();
    } else {
      setLoading(false);
    }
  }, [shop]);

  const loadOffers = async () => {
    try {
      setLoading(true);
      const data = await offersApi.getShopOffers(shop.slug);
      const list = Array.isArray(data) ? data : (Array.isArray(data?.offers) ? data.offers : []);
      setOffers(list);
    } catch (err) {
      console.error('Failed to load shop offers:', err);
      setOffers([]);
    } finally {
      setLoading(false);
    }
  };

  const handleApplyTemplate = (tpl) => {
    setFormData((prev) => ({
      ...prev,
      title: tpl.title,
      discount_text: tpl.discount_text,
      description: tpl.desc,
    }));
  };

  const handleCreateOffer = async (e) => {
    e.preventDefault();
    if (!formData.title.trim() || !formData.discount_text.trim()) {
      setFormError('Title aur Discount Text required hain');
      return;
    }

    try {
      setSubmitting(true);
      setFormError('');
      await offersApi.createOffer({
        title: formData.title.trim(),
        discount_text: formData.discount_text.trim(),
        description: formData.description.trim(),
        expires_in_days: Number(formData.expires_in_days) || 7,
        min_points_required: Number(formData.min_points_required) || 0,
      });

      setShowModal(false);
      setFormData({
        title: '',
        discount_text: '',
        description: '',
        expires_in_days: 7,
        min_points_required: 0,
      });
      await loadOffers();
    } catch (err) {
      setFormError(err.message || 'Offer create nahi ho saka');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AppLayout title="Offers & Promotions" subtitle="Publish Live Deals to Nearby Customers" showBack={true}>
      {/* Banner / Info Card */}
      <div
        style={{
          background: 'linear-gradient(135deg, rgba(37, 99, 235, 0.15), rgba(168, 85, 247, 0.15))',
          borderRadius: '16px',
          padding: '16px',
          marginBottom: '16px',
          border: '1px solid var(--border-subtle)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
          <Sparkles size={18} color="var(--color-primary)" />
          <span style={{ fontWeight: 800, fontSize: '0.95rem', color: 'var(--text-primary)' }}>
            Attract Nearby Walk-In Shoppers
          </span>
        </div>
        <p style={{ fontSize: '0.82rem', color: 'var(--text-secondary)', lineHeight: 1.45 }}>
          Publish a discount or festival deal. Nearby customers see live offers on your shop page and in the
          customer <strong>Deals Near You</strong> feed. Shops with live deals get up to 2x more walk-in visits!
        </p>

        <button
          onClick={() => setShowModal(true)}
          style={{
            marginTop: '12px',
            background: 'var(--color-primary)',
            color: '#fff',
            border: 'none',
            padding: '8px 16px',
            borderRadius: '10px',
            fontSize: '0.85rem',
            fontWeight: 700,
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
            cursor: 'pointer',
            boxShadow: '0 4px 12px rgba(37, 99, 235, 0.3)',
          }}
        >
          <Plus size={16} /> Create New Offer
        </button>
      </div>

      {/* Offers List */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '36px', color: 'var(--text-muted)' }}>
          Offers load ho rahe hain...
        </div>
      ) : offers.length === 0 ? (
        <div className="card" style={{ textAlign: 'center', padding: '36px 20px' }}>
          <Tag size={44} color="var(--text-muted)" style={{ margin: '0 auto 12px auto' }} />
          <div style={{ fontWeight: 700, fontSize: '1.05rem', color: 'var(--text-primary)' }}>
            No live offers right now
          </div>
          <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginTop: '6px' }}>
            Shops with a running deal show up in the customer Deals feed.
            <br />
            <strong>Idea:</strong> "Buy 1 Get 1 on slow movers" or "Flat 15% discount on all items this weekend".
          </p>
          <button
            onClick={() => setShowModal(true)}
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
            Create Your First Offer
          </button>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {offers.map((off) => (
            <div
              key={off.id}
              className="card"
              style={{
                margin: 0,
                padding: '16px',
                borderRadius: '14px',
                position: 'relative',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
                <span
                  style={{
                    background: 'linear-gradient(135deg, #10b981, #059669)',
                    color: '#fff',
                    padding: '4px 10px',
                    borderRadius: '8px',
                    fontSize: '0.78rem',
                    fontWeight: 700,
                  }}
                >
                  {off.discount_text}
                </span>

                <span
                  style={{
                    fontSize: '0.72rem',
                    color: off.is_active ? '#10b981' : 'var(--text-muted)',
                    background: off.is_active ? 'rgba(16, 185, 129, 0.12)' : 'rgba(255,255,255,0.06)',
                    padding: '3px 8px',
                    borderRadius: '6px',
                    fontWeight: 700,
                  }}
                >
                  {off.is_active ? 'Live in Customer Feed' : 'Inactive'}
                </span>
              </div>

              <div style={{ fontWeight: 800, fontSize: '1rem', color: 'var(--text-primary)', marginBottom: '4px' }}>
                {off.title}
              </div>
              {off.description && (
                <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', lineHeight: 1.4, marginBottom: '10px' }}>
                  {off.description}
                </p>
              )}

              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  paddingTop: '10px',
                  borderTop: '1px solid var(--border-subtle)',
                  fontSize: '0.75rem',
                  color: 'var(--text-muted)',
                }}
              >
                {off.expires_at ? (
                  <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                    <Clock size={12} /> Expires in: {new Date(off.expires_at).toLocaleDateString()}
                  </span>
                ) : (
                  <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                    <CheckCircle2 size={12} color="#10b981" /> Ongoing deal
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create Offer Modal */}
      {showModal && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            backgroundColor: 'rgba(0, 0, 0, 0.7)',
            backdropFilter: 'blur(5px)',
            zIndex: 1000,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '16px',
          }}
          onClick={() => setShowModal(false)}
        >
          <div
            className="card"
            style={{
              width: '100%',
              maxWidth: '480px',
              maxHeight: '90vh',
              overflowY: 'auto',
              padding: '20px',
              borderRadius: '20px',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <h2 style={{ fontSize: '1.2rem', fontWeight: 800, marginBottom: '6px' }}>
              Create Promotional Offer
            </h2>
            <p style={{ fontSize: '0.82rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
              Choose a high-converting template or write your custom deal.
            </p>

            {/* Quick Templates */}
            <div style={{ marginBottom: '16px' }}>
              <div style={{ fontSize: '0.75rem', fontWeight: 700, color: 'var(--text-muted)', marginBottom: '8px' }}>
                QUICK TEMPLATES
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '8px' }}>
                {TEMPLATES.map((tpl, i) => (
                  <button
                    key={i}
                    type="button"
                    onClick={() => handleApplyTemplate(tpl)}
                    style={{
                      background: 'var(--bg-app)',
                      border: '1px solid var(--border-subtle)',
                      borderRadius: '10px',
                      padding: '8px 10px',
                      textAlign: 'left',
                      cursor: 'pointer',
                      fontSize: '0.78rem',
                      fontWeight: 600,
                      color: 'var(--text-primary)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                    }}
                  >
                    <span>{tpl.label}</span>
                    <ArrowRight size={12} color="var(--color-primary)" />
                  </button>
                ))}
              </div>
            </div>

            {formError && (
              <div
                style={{
                  background: 'rgba(239, 68, 68, 0.1)',
                  color: '#ef4444',
                  padding: '8px 12px',
                  borderRadius: '8px',
                  fontSize: '0.82rem',
                  marginBottom: '12px',
                }}
              >
                {formError}
              </div>
            )}

            <form onSubmit={handleCreateOffer} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <div>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, display: 'block', marginBottom: '4px' }}>
                  Offer Title *
                </label>
                <input
                  type="text"
                  placeholder="e.g. Weekend Special Electronics Bonanza"
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                  required
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    borderRadius: '8px',
                    border: '1px solid var(--border-subtle)',
                    background: 'var(--bg-app)',
                    color: 'var(--text-primary)',
                  }}
                />
              </div>

              <div>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, display: 'block', marginBottom: '4px' }}>
                  Discount Badge Text *
                </label>
                <input
                  type="text"
                  placeholder="e.g. Flat 20% OFF or Buy 1 Get 1"
                  value={formData.discount_text}
                  onChange={(e) => setFormData({ ...formData, discount_text: e.target.value })}
                  required
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    borderRadius: '8px',
                    border: '1px solid var(--border-subtle)',
                    background: 'var(--bg-app)',
                    color: 'var(--text-primary)',
                  }}
                />
              </div>

              <div>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, display: 'block', marginBottom: '4px' }}>
                  Offer Description (Optional)
                </label>
                <textarea
                  placeholder="What items are included? Walk in and show this offer..."
                  rows={3}
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    borderRadius: '8px',
                    border: '1px solid var(--border-subtle)',
                    background: 'var(--bg-app)',
                    color: 'var(--text-primary)',
                    fontFamily: 'inherit',
                  }}
                />
              </div>

              <div>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, display: 'block', marginBottom: '4px' }}>
                  Valid For (Days)
                </label>
                <select
                  value={formData.expires_in_days}
                  onChange={(e) => setFormData({ ...formData, expires_in_days: Number(e.target.value) })}
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    borderRadius: '8px',
                    border: '1px solid var(--border-subtle)',
                    background: 'var(--bg-app)',
                    color: 'var(--text-primary)',
                  }}
                >
                  <option value={3}>3 Days</option>
                  <option value={7}>7 Days (1 Week)</option>
                  <option value={15}>15 Days</option>
                  <option value={30}>30 Days (1 Month)</option>
                </select>
              </div>

              <div style={{ display: 'flex', gap: '10px', marginTop: '10px' }}>
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  style={{
                    flex: 1,
                    padding: '10px',
                    borderRadius: '8px',
                    border: '1px solid var(--border-subtle)',
                    background: 'transparent',
                    color: 'var(--text-secondary)',
                    fontWeight: 600,
                    cursor: 'pointer',
                  }}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  style={{
                    flex: 1.5,
                    padding: '10px',
                    borderRadius: '8px',
                    border: 'none',
                    background: 'var(--color-primary)',
                    color: '#fff',
                    fontWeight: 700,
                    cursor: 'pointer',
                  }}
                >
                  {submitting ? 'Publishing...' : 'Publish Offer Live'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </AppLayout>
  );
};
