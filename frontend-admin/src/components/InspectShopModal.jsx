import React, { useState } from 'react';
import { 
  X, 
  Store, 
  MapPin, 
  Phone, 
  Globe, 
  ShieldAlert, 
  CheckCircle2, 
  AlertOctagon, 
  Cpu, 
  Clock 
} from 'lucide-react';
import { API_BASE_URL } from '../api/client';

export const InspectShopModal = ({ shop, onClose, onUpdateStatus, onOpenBanModal }) => {
  const [updating, setUpdating] = useState(false);

  if (!shop) return null;

  const handleStatusChange = async (newStatus, reason = '') => {
    setUpdating(true);
    try {
      await onUpdateStatus(shop.id, newStatus, reason);
      onClose();
    } catch (err) {
      alert(err.message);
    } finally {
      setUpdating(false);
    }
  };

  const getFullImageUrl = (url) => {
    if (!url) return '';
    if (url.startsWith('http')) return url;
    return `${API_BASE_URL}${url.startsWith('/') ? '' : '/'}${url}`;
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-card" style={{ maxWidth: '680px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div className="brand-icon" style={{ width: '32px', height: '32px' }}>
              <Store size={18} />
            </div>
            <div>
              <h3 style={{ fontSize: '1.15rem' }}>{shop.name}</h3>
              <p style={{ fontSize: '0.78rem', color: '#94a3b8' }}>Slug: /{shop.slug} • ID: {shop.id}</p>
            </div>
          </div>
          <button className="btn-logout" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <div className="modal-body">
          {/* Status & Flag Notice */}
          <div style={{ 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center', 
            background: '#090d16', 
            padding: '12px 16px', 
            borderRadius: '10px',
            marginBottom: '20px',
            border: '1px solid #1e293b'
          }}>
            <div>
              <span style={{ fontSize: '0.78rem', color: '#64748b', display: 'block' }}>Current Moderation Status</span>
              <span className={`status-pill ${shop.status}`} style={{ marginTop: '4px' }}>
                {shop.status.replace('_', ' ')}
              </span>
            </div>
            <div style={{ textAlign: 'right' }}>
              <span style={{ fontSize: '0.78rem', color: '#64748b', display: 'block' }}>Flagged Count</span>
              <span style={{ 
                fontSize: '1rem', 
                fontWeight: '800', 
                color: shop.flagged_count > 0 ? '#ef4444' : '#10b981' 
              }}>
                {shop.flagged_count} Reports
              </span>
            </div>
          </div>

          {shop.suspension_reason && (
            <div style={{ 
              background: 'rgba(239, 68, 68, 0.1)', 
              border: '1px solid rgba(239, 68, 68, 0.3)', 
              borderRadius: '8px', 
              padding: '10px 14px', 
              marginBottom: '20px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              color: '#f87171',
              fontSize: '0.85rem'
            }}>
              <ShieldAlert size={18} />
              <span><strong>Reason:</strong> {shop.suspension_reason}</span>
            </div>
          )}

          {/* Images Section */}
          <div style={{ marginBottom: '20px' }}>
            <span className="form-label">Shop Logo & Banners</span>
            <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', marginTop: '6px' }}>
              {shop.logo_url ? (
                <div style={{ width: '80px', height: '80px', borderRadius: '10px', overflow: 'hidden', border: '1px solid #334155' }}>
                  <img 
                    src={getFullImageUrl(shop.logo_url)} 
                    alt="Logo" 
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                    onError={(e) => { e.target.style.display = 'none'; }}
                  />
                </div>
              ) : (
                <div style={{ width: '80px', height: '80px', borderRadius: '10px', background: '#090d16', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#64748b', fontSize: '0.75rem' }}>
                  No Logo
                </div>
              )}

              {shop.banners && shop.banners.length > 0 ? (
                shop.banners.map((b, idx) => (
                  <div key={idx} style={{ width: '140px', height: '80px', borderRadius: '10px', overflow: 'hidden', border: '1px solid #334155' }}>
                    <img 
                      src={getFullImageUrl(b)} 
                      alt={`Banner ${idx}`} 
                      style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                      onError={(e) => { e.target.style.display = 'none'; }}
                    />
                  </div>
                ))
              ) : (
                <div style={{ height: '80px', padding: '0 16px', borderRadius: '10px', background: '#090d16', display: 'flex', alignItems: 'center', color: '#64748b', fontSize: '0.75rem' }}>
                  No Banners uploaded
                </div>
              )}
            </div>
          </div>

          {/* Details Grid */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '20px' }}>
            <div>
              <span className="form-label">Category</span>
              <p style={{ color: '#fff', fontSize: '0.9rem' }}>{shop.category || 'General'}</p>
            </div>
            <div>
              <span className="form-label">Phone / WhatsApp</span>
              <p style={{ color: '#fff', fontSize: '0.9rem' }}>{shop.phone || 'N/A'}</p>
            </div>
            <div>
              <span className="form-label">City & Pincode</span>
              <p style={{ color: '#fff', fontSize: '0.9rem' }}>{shop.city || 'N/A'} - {shop.pincode || 'N/A'}</p>
            </div>
            <div>
              <span className="form-label">Registered At</span>
              <p style={{ color: '#fff', fontSize: '0.9rem' }}>{new Date(shop.created_at).toLocaleString()}</p>
            </div>
          </div>

          {/* Audit Trail & Digital Footprint */}
          <div style={{ background: '#090d16', border: '1px solid #1e293b', borderRadius: '10px', padding: '14px' }}>
            <span style={{ fontSize: '0.75rem', fontWeight: '700', color: '#38bdf8', textTransform: 'uppercase', letterSpacing: '0.05em', display: 'flex', alignItems: 'center', gap: '6px' }}>
              <Cpu size={14} /> Legal Audit & Device Footprint
            </span>
            <div style={{ marginTop: '10px', display: 'grid', gridTemplateColumns: '1fr', gap: '8px', fontSize: '0.8rem' }}>
              <div>
                <span style={{ color: '#64748b' }}>Creation IP: </span>
                <code style={{ color: '#a5b4fc', background: '#1e293b', padding: '2px 6px', borderRadius: '4px' }}>
                  {shop.creation_ip || '127.0.0.1 (Localhost / Direct)'}
                </code>
              </div>
              <div>
                <span style={{ color: '#64748b' }}>Device ID: </span>
                <code style={{ color: '#a5b4fc', background: '#1e293b', padding: '2px 6px', borderRadius: '4px' }}>
                  {shop.device_fingerprint || 'Not provided'}
                </code>
              </div>
              <div style={{ wordBreak: 'break-all' }}>
                <span style={{ color: '#64748b' }}>User Agent: </span>
                <span style={{ color: '#94a3b8' }}>{shop.creation_user_agent || 'Standard Web Browser'}</span>
              </div>
            </div>
          </div>
        </div>

        <div className="modal-footer" style={{ justifyContent: 'space-between' }}>
          <button 
            className="btn btn-danger"
            onClick={() => onOpenBanModal(shop)}
            disabled={updating || shop.status === 'banned'}
          >
            <AlertOctagon size={16} /> 1-Click Ban Shop
          </button>

          <div style={{ display: 'flex', gap: '10px' }}>
            {shop.status !== 'active' && (
              <button 
                className="btn btn-success"
                onClick={() => handleStatusChange('active', 'Approved by administrator')}
                disabled={updating}
              >
                <CheckCircle2 size={16} /> 1-Click Approve
              </button>
            )}

            {shop.status === 'active' && (
              <button 
                className="btn btn-outline"
                onClick={() => handleStatusChange('suspended', 'Suspended by admin review')}
                disabled={updating}
              >
                Suspend Shop
              </button>
            )}

            <button className="btn btn-outline" onClick={onClose}>
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
