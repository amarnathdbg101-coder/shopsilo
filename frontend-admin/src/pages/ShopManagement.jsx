import React, { useState, useEffect } from 'react';
import { 
  Store, 
  Search, 
  CheckCircle2, 
  AlertOctagon, 
  Eye, 
  ShieldAlert, 
  RefreshCw,
  Clock 
} from 'lucide-react';
import { adminApi } from '../api/admin.api';
import { InspectShopModal } from '../components/InspectShopModal';
import { BanShopModal } from '../components/BanShopModal';

export const ShopManagement = ({ initialFilter = 'all', onRefreshStats }) => {
  const [shops, setShops] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState(initialFilter);
  const [searchQuery, setSearchQuery] = useState('');
  
  const [selectedShop, setSelectedShop] = useState(null);
  const [shopToBan, setShopToBan] = useState(null);

  useEffect(() => {
    setStatusFilter(initialFilter);
  }, [initialFilter]);

  useEffect(() => {
    fetchShops();
  }, [statusFilter]);

  const fetchShops = async () => {
    setLoading(true);
    try {
      const data = await adminApi.getShops({
        status: statusFilter,
        q: searchQuery.trim(),
        limit: 50,
      });
      setShops(data.shops || []);
      setTotal(data.total || 0);
    } catch (err) {
      console.error('Failed to load shops:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e) => {
    e.preventDefault();
    fetchShops();
  };

  const handleUpdateStatus = async (shopId, newStatus, reason = '') => {
    await adminApi.updateShopStatus(shopId, newStatus, reason);
    await fetchShops();
    if (onRefreshStats) onRefreshStats();
  };

  const handleConfirmBan = async (shopId, banData) => {
    await adminApi.banShop(shopId, banData);
    await fetchShops();
    if (onRefreshStats) onRefreshStats();
  };

  return (
    <div>
      <div className="content-card">
        <div className="card-header-bar">
          <div className="filter-tabs">
            <button 
              className={`filter-btn ${statusFilter === 'all' ? 'active' : ''}`}
              onClick={() => setStatusFilter('all')}
            >
              All Shops
            </button>
            <button 
              className={`filter-btn ${statusFilter === 'pending_review' ? 'active' : ''}`}
              onClick={() => setStatusFilter('pending_review')}
            >
              Pending Approval
            </button>
            <button 
              className={`filter-btn ${statusFilter === 'flagged' ? 'active' : ''}`}
              onClick={() => setStatusFilter('flagged')}
            >
              Flagged (High Risk)
            </button>
            <button 
              className={`filter-btn ${statusFilter === 'active' ? 'active' : ''}`}
              onClick={() => setStatusFilter('active')}
            >
              Active
            </button>
            <button 
              className={`filter-btn ${statusFilter === 'suspended' ? 'active' : ''}`}
              onClick={() => setStatusFilter('suspended')}
            >
              Suspended
            </button>
            <button 
              className={`filter-btn ${statusFilter === 'banned' ? 'active' : ''}`}
              onClick={() => setStatusFilter('banned')}
            >
              Banned
            </button>
          </div>

          <form onSubmit={handleSearch} className="search-input-wrap">
            <Search size={16} className="search-icon" />
            <input
              type="text"
              placeholder="Search by shop name, city, phone..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </form>
        </div>

        {loading ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#94a3b8' }}>
            <RefreshCw size={24} className="spin" style={{ margin: '0 auto 12px' }} />
            Loading shops directory...
          </div>
        ) : shops.length === 0 ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#64748b' }}>
            <Store size={36} style={{ margin: '0 auto 12px', opacity: 0.4 }} />
            <p>No shops found matching "{statusFilter}".</p>
          </div>
        ) : (
          <div className="table-responsive">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>Shop Details</th>
                  <th>Category / City</th>
                  <th>Status</th>
                  <th>Reports</th>
                  <th>Audit IP</th>
                  <th>Registered</th>
                  <th style={{ textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {shops.map((s) => (
                  <tr key={s.id}>
                    <td>
                      <div style={{ fontWeight: '700', color: '#fff', fontSize: '0.95rem' }}>{s.name}</div>
                      <div style={{ fontSize: '0.75rem', color: '#94a3b8' }}>/{s.slug} • {s.phone || 'No phone'}</div>
                    </td>
                    <td>
                      <div>{s.category || 'General'}</div>
                      <div style={{ fontSize: '0.75rem', color: '#64748b' }}>{s.city || 'N/A'} ({s.pincode || 'N/A'})</div>
                    </td>
                    <td>
                      <span className={`status-pill ${s.status}`}>
                        {s.status.replace('_', ' ')}
                      </span>
                    </td>
                    <td>
                      {s.flagged_count > 0 ? (
                        <span style={{ color: '#ef4444', fontWeight: '700', display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <ShieldAlert size={14} /> {s.flagged_count}
                        </span>
                      ) : (
                        <span style={{ color: '#64748b' }}>0</span>
                      )}
                    </td>
                    <td>
                      <code style={{ fontSize: '0.75rem', color: '#a5b4fc', background: '#1e293b', padding: '2px 6px', borderRadius: '4px' }}>
                        {s.creation_ip || 'Localhost'}
                      </code>
                    </td>
                    <td style={{ fontSize: '0.8rem', color: '#94a3b8' }}>
                      {new Date(s.created_at).toLocaleDateString()}
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                        <button 
                          className="btn btn-outline btn-sm"
                          onClick={() => setSelectedShop(s)}
                          title="Inspect Shop Details"
                        >
                          <Eye size={14} /> Inspect
                        </button>

                        {s.status !== 'active' && s.status !== 'banned' && (
                          <button 
                            className="btn btn-success btn-sm"
                            onClick={() => handleUpdateStatus(s.id, 'active', 'Approved by administrator')}
                            title="1-Click Approve"
                          >
                            <CheckCircle2 size={14} /> Approve
                          </button>
                        )}

                        {s.status !== 'banned' && (
                          <button 
                            className="btn btn-danger btn-sm"
                            onClick={() => setShopToBan(s)}
                            title="1-Click Ban Shop"
                          >
                            <AlertOctagon size={14} /> Ban
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Inspect Modal */}
      {selectedShop && (
        <InspectShopModal
          shop={selectedShop}
          onClose={() => setSelectedShop(null)}
          onUpdateStatus={handleUpdateStatus}
          onOpenBanModal={(shop) => {
            setSelectedShop(null);
            setShopToBan(shop);
          }}
        />
      )}

      {/* Ban Modal */}
      {shopToBan && (
        <BanShopModal
          shop={shopToBan}
          onClose={() => setShopToBan(null)}
          onConfirmBan={handleConfirmBan}
        />
      )}
    </div>
  );
};
