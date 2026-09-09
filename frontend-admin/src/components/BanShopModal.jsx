import React, { useState } from 'react';
import { X, AlertOctagon, ShieldAlert, Check } from 'lucide-react';

export const BanShopModal = ({ shop, onClose, onConfirmBan }) => {
  const [reason, setReason] = useState('Inappropriate or sexual content violation');
  const [blacklistIP, setBlacklistIP] = useState(true);
  const [blacklistDevice, setBlacklistDevice] = useState(true);
  const [blacklistImages, setBlacklistImages] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  if (!shop) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!reason.trim()) {
      alert('A ban reason is required.');
      return;
    }

    setSubmitting(true);
    try {
      await onConfirmBan(shop.id, {
        reason: reason.trim(),
        blacklist_ip: blacklistIP,
        blacklist_device: blacklistDevice,
        blacklist_images: blacklistImages,
      });
      onClose();
    } catch (err) {
      alert(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-card" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header" style={{ borderColor: 'rgba(239, 68, 68, 0.3)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div className="brand-icon" style={{ background: '#ef4444', boxShadow: '0 0 15px rgba(239, 68, 68, 0.4)' }}>
              <AlertOctagon size={20} />
            </div>
            <div>
              <h3 style={{ fontSize: '1.15rem', color: '#f87171' }}>1-Click Ban Shop & Blacklist</h3>
              <p style={{ fontSize: '0.78rem', color: '#94a3b8' }}>Shop: {shop.name} ({shop.slug})</p>
            </div>
          </div>
          <button className="btn-logout" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <div style={{ 
              background: 'rgba(239, 68, 68, 0.1)', 
              border: '1px solid rgba(239, 68, 68, 0.25)', 
              borderRadius: '8px', 
              padding: '12px 14px', 
              marginBottom: '18px',
              display: 'flex',
              gap: '10px',
              color: '#fca5a5',
              fontSize: '0.85rem'
            }}>
              <ShieldAlert size={20} style={{ flexShrink: 0, marginTop: '2px' }} />
              <div>
                <strong>Warning:</strong> Banning will instantly hide this shop from public directories, deactivate the owner's account, and block future registration attempts from associated network/device identifiers.
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Reason for Ban *</label>
              <select 
                className="form-select" 
                value={reason} 
                onChange={(e) => setReason(e.target.value)}
                style={{ marginBottom: '8px' }}
              >
                <option value="Inappropriate or sexual content violation">Inappropriate or sexual content violation</option>
                <option value="Counterfeit or fraudulent goods">Counterfeit or fraudulent goods</option>
                <option value="Severe community harassment or spam">Severe community harassment or spam</option>
                <option value="Illegal substance or restricted goods">Illegal substance or restricted goods</option>
                <option value="Repeated terms of service violation">Repeated terms of service violation</option>
              </select>
              <textarea
                className="form-textarea"
                rows="2"
                placeholder="Additional audit notes or case references..."
                value={reason}
                onChange={(e) => setReason(e.target.value)}
              />
            </div>

            <div style={{ marginTop: '16px' }}>
              <span className="form-label">Ban Evasion Shield Options</span>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', marginTop: '8px' }}>
                <label style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.85rem', cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={blacklistIP}
                    onChange={(e) => setBlacklistIP(e.target.checked)}
                    style={{ width: '16px', height: '16px', accentColor: '#ef4444' }}
                  />
                  <span>Blacklist IP Address (<code>{shop.creation_ip || 'Client Network'}</code>)</span>
                </label>

                <label style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.85rem', cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={blacklistDevice}
                    onChange={(e) => setBlacklistDevice(e.target.checked)}
                    style={{ width: '16px', height: '16px', accentColor: '#ef4444' }}
                  />
                  <span>Blacklist Device Fingerprint</span>
                </label>

                <label style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.85rem', cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={blacklistImages}
                    onChange={(e) => setBlacklistImages(e.target.checked)}
                    style={{ width: '16px', height: '16px', accentColor: '#ef4444' }}
                  />
                  <span>Blacklist Logo/Banner Image Perceptual Hashes</span>
                </label>
              </div>
            </div>
          </div>

          <div className="modal-footer">
            <button type="button" className="btn btn-outline" onClick={onClose} disabled={submitting}>
              Cancel
            </button>
            <button type="submit" className="btn btn-danger" disabled={submitting}>
              <AlertOctagon size={16} /> {submitting ? 'Enforcing Ban...' : 'Confirm Permanent Ban'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
