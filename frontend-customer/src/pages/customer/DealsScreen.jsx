import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Tag, MapPin, Store, Navigation, Sparkles, Clock } from 'lucide-react';
import { dealsApi } from '../../api/deals.api';
import { AppLayout } from '../../components/layout/AppLayout';
import { useLocation } from '../../context/LocationContext';
import { calculateDistanceKm, formatDistance } from '../../utils/distance';
import { getImageUrl } from '../../utils/imageUrl';

const CATEGORIES = ['All', 'Electronics', 'Kirana & Grocery', 'Pharmacy', 'Fashion', 'Home & Kitchen'];

export const DealsScreen = () => {
  const navigate = useNavigate();
  const { coords } = useLocation();

  const [deals, setDeals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedCategory, setSelectedCategory] = useState('All');

  useEffect(() => {
    loadDeals();
  }, [selectedCategory]);

  const loadDeals = async () => {
    try {
      setLoading(true);
      const params = {};
      if (selectedCategory !== 'All') {
        params.category = selectedCategory;
      }
      const data = await dealsApi.listDeals(params);
      const list = Array.isArray(data) ? data : (Array.isArray(data?.offers) ? data.offers : []);
      setDeals(list);
    } catch (err) {
      console.error('Failed to load deals:', err);
      setDeals([]);
    } finally {
      setLoading(false);
    }
  };

  const getBadgeStyle = (text = '') => {
    const lower = text.toLowerCase();
    if (lower.includes('bogo') || lower.includes('buy 1 get 1') || lower.includes('buy x')) {
      return { background: 'linear-gradient(135deg, #ec4899, #f43f5e)', color: '#fff' };
    }
    if (lower.includes('%') || lower.includes('flat') || lower.includes('off')) {
      return { background: 'linear-gradient(135deg, #10b981, #059669)', color: '#fff' };
    }
    return { background: 'linear-gradient(135deg, #6366f1, #4f46e5)', color: '#fff' };
  };

  return (
    <AppLayout title="Deals Near You" subtitle="Aas-Paas Ke Live Offers">
      {/* Category Pills */}
      <div
        style={{
          display: 'flex',
          gap: '8px',
          overflowX: 'auto',
          paddingBottom: '8px',
          marginBottom: '14px',
          scrollbarWidth: 'none',
        }}
      >
        {CATEGORIES.map((cat) => (
          <button
            key={cat}
            onClick={() => setSelectedCategory(cat)}
            style={{
              padding: '6px 14px',
              borderRadius: '20px',
              border: 'none',
              fontSize: '0.8rem',
              fontWeight: 600,
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              background: selectedCategory === cat ? 'var(--color-primary)' : 'var(--bg-card)',
              color: selectedCategory === cat ? '#fff' : 'var(--text-secondary)',
              boxShadow: selectedCategory === cat ? '0 2px 8px rgba(37, 99, 235, 0.3)' : 'none',
              transition: 'all 0.2s',
            }}
          >
            {cat}
          </button>
        ))}
      </div>

      {/* Deals Feed */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px', color: 'var(--text-muted)' }}>
          Live offers dhoondhe ja rahe hain...
        </div>
      ) : deals.length === 0 ? (
        <div className="card" style={{ textAlign: 'center', padding: '36px 20px' }}>
          <Tag size={44} color="var(--text-muted)" style={{ margin: '0 auto 12px auto' }} />
          <div style={{ fontWeight: 700, fontSize: '1.05rem', color: 'var(--text-primary)' }}>
            No live deals nearby right now
          </div>
          <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginTop: '6px' }}>
            Aas-paas ki dukaanon ke offers yahan live dikhenge. Dubara check karein!
          </p>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          {deals.map((deal) => {
            const distKm = calculateDistanceKm(
              coords?.lat,
              coords?.lng,
              deal.shop_latitude,
              deal.shop_longitude
            );

            return (
              <div
                key={deal.id}
                className="card"
                style={{
                  margin: 0,
                  padding: '16px',
                  borderRadius: '16px',
                  position: 'relative',
                  overflow: 'hidden',
                  border: '1px solid var(--border-subtle)',
                }}
              >
                {/* Offer Header */}
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '10px' }}>
                  <span
                    style={{
                      ...getBadgeStyle(deal.discount_text),
                      fontSize: '0.78rem',
                      fontWeight: 700,
                      padding: '4px 10px',
                      borderRadius: '8px',
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '4px',
                    }}
                  >
                    <Sparkles size={12} /> {deal.discount_text || 'Special Deal'}
                  </span>

                  {distKm != null && (
                    <span
                      style={{
                        fontSize: '0.75rem',
                        color: 'var(--text-muted)',
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: '3px',
                      }}
                    >
                      <MapPin size={12} /> {formatDistance(distKm)}
                    </span>
                  )}
                </div>

                {/* Offer Title & Description */}
                <div style={{ fontWeight: 700, fontSize: '1.05rem', color: 'var(--text-primary)', marginBottom: '6px' }}>
                  {deal.title}
                </div>
                {deal.description && (
                  <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginBottom: '12px', lineHeight: 1.4 }}>
                    {deal.description}
                  </p>
                )}

                {/* Participating Shop Info */}
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    paddingTop: '12px',
                    borderTop: '1px solid var(--border-subtle)',
                  }}
                >
                  <div
                    onClick={() => navigate(`/shop/${deal.shop_slug}`)}
                    style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}
                  >
                    <div
                      style={{
                        width: '34px',
                        height: '34px',
                        borderRadius: '8px',
                        backgroundColor: 'var(--color-primary-light)',
                        overflow: 'hidden',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {deal.shop_logo_url ? (
                        <img
                          src={getImageUrl(deal.shop_logo_url)}
                          alt={deal.shop_name}
                          style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                        />
                      ) : (
                        <Store size={18} color="var(--color-primary)" />
                      )}
                    </div>
                    <div>
                      <div style={{ fontWeight: 600, fontSize: '0.85rem' }}>{deal.shop_name}</div>
                      <div style={{ fontSize: '0.72rem', color: 'var(--text-muted)' }}>
                        {deal.shop_category || 'Local Shop'}
                      </div>
                    </div>
                  </div>

                  {/* Actions */}
                  <div style={{ display: 'flex', gap: '8px' }}>
                    {deal.shop_latitude && deal.shop_longitude && (
                      <a
                        href={`https://maps.google.com/?q=${deal.shop_latitude},${deal.shop_longitude}`}
                        target="_blank"
                        rel="noreferrer"
                        style={{
                          background: 'rgba(59, 130, 246, 0.1)',
                          color: '#3b82f6',
                          border: 'none',
                          padding: '6px 10px',
                          borderRadius: '8px',
                          fontSize: '0.75rem',
                          fontWeight: 600,
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '4px',
                          textDecoration: 'none',
                        }}
                      >
                        <Navigation size={12} /> Directions
                      </a>
                    )}
                    <button
                      onClick={() => navigate(`/shop/${deal.shop_slug}`)}
                      style={{
                        background: 'var(--color-primary)',
                        color: '#fff',
                        border: 'none',
                        padding: '6px 12px',
                        borderRadius: '8px',
                        fontSize: '0.75rem',
                        fontWeight: 600,
                        cursor: 'pointer',
                      }}
                    >
                      Dukan Dekhein
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </AppLayout>
  );
};
