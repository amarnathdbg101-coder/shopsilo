/**
 * Shop Expenses (Dukan ke Roz ke Kharche) Screen
 * 
 * Hinglish Hint:
 * Dukan ke daily kharche log karne ke liye:
 * - Chai-nashta, Bijli bill, Dukan rent, Staff salary, Packaging
 * - Category-wise kharcho ka hisab
 * - Roz ka kharcha add aur delete karna
 */

import React, { useState, useEffect } from 'react';
import {
  Wallet,
  Plus,
  Trash2,
  Coffee,
  Zap,
  Building,
  Users,
  Package,
  CircleDollarSign,
  AlertCircle,
} from 'lucide-react';
import { expenseApi } from '../../api/expense.api';
import { AppLayout } from '../../components/layout/AppLayout';

export const ExpenseScreen = () => {
  const [expenseData, setExpenseData] = useState({
    expenses: [],
    total_amount: 0,
    category_total: {},
  });
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);

  const [form, setForm] = useState({
    category: 'tea_snacks',
    amount: '',
    payment_method: 'cash',
    notes: '',
  });
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState('');

  const loadExpenses = async () => {
    try {
      setLoading(true);
      const data = await expenseApi.listExpenses();
      setExpenseData(data || { expenses: [], total_amount: 0, category_total: {} });
    } catch (err) {
      console.error('Failed to load expenses:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadExpenses();
  }, []);

  const handleCreateExpense = async (e) => {
    e.preventDefault();
    try {
      setActionLoading(true);
      setError('');
      await expenseApi.createExpense({
        ...form,
        amount: Number(form.amount),
      });
      setShowAddModal(false);
      setForm({ category: 'tea_snacks', amount: '', payment_method: 'cash', notes: '' });
      await loadExpenses();
    } catch (err) {
      setError(err.message || 'Kharcha add nahi ho saka');
    } finally {
      setActionLoading(false);
    }
  };

  const handleDeleteExpense = async (id) => {
    if (!window.confirm('Kya aap yeh kharcha delete karna chahte hain?')) return;
    try {
      await expenseApi.deleteExpense(id);
      await loadExpenses();
    } catch (err) {
      alert('Delete asafal: ' + err.message);
    }
  };

  // Helper for category icon and label
  const getCategoryMeta = (cat) => {
    switch (cat) {
      case 'tea_snacks':
        return { label: 'Chai / Nashta', icon: <Coffee size={16} /> };
      case 'electricity':
        return { label: 'Bijli Ka Bill', icon: <Zap size={16} /> };
      case 'rent':
        return { label: 'Dukan Ka Kiraya', icon: <Building size={16} /> };
      case 'staff_salary':
        return { label: 'Staff Ki Salary', icon: <Users size={16} /> };
      case 'packaging':
        return { label: 'Polythene / Packing', icon: <Package size={16} /> };
      default:
        return { label: 'Anya Kharcha', icon: <CircleDollarSign size={16} /> };
    }
  };

  return (
    <AppLayout title="Dukan Ke Kharche" subtitle="Roz Ka Kharcha Track Karein">
      {/* Total Expenses Card */}
      <div
        className="card"
        style={{
          background: 'linear-gradient(135deg, #78350f 0%, #92400e 100%)',
          color: '#ffffff',
          border: 'none',
          padding: '16px',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <div style={{ fontSize: '0.75rem', opacity: 0.9, textTransform: 'uppercase' }}>
              Iss Mahine Ka Kul Kharcha
            </div>
            <div style={{ fontSize: '1.6rem', fontWeight: 900, marginTop: '2px' }}>
              ₹{(expenseData.total_amount || 0).toLocaleString('en-IN')}
            </div>
          </div>
          <button
            onClick={() => setShowAddModal(true)}
            className="btn btn-sm"
            style={{
              backgroundColor: '#ffffff',
              color: '#92400e',
              fontWeight: 700,
              gap: '4px',
            }}
          >
            <Plus size={16} /> Kharcha Likhein
          </button>
        </div>
      </div>

      {/* Category Breakdown Chips */}
      {expenseData.category_total && Object.keys(expenseData.category_total).length > 0 && (
        <div style={{ display: 'flex', gap: '8px', overflowX: 'auto', paddingBottom: '8px', marginBottom: '12px' }}>
          {Object.entries(expenseData.category_total).map(([cat, total]) => {
            const meta = getCategoryMeta(cat);
            return (
              <div
                key={cat}
                style={{
                  backgroundColor: 'var(--bg-surface)',
                  border: '1px solid var(--border-subtle)',
                  borderRadius: 'var(--radius-md)',
                  padding: '8px 12px',
                  whiteSpace: 'nowrap',
                  fontSize: '0.75rem',
                }}
              >
                <div style={{ color: 'var(--text-secondary)', display: 'flex', alignItems: 'center', gap: '4px' }}>
                  {meta.icon} <span>{meta.label}</span>
                </div>
                <div style={{ fontWeight: 800, fontSize: '0.9rem', color: 'var(--text-primary)', marginTop: '2px' }}>
                  ₹{Number(total).toLocaleString('en-IN')}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Expenses List */}
      <div className="card" style={{ padding: '8px 12px' }}>
        <div style={{ padding: '8px 4px', fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-secondary)' }}>
          KHAARCHA SOOCHI ({expenseData.expenses?.length || 0})
        </div>

        {expenseData.expenses?.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '24px', color: 'var(--text-muted)', fontSize: '0.85rem' }}>
            Koi kharcha darj nahi hai. "+ Kharcha Likhein" dabakar add karein.
          </div>
        ) : (
          expenseData.expenses?.map((exp) => {
            const meta = getCategoryMeta(exp.category);
            return (
              <div
                key={exp.id}
                className="list-item"
                style={{ padding: '10px 0' }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                  <div
                    style={{
                      width: '36px',
                      height: '36px',
                      borderRadius: 'var(--radius-md)',
                      backgroundColor: 'var(--color-warning-light)',
                      color: 'var(--color-warning)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    {meta.icon}
                  </div>
                  <div>
                    <div style={{ fontWeight: 700, fontSize: '0.85rem' }}>{meta.label}</div>
                    <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>
                      {exp.notes || `Paid via ${exp.payment_method?.toUpperCase()}`} • {exp.expense_date?.slice(0, 10)}
                    </div>
                  </div>
                </div>

                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <span style={{ fontWeight: 800, fontSize: '0.95rem', color: 'var(--color-danger)' }}>
                    -₹{exp.amount}
                  </span>
                  <button
                    onClick={() => handleDeleteExpense(exp.id)}
                    style={{
                      border: 'none',
                      background: 'transparent',
                      color: 'var(--text-muted)',
                      cursor: 'pointer',
                      padding: '4px',
                    }}
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Add Expense Modal */}
      {showAddModal && (
        <div className="modal-backdrop" onClick={() => setShowAddModal(false)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <h3 style={{ fontSize: '1.1rem', fontWeight: 700, marginBottom: '12px' }}>
              Naya Kharcha Darj Karein
            </h3>
            {error && (
              <div style={{ color: 'var(--color-danger)', fontSize: '0.8rem', marginBottom: '8px' }}>
                {error}
              </div>
            )}
            <form onSubmit={handleCreateExpense}>
              <div className="form-group">
                <label className="form-label">Category (Kis Cheez Ka Kharcha?)</label>
                <select
                  className="form-select"
                  value={form.category}
                  onChange={(e) => setForm({ ...form, category: e.target.value })}
                >
                  <option value="tea_snacks">☕ Chai / Nashta</option>
                  <option value="electricity">⚡ Bijli Ka Bill</option>
                  <option value="rent">🏢 Dukan Ka Kiraya</option>
                  <option value="staff_salary">👥 Staff Salary / Dihaadi</option>
                  <option value="packaging">📦 Packaging / Theli</option>
                  <option value="other">Anya Kharcha</option>
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">Amount (Kitna Kharcha Hua ₹)</label>
                <input
                  type="number"
                  step="0.01"
                  required
                  className="form-input"
                  placeholder="e.g. 50"
                  value={form.amount}
                  onChange={(e) => setForm({ ...form, amount: e.target.value })}
                />
              </div>

              <div className="form-group">
                <label className="form-label">Payment Mode</label>
                <select
                  className="form-select"
                  value={form.payment_method}
                  onChange={(e) => setForm({ ...form, payment_method: e.target.value })}
                >
                  <option value="cash">💵 Cash (Galle Se Diya)</option>
                  <option value="upi">📱 Online UPI</option>
                  <option value="card">💳 Card</option>
                  <option value="other">Other</option>
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">Notes (Kisne Kharch Kiya ya Details)</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder="e.g. Dukan me grahako ke liye chai mangwayi"
                  value={form.notes}
                  onChange={(e) => setForm({ ...form, notes: e.target.value })}
                />
              </div>

              <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
                <button type="submit" className="btn btn-primary btn-block" disabled={actionLoading}>
                  {actionLoading ? 'Save ho raha hai...' : 'Kharcha Save Karein'}
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setShowAddModal(false)}
                >
                  Radd
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </AppLayout>
  );
};
