import React, { useState, useEffect } from 'react';
import { 
  AlertTriangle, 
  CheckCircle2, 
  XCircle, 
  ShieldAlert, 
  RefreshCw, 
  MessageSquare,
  Clock 
} from 'lucide-react';
import { adminApi } from '../api/admin.api';

export const ReportsDesk = ({ initialStatus = 'pending', onRefreshStats }) => {
  const [reports, setReports] = useState([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState(initialStatus);
  const [actionModalReport, setActionModalReport] = useState(null);
  const [actionTakenText, setActionTakenText] = useState('Reviewed and addressed under IT Rules safety guidelines');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetchReports();
  }, [statusFilter]);

  const fetchReports = async () => {
    setLoading(true);
    try {
      const data = await adminApi.getReports({
        status: statusFilter === 'all' ? '' : statusFilter,
        limit: 50,
      });
      setReports(data || []);
    } catch (err) {
      console.error('Failed to load reports:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleResolve = async (status) => {
    if (!actionModalReport) return;
    setSubmitting(true);
    try {
      await adminApi.resolveReport(actionModalReport.id, status, actionTakenText);
      setActionModalReport(null);
      await fetchReports();
      if (onRefreshStats) onRefreshStats();
    } catch (err) {
      alert(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div>
      <div className="content-card">
        <div className="card-header-bar">
          <div className="filter-tabs">
            <button 
              className={`filter-btn ${statusFilter === 'pending' ? 'active' : ''}`}
              onClick={() => setStatusFilter('pending')}
            >
              Pending Review
            </button>
            <button 
              className={`filter-btn ${statusFilter === 'action_taken' ? 'active' : ''}`}
              onClick={() => setStatusFilter('action_taken')}
            >
              Action Taken
            </button>
            <button 
              className={`filter-btn ${statusFilter === 'dismissed' ? 'active' : ''}`}
              onClick={() => setStatusFilter('dismissed')}
            >
              Dismissed
            </button>
            <button 
              className={`filter-btn ${statusFilter === 'all' ? 'active' : ''}`}
              onClick={() => setStatusFilter('all')}
            >
              All Reports
            </button>
          </div>

          <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>
            IT Rules 2021: 24-Hour Grievance Redressal SLA
          </div>
        </div>

        {loading ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#94a3b8' }}>
            <RefreshCw size={24} className="spin" style={{ margin: '0 auto 12px' }} />
            Loading grievance reports...
          </div>
        ) : reports.length === 0 ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#64748b' }}>
            <CheckCircle2 size={36} style={{ margin: '0 auto 12px', opacity: 0.4, color: '#10b981' }} />
            <p>No reports found in this category.</p>
          </div>
        ) : (
          <div className="table-responsive">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>Violation Category</th>
                  <th>Target ID / Type</th>
                  <th>Reporter Notes</th>
                  <th>Reporter Footprint</th>
                  <th>Status</th>
                  <th>Timestamp</th>
                  <th style={{ textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {reports.map((r) => (
                  <tr key={r.id}>
                    <td>
                      <span style={{ 
                        display: 'inline-flex', 
                        alignItems: 'center', 
                        gap: '6px', 
                        color: r.reason === 'sexual_content' ? '#ef4444' : '#f59e0b',
                        fontWeight: '700',
                        fontSize: '0.85rem'
                      }}>
                        <ShieldAlert size={16} />
                        {r.reason.replace('_', ' ').toUpperCase()}
                      </span>
                    </td>
                    <td>
                      <div style={{ fontWeight: '600', color: '#fff' }}>{r.target_type.toUpperCase()}</div>
                      <div style={{ fontSize: '0.72rem', color: '#64748b' }}>{r.target_id}</div>
                    </td>
                    <td style={{ maxWidth: '240px' }}>
                      <div style={{ fontSize: '0.82rem', color: '#cbd5e1' }}>{r.details || 'No additional comment provided.'}</div>
                      {r.action_taken && (
                        <div style={{ fontSize: '0.75rem', color: '#38bdf8', marginTop: '4px' }}>
                          <strong>Action:</strong> {r.action_taken}
                        </div>
                      )}
                    </td>
                    <td>
                      <code style={{ fontSize: '0.75rem', color: '#a5b4fc', background: '#1e293b', padding: '2px 6px', borderRadius: '4px' }}>
                        {r.reporter_ip || 'IP Logged'}
                      </code>
                    </td>
                    <td>
                      <span className={`status-pill ${r.status}`}>
                        {r.status.replace('_', ' ')}
                      </span>
                    </td>
                    <td style={{ fontSize: '0.8rem', color: '#94a3b8' }}>
                      {new Date(r.created_at).toLocaleString()}
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                        {r.status === 'pending' ? (
                          <button 
                            className="btn btn-primary btn-sm"
                            onClick={() => setActionModalReport(r)}
                          >
                            Review & Resolve
                          </button>
                        ) : (
                          <span style={{ fontSize: '0.75rem', color: '#64748b' }}>Resolved</span>
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

      {/* Resolve Modal */}
      {actionModalReport && (
        <div className="modal-overlay" onClick={() => setActionModalReport(null)}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Resolve Grievance Report</h3>
              <button className="btn-logout" onClick={() => setActionModalReport(null)}>
                &times;
              </button>
            </div>

            <div className="modal-body">
              <div style={{ marginBottom: '16px' }}>
                <span className="form-label">Reported Reason</span>
                <p style={{ color: '#ef4444', fontWeight: '700' }}>
                  {actionModalReport.reason.replace('_', ' ').toUpperCase()}
                </p>
              </div>

              <div style={{ marginBottom: '16px' }}>
                <span className="form-label">Reporter Notes</span>
                <p style={{ color: '#cbd5e1', fontSize: '0.875rem' }}>
                  {actionModalReport.details || 'No extra notes provided.'}
                </p>
              </div>

              <div className="form-group">
                <label className="form-label">Resolution Notes / Action Taken</label>
                <textarea
                  className="form-textarea"
                  rows="3"
                  value={actionTakenText}
                  onChange={(e) => setActionTakenText(e.target.value)}
                  placeholder="Describe the action taken (e.g., Shop was quarantined, content removed, or report was false positive)..."
                />
              </div>
            </div>

            <div className="modal-footer" style={{ justifyContent: 'space-between' }}>
              <button 
                type="button" 
                className="btn btn-outline"
                onClick={() => handleResolve('dismissed')}
                disabled={submitting}
              >
                <XCircle size={16} /> Dismiss Report
              </button>

              <button 
                type="button" 
                className="btn btn-danger"
                onClick={() => handleResolve('action_taken')}
                disabled={submitting}
              >
                <CheckCircle2 size={16} /> Enforce Action & Resolve
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
