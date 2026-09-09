import React, { useState, useEffect } from 'react';
import { 
  Users, 
  Search, 
  Shield, 
  Store, 
  UserCheck, 
  UserX, 
  RefreshCw, 
  Mail, 
  Phone, 
  Edit3, 
  CheckCircle2, 
  XCircle,
  X
} from 'lucide-react';
import { adminApi } from '../api/admin.api';

export const UserManagement = ({ onRefreshStats }) => {
  const [users, setUsers] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [roleFilter, setRoleFilter] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');
  
  // Role change modal
  const [editingUser, setEditingUser] = useState(null);
  const [targetRole, setTargetRole] = useState('customer');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetchUsers();
  }, [roleFilter]);

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const data = await adminApi.getUsers({
        role: roleFilter === 'all' ? '' : roleFilter,
        q: searchQuery.trim(),
        limit: 50,
      });
      setUsers(data.users || []);
      setTotal(data.total || 0);
    } catch (err) {
      console.error('Failed to load users:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e) => {
    e.preventDefault();
    fetchUsers();
  };

  const handleToggleStatus = async (user) => {
    const action = user.is_active ? 'suspend' : 'activate';
    if (!window.confirm(`Are you sure you want to ${action} account for ${user.full_name || user.email}?`)) {
      return;
    }

    try {
      await adminApi.updateUserStatus(user.id, {
        is_active: !user.is_active,
      });
      await fetchUsers();
      if (onRefreshStats) onRefreshStats();
    } catch (err) {
      alert(err.message);
    }
  };

  const handleSaveRole = async (e) => {
    e.preventDefault();
    if (!editingUser) return;
    setSubmitting(true);
    try {
      await adminApi.updateUserStatus(editingUser.id, {
        role: targetRole,
      });
      setEditingUser(null);
      await fetchUsers();
      if (onRefreshStats) onRefreshStats();
    } catch (err) {
      alert(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const getRoleBadge = (role) => {
    switch (role) {
      case 'admin':
        return <span className="status-pill" style={{ background: 'rgba(245, 158, 11, 0.15)', color: '#fbbf24', border: '1px solid rgba(245, 158, 11, 0.3)' }}>Admin</span>;
      case 'merchant':
        return <span className="status-pill" style={{ background: 'rgba(168, 85, 247, 0.15)', color: '#c084fc', border: '1px solid rgba(168, 85, 247, 0.3)' }}>Merchant</span>;
      default:
        return <span className="status-pill" style={{ background: 'rgba(59, 130, 246, 0.15)', color: '#60a5fa', border: '1px solid rgba(59, 130, 246, 0.3)' }}>Customer</span>;
    }
  };

  return (
    <div>
      <div className="content-card">
        <div className="card-header-bar">
          <div className="filter-tabs">
            <button
              className={`filter-btn ${roleFilter === 'all' ? 'active' : ''}`}
              onClick={() => setRoleFilter('all')}
            >
              All Users ({total})
            </button>
            <button
              className={`filter-btn ${roleFilter === 'merchant' ? 'active' : ''}`}
              onClick={() => setRoleFilter('merchant')}
            >
              Merchants
            </button>
            <button
              className={`filter-btn ${roleFilter === 'customer' ? 'active' : ''}`}
              onClick={() => setRoleFilter('customer')}
            >
              Customers
            </button>
            <button
              className={`filter-btn ${roleFilter === 'admin' ? 'active' : ''}`}
              onClick={() => setRoleFilter('admin')}
            >
              Admins
            </button>
          </div>

          <form onSubmit={handleSearch} className="search-input-wrap">
            <Search size={16} className="search-icon" />
            <input
              type="text"
              placeholder="Search by name, email, phone..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </form>
        </div>

        {loading ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#94a3b8' }}>
            <RefreshCw size={24} className="spin" style={{ margin: '0 auto 12px' }} />
            Loading registered users...
          </div>
        ) : users.length === 0 ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#64748b' }}>
            <Users size={36} style={{ margin: '0 auto 12px', opacity: 0.4 }} />
            <p>No users found matching query.</p>
          </div>
        ) : (
          <div className="table-responsive">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>User Details</th>
                  <th>Contact</th>
                  <th>Role</th>
                  <th>Status</th>
                  <th>Joined Date</th>
                  <th style={{ textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id}>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <div style={{
                          width: '38px',
                          height: '38px',
                          borderRadius: '50%',
                          background: u.role === 'admin' ? '#f59e0b' : u.role === 'merchant' ? '#9333ea' : '#2563eb',
                          color: '#fff',
                          fontWeight: '700',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: '0.9rem',
                          flexShrink: 0
                        }}>
                          {u.full_name ? u.full_name[0].toUpperCase() : u.email[0].toUpperCase()}
                        </div>
                        <div>
                          <div style={{ fontWeight: '700', color: '#fff', fontSize: '0.95rem' }}>
                            {u.full_name || 'No name'}
                          </div>
                          <div style={{ fontSize: '0.75rem', color: '#64748b' }}>ID: {u.id.slice(0, 8)}...</div>
                        </div>
                      </div>
                    </td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.85rem', color: '#cbd5e1' }}>
                        <Mail size={13} color="#94a3b8" /> {u.email}
                      </div>
                      {u.phone && (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.78rem', color: '#94a3b8', marginTop: '3px' }}>
                          <Phone size={12} color="#64748b" /> {u.phone}
                        </div>
                      )}
                    </td>
                    <td>{getRoleBadge(u.role)}</td>
                    <td>
                      <span className={`status-pill ${u.is_active ? 'active' : 'suspended'}`}>
                        {u.is_active ? 'Active' : 'Suspended'}
                      </span>
                    </td>
                    <td style={{ fontSize: '0.8rem', color: '#94a3b8' }}>
                      {new Date(u.created_at).toLocaleDateString()}
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                        <button
                          className="btn btn-outline btn-sm"
                          onClick={() => {
                            setEditingUser(u);
                            setTargetRole(u.role);
                          }}
                          title="Change Role"
                        >
                          <Edit3 size={14} /> Role
                        </button>

                        <button
                          className={`btn btn-sm ${u.is_active ? 'btn-danger' : 'btn-success'}`}
                          onClick={() => handleToggleStatus(u)}
                          title={u.is_active ? 'Suspend User' : 'Reactivate User'}
                        >
                          {u.is_active ? (
                            <>
                              <UserX size={14} /> Suspend
                            </>
                          ) : (
                            <>
                              <UserCheck size={14} /> Activate
                            </>
                          )}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Role Change Modal */}
      {editingUser && (
        <div className="modal-overlay" onClick={() => setEditingUser(null)}>
          <div className="modal-card" style={{ maxWidth: '460px' }} onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <div className="brand-icon" style={{ width: '32px', height: '32px' }}>
                  <Shield size={18} />
                </div>
                <div>
                  <h3 style={{ fontSize: '1.1rem' }}>Change User Role</h3>
                  <p style={{ fontSize: '0.78rem', color: '#94a3b8' }}>{editingUser.full_name || editingUser.email}</p>
                </div>
              </div>
              <button className="btn-logout" onClick={() => setEditingUser(null)}>
                <X size={20} />
              </button>
            </div>

            <form onSubmit={handleSaveRole}>
              <div className="modal-body">
                <div className="form-group">
                  <label className="form-label">Select System Role</label>
                  <select
                    className="form-select"
                    value={targetRole}
                    onChange={(e) => setTargetRole(e.target.value)}
                  >
                    <option value="customer">Customer (Shopper & Reservations)</option>
                    <option value="merchant">Merchant (Store Owner & POS Cashier)</option>
                    <option value="admin">Administrator (Super User Control)</option>
                  </select>
                </div>

                <div style={{
                  background: '#090d16',
                  padding: '12px 16px',
                  borderRadius: '10px',
                  border: '1px solid #1e293b',
                  fontSize: '0.78rem',
                  color: '#94a3b8'
                }}>
                  {targetRole === 'admin' && '⚠️ Administrator role grants full access to platform analytics, user moderation, shop banning, and grievance management.'}
                  {targetRole === 'merchant' && 'Store owners can create and manage their retail shop, inventory, POS billing, and digital receipts.'}
                  {targetRole === 'customer' && 'Customers can browse shops, reserve items, view receipt passbooks, and submit reviews.'}
                </div>
              </div>

              <div className="modal-footer">
                <button
                  type="button"
                  className="btn btn-outline"
                  onClick={() => setEditingUser(null)}
                  disabled={submitting}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={submitting}
                >
                  {submitting ? 'Updating...' : 'Save Role'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
