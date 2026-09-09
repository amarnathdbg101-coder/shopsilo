import React, { useState, useEffect } from 'react';
import { 
  Ban, 
  Plus, 
  Trash2, 
  RefreshCw, 
  ShieldCheck, 
  Globe, 
  Smartphone, 
  Phone, 
  Image as ImageIcon 
} from 'lucide-react';
import { adminApi } from '../api/admin.api';

export const BlacklistVault = ({ onRefreshStats }) => {
  const [entities, setEntities] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);

  // Form state
  const [entityType, setEntityType] = useState('ip');
  const [entityValue, setEntityValue] = useState('');
  const [reason, setReason] = useState('Safety policy violation');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetchEntities();
  }, []);

  const fetchEntities = async () => {
    setLoading(true);
    try {
      const data = await adminApi.getBannedEntities({ limit: 100 });
      setEntities(data || []);
    } catch (err) {
      console.error('Failed to load banned entities:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = async (e) => {
    e.preventDefault();
    if (!entityValue.trim() || !reason.trim()) {
      alert('Value and reason are required.');
      return;
    }

    setSubmitting(true);
    try {
      await adminApi.addBannedEntity({
        entity_type: entityType,
        entity_value: entityValue.trim(),
        reason: reason.trim(),
      });
      setShowAddModal(false);
      setEntityValue('');
      await fetchEntities();
      if (onRefreshStats) onRefreshStats();
    } catch (err) {
      alert(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleUnban = async (id, value) => {
    if (!window.confirm(`Are you sure you want to unban and lift restrictions for: ${value}?`)) {
      return;
    }
    try {
      await adminApi.unbanEntity(id);
      await fetchEntities();
      if (onRefreshStats) onRefreshStats();
    } catch (err) {
      alert(err.message);
    }
  };

  const getEntityIcon = (type) => {
    switch (type) {
      case 'ip': return <Globe size={16} color="#60a5fa" />;
      case 'device_id': return <Smartphone size={16} color="#a78bfa" />;
      case 'phone': return <Phone size={16} color="#34d399" />;
      case 'image_hash': return <ImageIcon size={16} color="#fbbf24" />;
      default: return <Ban size={16} color="#f87171" />;
    }
  };

  return (
    <div>
      <div className="content-card">
        <div className="card-header-bar">
          <div>
            <h3 style={{ fontSize: '1.1rem' }}>Ban Evasion Blacklist Vault</h3>
            <p style={{ fontSize: '0.78rem', color: '#94a3b8' }}>
              Identifiers that are permanently blocked by BanGuard and cannot register shops or upload images.
            </p>
          </div>

          <button 
            className="btn btn-primary"
            onClick={() => setShowAddModal(true)}
          >
            <Plus size={16} /> Add Blacklist Entry
          </button>
        </div>

        {loading ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#94a3b8' }}>
            <RefreshCw size={24} className="spin" style={{ margin: '0 auto 12px' }} />
            Loading blacklist...
          </div>
        ) : entities.length === 0 ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#64748b' }}>
            <ShieldCheck size={36} style={{ margin: '0 auto 12px', opacity: 0.4, color: '#10b981' }} />
            <p>No active blacklists. The platform has zero banned identifiers.</p>
          </div>
        ) : (
          <div className="table-responsive">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Banned Value / Identifier</th>
                  <th>Reason</th>
                  <th>Added On</th>
                  <th style={{ textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {entities.map((ent) => (
                  <tr key={ent.id}>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontWeight: '600', textTransform: 'uppercase', fontSize: '0.75rem' }}>
                        {getEntityIcon(ent.entity_type)}
                        <span>{ent.entity_type.replace('_', ' ')}</span>
                      </div>
                    </td>
                    <td>
                      <code style={{ fontSize: '0.85rem', color: '#f87171', background: 'rgba(239, 68, 68, 0.1)', padding: '3px 8px', borderRadius: '6px' }}>
                        {ent.entity_value}
                      </code>
                    </td>
                    <td style={{ color: '#cbd5e1', fontSize: '0.85rem' }}>
                      {ent.reason}
                    </td>
                    <td style={{ fontSize: '0.8rem', color: '#94a3b8' }}>
                      {new Date(ent.created_at).toLocaleString()}
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      <button 
                        className="btn btn-outline btn-sm"
                        style={{ color: '#f87171', borderColor: 'rgba(239, 68, 68, 0.4)' }}
                        onClick={() => handleUnban(ent.id, ent.entity_value)}
                        title="Unban this entity"
                      >
                        <Trash2 size={14} /> Unban
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Add Banned Entity Modal */}
      {showAddModal && (
        <div className="modal-overlay" onClick={() => setShowAddModal(false)}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Add Identifier to Blacklist</h3>
              <button className="btn-logout" onClick={() => setShowAddModal(false)}>
                &times;
              </button>
            </div>

            <form onSubmit={handleAdd}>
              <div className="modal-body">
                <div className="form-group">
                  <label className="form-label">Identifier Type</label>
                  <select 
                    className="form-select"
                    value={entityType}
                    onChange={(e) => setEntityType(e.target.value)}
                  >
                    <option value="ip">IP Address (Network block)</option>
                    <option value="device_id">Device Fingerprint</option>
                    <option value="phone">Phone Number</option>
                    <option value="image_hash">Image Perceptual Hash (dHash)</option>
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Identifier Value *</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder={entityType === 'ip' ? 'e.g. 103.21.244.0' : entityType === 'phone' ? '+919876543210' : 'Value string...'}
                    value={entityValue}
                    onChange={(e) => setEntityValue(e.target.value)}
                    required
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Ban Reason *</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="e.g. Upload of illicit sexual material, repeat scammer..."
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                    required
                  />
                </div>
              </div>

              <div className="modal-footer">
                <button type="button" className="btn btn-outline" onClick={() => setShowAddModal(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn btn-danger" disabled={submitting}>
                  <Ban size={16} /> {submitting ? 'Adding...' : 'Enforce Blacklist'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
