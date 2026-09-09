import React from 'react';
import { 
  Store, 
  Clock, 
  AlertTriangle, 
  ShieldCheck, 
  Ban, 
  Users, 
  FolderTree,
  Package,
  ArrowUpRight, 
  CheckCircle2, 
  AlertOctagon 
} from 'lucide-react';

export const DashboardOverview = ({ stats, onNavigateTab }) => {
  return (
    <div>
      {/* Action Alerts */}
      {((stats?.pending_shops || 0) > 0 || (stats?.flagged_shops || 0) > 0 || (stats?.pending_reports || 0) > 0) && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginBottom: '28px' }}>
          {(stats?.pending_shops || 0) > 0 && (
            <div style={{
              background: 'rgba(245, 158, 11, 0.1)',
              border: '1px solid rgba(245, 158, 11, 0.3)',
              borderRadius: '12px',
              padding: '14px 20px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              color: '#fbbf24',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <Clock size={20} />
                <span>
                  <strong>{stats.pending_shops} New Shop(s)</strong> are awaiting merchant verification and approval.
                </span>
              </div>
              <button 
                className="btn btn-sm"
                style={{ background: '#f59e0b', color: '#000', fontWeight: '700' }}
                onClick={() => onNavigateTab('shops', 'pending_review')}
              >
                Review Approvals
              </button>
            </div>
          )}

          {(stats?.pending_reports || 0) > 0 && (
            <div style={{
              background: 'rgba(239, 68, 68, 0.1)',
              border: '1px solid rgba(239, 68, 68, 0.3)',
              borderRadius: '12px',
              padding: '14px 20px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              color: '#f87171',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <AlertTriangle size={20} />
                <span>
                  <strong>{stats.pending_reports} Open Grievance Report(s)</strong> require action under IT Rules 2021.
                </span>
              </div>
              <button 
                className="btn btn-danger btn-sm"
                onClick={() => onNavigateTab('reports', 'pending')}
              >
                Inspect Reports
              </button>
            </div>
          )}
        </div>
      )}

      {/* KPI Cards */}
      <div className="metrics-grid">
        <div className="metric-card primary" onClick={() => onNavigateTab('shops', 'all')} style={{ cursor: 'pointer' }}>
          <div className="metric-top">
            <span className="metric-title">Total Shops</span>
            <div className="metric-icon-wrap">
              <Store size={20} />
            </div>
          </div>
          <div className="metric-value">{stats?.total_shops || 0}</div>
          <div className="metric-subtitle">{stats?.active_shops || 0} Active & Published</div>
        </div>

        <div className="metric-card warning" onClick={() => onNavigateTab('shops', 'pending_review')} style={{ cursor: 'pointer' }}>
          <div className="metric-top">
            <span className="metric-title">Pending Approvals</span>
            <div className="metric-icon-wrap">
              <Clock size={20} />
            </div>
          </div>
          <div className="metric-value">{stats?.pending_shops || 0}</div>
          <div className="metric-subtitle">Awaiting Administrator Review</div>
        </div>

        <div className="metric-card danger" onClick={() => onNavigateTab('shops', 'flagged')} style={{ cursor: 'pointer' }}>
          <div className="metric-top">
            <span className="metric-title">High-Risk Flagged</span>
            <div className="metric-icon-wrap">
              <AlertTriangle size={20} />
            </div>
          </div>
          <div className="metric-value">{stats?.flagged_shops || 0}</div>
          <div className="metric-subtitle">Auto-Quarantined from Public Search</div>
        </div>

        <div className="metric-card danger" onClick={() => onNavigateTab('reports', 'pending')} style={{ cursor: 'pointer' }}>
          <div className="metric-top">
            <span className="metric-title">IT Grievance Reports</span>
            <div className="metric-icon-wrap">
              <ShieldCheck size={20} />
            </div>
          </div>
          <div className="metric-value">{stats?.pending_reports || 0}</div>
          <div className="metric-subtitle">{stats?.total_reports || 0} Total Grievances Handled</div>
        </div>

        <div className="metric-card" onClick={() => onNavigateTab('blacklist')} style={{ cursor: 'pointer' }}>
          <div className="metric-top">
            <span className="metric-title">Banned Identifiers</span>
            <div className="metric-icon-wrap" style={{ background: 'rgba(148, 163, 184, 0.15)', color: '#94a3b8' }}>
              <Ban size={20} />
            </div>
          </div>
          <div className="metric-value">{stats?.banned_entities || 0}</div>
          <div className="metric-subtitle">Banned IPs, Devices, Hashes</div>
        </div>

        <div className="metric-card success" onClick={() => onNavigateTab('users')} style={{ cursor: 'pointer' }}>
          <div className="metric-top">
            <span className="metric-title">Registered Users</span>
            <div className="metric-icon-wrap">
              <Users size={20} />
            </div>
          </div>
          <div className="metric-value">{stats?.total_users || 0}</div>
          <div className="metric-subtitle">Customers & Store Owners</div>
        </div>

        <div className="metric-card primary" onClick={() => onNavigateTab('categories')} style={{ cursor: 'pointer' }}>
          <div className="metric-top">
            <span className="metric-title">Catalog Categories</span>
            <div className="metric-icon-wrap" style={{ background: 'rgba(59, 130, 246, 0.15)', color: '#60a5fa' }}>
              <FolderTree size={20} />
            </div>
          </div>
          <div className="metric-value">{stats?.total_categories || 0}</div>
          <div className="metric-subtitle">Global Product Categories</div>
        </div>

        <div className="metric-card">
          <div className="metric-top">
            <span className="metric-title">Live Products</span>
            <div className="metric-icon-wrap" style={{ background: 'rgba(16, 185, 129, 0.15)', color: '#10b981' }}>
              <Package size={20} />
            </div>
          </div>
          <div className="metric-value">{stats?.total_products || 0}</div>
          <div className="metric-subtitle">Merchant Catalog Inventory</div>
        </div>
      </div>

      {/* Safety & Compliance Card */}
      <div className="content-card" style={{ padding: '24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div className="brand-icon" style={{ width: '32px', height: '32px' }}>
              <ShieldCheck size={18} />
            </div>
            <h3 style={{ fontSize: '1.1rem' }}>Active Safety Protocols</h3>
          </div>
          <span className="status-pill active">All Protections Live</span>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '16px' }}>
          <div style={{ background: '#090d16', padding: '16px', borderRadius: '12px', border: '1px solid #1e293b' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: '#10b981', fontWeight: '700', fontSize: '0.85rem' }}>
              <CheckCircle2 size={16} /> Perceptual Image Hashing (dHash)
            </div>
            <p style={{ color: '#94a3b8', fontSize: '0.78rem', marginTop: '6px' }}>
              Zero-dependency visual fingerprinting active. Duplicate or cropped adult/banned images are blocked on upload.
            </p>
          </div>

          <div style={{ background: '#090d16', padding: '16px', borderRadius: '12px', border: '1px solid #1e293b' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: '#10b981', fontWeight: '700', fontSize: '0.85rem' }}>
              <CheckCircle2 size={16} /> Ban Evasion Guard
            </div>
            <p style={{ color: '#94a3b8', fontSize: '0.78rem', marginTop: '6px' }}>
              Client network IP and device fingerprints verified against blacklist to prevent repeat abusers creating accounts.
            </p>
          </div>

          <div style={{ background: '#090d16', padding: '16px', borderRadius: '12px', border: '1px solid #1e293b' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: '#10b981', fontWeight: '700', fontSize: '0.85rem' }}>
              <CheckCircle2 size={16} /> IT Rules 2021 Safe Harbor
            </div>
            <p style={{ color: '#94a3b8', fontSize: '0.78rem', marginTop: '6px' }}>
              Auto-quarantine triggered upon 3 severe reports, shielding founders from platform liability.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};
