/**
 * Customer Khata (Udhar & Credit Book) Screen
 * Inspired by: Khatabook, OkCredit
 * 
 * Hinglish Hint:
 * - Grahako ke udhar aur jama ka bahi-khata
 * - 1-Click WhatsApp Payment Reminder button (wa.me link with pre-filled text)
 * - Color-coded passbook transactions (Red for Udhar, Green for Jama)
 */

import React, { useState, useEffect } from 'react';
import {
  BookOpen,
  Search,
  Plus,
  ArrowDownLeft,
  ArrowUpRight,
  Phone,
  MessageSquare,
  CheckCircle,
  AlertCircle,
  Share2,
} from 'lucide-react';
import { khataApi } from '../../api/khata.api';
import { useAuth } from '../../context/AuthContext';
import { AppLayout } from '../../components/layout/AppLayout';

export const KhataScreen = () => {
  const { shop } = useAuth();
  const [summary, setSummary] = useState({ total_outstanding_amount: 0, total_customers: 0 });
  const [customers, setCustomers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  // Selected customer passbook
  const [selectedCustomer, setSelectedCustomer] = useState(null);
  const [customerHistory, setCustomerHistory] = useState(null);
  const [loadingHistory, setLoadingHistory] = useState(false);

  // Modals
  const [showAddCreditModal, setShowAddCreditModal] = useState(false);
  const [showRecordPaymentModal, setShowRecordPaymentModal] = useState(false);

  // Forms
  const [creditForm, setCreditForm] = useState({
    customer_name: '',
    customer_mobile: '',
    amount: '',
    notes: '',
  });

  const [paymentForm, setPaymentForm] = useState({
    amount: '',
    payment_mode: 'cash',
    notes: '',
  });

  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState('');

  // Initial data load
  const loadKhata = async () => {
    try {
      setLoading(true);
      const [sumRes, custRes] = await Promise.all([
        khataApi.getSummary(),
        khataApi.listCustomers(search),
      ]);
      setSummary(sumRes || { total_outstanding_amount: 0, total_customers: 0 });
      setCustomers(custRes || []);
    } catch (err) {
      console.error('Khata load error:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadKhata();
  }, [search]);

  // Grahak ki passbook kholna
  const handleOpenCustomer = async (cust) => {
    try {
      setSelectedCustomer(cust);
      setLoadingHistory(true);
      const history = await khataApi.getCustomerKhataHistory(cust.customer_mobile);
      setCustomerHistory(history);
    } catch (err) {
      console.error('History load error:', err);
    } finally {
      setLoadingHistory(false);
    }
  };

  // WhatsApp Reminder Link Generator
  const getWhatsAppReminderUrl = (customer) => {
    const shopName = shop?.name || 'Hamari Dukan';
    const cleanPhone = customer.customer_mobile.replace(/[^0-9]/g, '');
    const phoneWithCountry = cleanPhone.length === 10 ? `91${cleanPhone}` : cleanPhone;
    const msg = encodeURIComponent(
      `Namaste ${customer.customer_name} ji! Aapka ${shopName} par kul ₹${customer.current_balance} ka baki udhar hai. Kripya samay par iska bhugtan karein. Dhanyawad!`
    );
    return `https://wa.me/${phoneWithCountry}?text=${msg}`;
  };

  // Naya Udhar Save karna
  const handleAddCredit = async (e) => {
    e.preventDefault();
    try {
      setActionLoading(true);
      setError('');
      await khataApi.recordCredit({
        ...creditForm,
        amount: Number(creditForm.amount),
      });
      setShowAddCreditModal(false);
      setCreditForm({ customer_name: '', customer_mobile: '', amount: '', notes: '' });
      await loadKhata();
    } catch (err) {
      setError(err.message || 'Udhar add nahi ho saka');
    } finally {
      setActionLoading(false);
    }
  };

  // Payment Jama karna
  const handleRecordPayment = async (e) => {
    e.preventDefault();
    if (!selectedCustomer) return;
    try {
      setActionLoading(true);
      setError('');
      await khataApi.recordPayment(selectedCustomer.customer_mobile, {
        customer_name: selectedCustomer.customer_name,
        amount: Number(paymentForm.amount),
        payment_mode: paymentForm.payment_mode,
        notes: paymentForm.notes,
      });
      setShowRecordPaymentModal(false);
      setPaymentForm({ amount: '', payment_mode: 'cash', notes: '' });
      const history = await khataApi.getCustomerKhataHistory(selectedCustomer.customer_mobile);
      setCustomerHistory(history);
      await loadKhata();
    } catch (err) {
      setError(err.message || 'Payment record nahi ho saki');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <AppLayout title="Khata Book" subtitle="Grahak Udhar & Jama">
      {/* Total Udhar Summary Card */}
      <div
        className="card"
        style={{
          background: 'linear-gradient(135deg, #7f1d1d 0%, #991b1b 100%)',
          color: '#ffffff',
          border: 'none',
          padding: '18px',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <div style={{ fontSize: '0.75rem', opacity: 0.9, textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              Market Me Baki Total Udhar
            </div>
            <div style={{ fontSize: '1.75rem', fontWeight: 900, marginTop: '3px' }}>
              ₹{(summary.total_outstanding_amount || 0).toLocaleString('en-IN')}
            </div>
          </div>
          <button
            onClick={() => setShowAddCreditModal(true)}
            className="btn btn-sm"
            style={{
              backgroundColor: '#ffffff',
              color: '#991b1b',
              fontWeight: 800,
              gap: '5px',
            }}
          >
            <Plus size={16} /> Naya Udhar
          </button>
        </div>
      </div>

      {/* Customer Search Bar */}
      <div className="search-box">
        <Search size={18} />
        <input
          type="text"
          placeholder="Grahak ka naam ya phone khojein..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {/* Customer List */}
      <div className="card" style={{ padding: '8px 12px' }}>
        <div style={{ padding: '8px 4px', fontSize: '0.8rem', fontWeight: 800, color: 'var(--text-secondary)' }}>
          GRAHAK BAHI-KHATA ({customers.length})
        </div>

        {customers.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '28px', color: 'var(--text-muted)' }}>
            <BookOpen size={36} style={{ margin: '0 auto 8px auto', opacity: 0.6 }} />
            <div style={{ fontWeight: 700 }}>Koi udhar record nahi mila</div>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
              Upar "+ Naya Udhar" dabakar entry shuru karein.
            </p>
          </div>
        ) : (
          customers.map((c) => (
            <div
              key={c.id || c.customer_mobile}
              className="list-item card-clickable"
              onClick={() => handleOpenCustomer(c)}
              style={{ padding: '12px 0' }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <div className="list-avatar">
                  {c.customer_name?.charAt(0)?.toUpperCase() || 'C'}
                </div>
                <div>
                  <div style={{ fontWeight: 700, fontSize: '0.92rem' }}>{c.customer_name}</div>
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', display: 'flex', alignItems: 'center', gap: '4px' }}>
                    <Phone size={12} /> {c.customer_mobile}
                  </div>
                </div>
              </div>

              <div style={{ textAlign: 'right' }}>
                <div style={{ fontWeight: 900, fontSize: '1.05rem', color: 'var(--color-danger)' }}>
                  ₹{c.current_balance?.toLocaleString('en-IN')}
                </div>
                <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)' }}>
                  Baki Hai
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Customer Passbook Modal */}
      {selectedCustomer && (
        <div className="modal-backdrop" onClick={() => setSelectedCustomer(null)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <h3 style={{ fontSize: '1.25rem', fontWeight: 800 }}>{selectedCustomer.customer_name}</h3>
                <div style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>
                  {selectedCustomer.customer_mobile}
                </div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>Baki Balance:</div>
                <div style={{ fontSize: '1.4rem', fontWeight: 900, color: 'var(--color-danger)' }}>
                  ₹{selectedCustomer.current_balance?.toLocaleString('en-IN')}
                </div>
              </div>
            </div>

            {/* Quick Actions: Jama Karein + WhatsApp Reminder */}
            <div style={{ display: 'flex', gap: '8px', margin: '16px 0' }}>
              <button
                onClick={() => setShowRecordPaymentModal(true)}
                className="btn btn-success"
                style={{ flex: 1 }}
              >
                <ArrowDownLeft size={16} /> Paise Jama
              </button>

              <a
                href={getWhatsAppReminderUrl(selectedCustomer)}
                target="_blank"
                rel="noreferrer"
                className="btn btn-whatsapp"
                style={{ flex: 1, textDecoration: 'none' }}
              >
                <MessageSquare size={16} /> WhatsApp Reminder
              </a>
            </div>

            {/* Transactions Passbook */}
            <div style={{ fontSize: '0.8rem', fontWeight: 800, color: 'var(--text-secondary)', marginBottom: '8px' }}>
              LEN-DEN PASSBOOK HISTORY
            </div>

            {loadingHistory ? (
              <div style={{ textAlign: 'center', padding: '16px' }}>Passbook load ho rahi hai...</div>
            ) : (
              <div style={{ maxHeight: '200px', overflowY: 'auto' }}>
                {customerHistory?.transactions?.map((t) => (
                  <div
                    key={t.id}
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      padding: '10px 0',
                      borderBottom: '1px solid var(--border-subtle)',
                      fontSize: '0.85rem',
                    }}
                  >
                    <div>
                      <div style={{ fontWeight: 700 }}>
                        {t.transaction_type === 'credit' ? 'Udhar Diya' : 'Jama Hua (Payment)'}
                      </div>
                      <div style={{ fontSize: '0.72rem', color: 'var(--text-muted)' }}>
                        {t.notes || (t.transaction_type === 'payment' ? `Mode: ${t.payment_mode}` : 'POS Billing')}
                      </div>
                    </div>
                    <div style={{ textAlign: 'right' }}>
                      <span
                        style={{
                          fontWeight: 800,
                          fontSize: '0.95rem',
                          color: t.transaction_type === 'credit' ? 'var(--color-danger)' : 'var(--color-success)',
                        }}
                      >
                        {t.transaction_type === 'credit' ? '+' : '-'}₹{t.amount}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}

            <button
              className="btn btn-secondary btn-block"
              style={{ marginTop: '16px' }}
              onClick={() => setSelectedCustomer(null)}
            >
              Band Karein
            </button>
          </div>
        </div>
      )}

      {/* Add Credit Modal */}
      {showAddCreditModal && (
        <div className="modal-backdrop" onClick={() => setShowAddCreditModal(false)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <h3 style={{ fontSize: '1.15rem', fontWeight: 800, marginBottom: '12px' }}>Naya Udhar Jodein</h3>
            {error && (
              <div style={{ color: 'var(--color-danger)', fontSize: '0.82rem', marginBottom: '8px' }}>
                {error}
              </div>
            )}
            <form onSubmit={handleAddCredit}>
              <div className="form-group">
                <label className="form-label">Grahak Ka Naam</label>
                <input
                  type="text"
                  required
                  className="form-input"
                  placeholder="e.g. Ramesh Kumar"
                  value={creditForm.customer_name}
                  onChange={(e) => setCreditForm({ ...creditForm, customer_name: e.target.value })}
                />
              </div>

              <div className="form-group">
                <label className="form-label">Mobile Number</label>
                <input
                  type="tel"
                  required
                  className="form-input"
                  placeholder="9876543210"
                  value={creditForm.customer_mobile}
                  onChange={(e) => setCreditForm({ ...creditForm, customer_mobile: e.target.value })}
                />
              </div>

              <div className="form-group">
                <label className="form-label">Udhar Amount (₹)</label>
                <input
                  type="number"
                  step="0.01"
                  required
                  className="form-input"
                  placeholder="500"
                  value={creditForm.amount}
                  onChange={(e) => setCreditForm({ ...creditForm, amount: e.target.value })}
                />
              </div>

              <div className="form-group">
                <label className="form-label">Notes (Optional)</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder="e.g. Atta aur tel liya tha"
                  value={creditForm.notes}
                  onChange={(e) => setCreditForm({ ...creditForm, notes: e.target.value })}
                />
              </div>

              <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
                <button type="submit" className="btn btn-danger btn-block" disabled={actionLoading}>
                  {actionLoading ? 'Save ho raha hai...' : 'Udhar Darj Karein'}
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setShowAddCreditModal(false)}
                >
                  Radd
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Record Payment Modal */}
      {showRecordPaymentModal && (
        <div className="modal-backdrop" onClick={() => setShowRecordPaymentModal(false)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <h3 style={{ fontSize: '1.15rem', fontWeight: 800, marginBottom: '12px' }}>
              Paise Jama Karein ({selectedCustomer?.customer_name})
            </h3>
            {error && (
              <div style={{ color: 'var(--color-danger)', fontSize: '0.82rem', marginBottom: '8px' }}>
                {error}
              </div>
            )}
            <form onSubmit={handleRecordPayment}>
              <div className="form-group">
                <label className="form-label">Jama Amount (₹)</label>
                <input
                  type="number"
                  step="0.01"
                  required
                  className="form-input"
                  placeholder="Kitna paisa mila?"
                  value={paymentForm.amount}
                  onChange={(e) => setPaymentForm({ ...paymentForm, amount: e.target.value })}
                />
              </div>

              <div className="form-group">
                <label className="form-label">Payment Mode</label>
                <select
                  className="form-select"
                  value={paymentForm.payment_mode}
                  onChange={(e) => setPaymentForm({ ...paymentForm, payment_mode: e.target.value })}
                >
                  <option value="cash">💵 Cash (Nakad)</option>
                  <option value="upi">📱 UPI (GPay/PhonePe/Paytm)</option>
                  <option value="card">💳 Card</option>
                  <option value="other">Koyi Aur</option>
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">Notes (Optional)</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder="e.g. Aadha hisab chukta kiya"
                  value={paymentForm.notes}
                  onChange={(e) => setPaymentForm({ ...paymentForm, notes: e.target.value })}
                />
              </div>

              <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
                <button type="submit" className="btn btn-success btn-block" disabled={actionLoading}>
                  {actionLoading ? 'Jama ho raha hai...' : 'Jama Confirm Karein'}
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setShowRecordPaymentModal(false)}
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
