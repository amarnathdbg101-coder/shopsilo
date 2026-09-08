/**
 * Customer Explore & Marketplace Screen
 * 
 * Hinglish Hint:
 * Grahako ke liye aas-paas ki dukanon ko khojne ka page:
 * - Dukan search by name / area
 * - Shop open/closed indicator
 * - Dukan par tap karke uska online storefront kholna
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Store, Search, MapPin, ChevronRight } from 'lucide-react';
import { shopApi } from '../../api/shop.api';
import { AppLayout } from '../../components/layout/AppLayout';

export const ExploreShopsScreen = () => {
  const navigate = useNavigate();
  const [shops, setShops] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    loadShops();
  }, []);

  const loadShops = async () => {
    try {
      setLoading(true);
      const data = await shopApi.listPublicShops();
      const list = Array.isArray(data?.shops) ? data.shops : (Array.isArray(data) ? data : []);
      setShops(list);
    } catch (err) {
      console.error('Failed to load shops:', err);
      setShops([]);
    } finally {
      setLoading(false);
    }
  };

  const filteredShops = (Array.isArray(shops) ? shops : []).filter(
    (s) =>
      (s.name || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (s.category || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (s.city || '').toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <AppLayout title="Local Shops" subtitle="Aas-Paas Ki Dukanien">
      {/* Search Bar */}
      <div className="search-box">
        <Search size={18} />
        <input
          type="text"
          placeholder="Dukan ka naam, category ya jagah khojein..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
      </div>

      {/* Shops List */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {filteredShops.length === 0 ? (
          <div className="card" style={{ textAlign: 'center', padding: '32px' }}>
            <Store size={40} color="var(--text-muted)" style={{ margin: '0 auto 8px auto' }} />
            <div style={{ fontWeight: 700 }}>Koi dukan nahi mili</div>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
              Dusre shahar ya naam se khoj kar dekhein.
            </p>
          </div>
        ) : (
          filteredShops.map((s) => (
            <div
              key={s.id}
              className="card card-clickable"
              onClick={() => navigate(`/shop/${s.slug}`)}
              style={{ margin: 0, padding: '14px' }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                  <div
                    style={{
                      width: '48px',
                      height: '48px',
                      borderRadius: 'var(--radius-md)',
                      backgroundColor: 'var(--color-primary-light)',
                      color: 'var(--color-primary)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexShrink: 0,
                      overflow: 'hidden',
                      border: '1px solid var(--border-subtle)',
                    }}
                  >
                    {s.logo_url ? (
                      <img
                        src={s.logo_url}
                        alt={s.name}
                        style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                      />
                    ) : (
                      <Store size={24} />
                    )}
                  </div>
                  <div>
                    <div style={{ fontWeight: 700, fontSize: '0.95rem' }}>{s.name}</div>
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)', marginTop: '2px' }}>
                      {s.category || 'General Store'}
                    </div>
                    <div
                      style={{
                        fontSize: '0.72rem',
                        color: 'var(--text-muted)',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '4px',
                        marginTop: '4px',
                      }}
                    >
                      <MapPin size={12} /> {s.address || s.city || 'Local Bazar'}
                    </div>
                  </div>
                </div>

                <div style={{ textAlign: 'right' }}>
                  <span className={`badge ${s.is_active ? 'badge-success' : 'badge-danger'}`}>
                    {s.is_active ? 'Khuli Hai' : 'Band Hai'}
                  </span>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </AppLayout>
  );
};
