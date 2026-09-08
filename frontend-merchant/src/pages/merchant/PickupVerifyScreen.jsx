/**
 * Shop Owner Pickup Counter Verification & Hold Orders Manager
 * 
 * Hinglish Hint:
 * Dukaandar ke counter ke liye dedicated pickup verification screen:
 * 1. Grahak counter par aakar apna 6-digit Pickup OTP code dikhata hai.
 * 2. Dukaandar code enter karke verify karta hai aur saman handover karta hai.
 * 3. Dukan ke sabhi active customer holds ki live list dikhti hai.
 */

import React, { useState, useEffect } from 'react';
import {
  ShieldCheck,
  Search,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
  ShoppingBag,
  User,
  Phone,
  RefreshCw,
} from 'lucide-react';
import { reservationApi } from '../../api/reservation.api';
import { AppLayout } from '../../components/layout/AppLayout';

export const PickupVerifyScreen = () => {
  const [pickupCode, setPickupCode] = useState('');
  const [verifying, setVerifying] = useState(false);
  const [verifyMessage, setVerifyMessage] = useState(null);
  const [reservations, setReservations] = useState([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('ALL'); // ALL, PENDING, FULFILLED, EXPIRED

  useEffect(() => {
    loadShopReservations();
  }, []);

  const loadShopReservations = async () => {
    try {
      setLoading(true);
      const data = await reservationApi.listShopReservations();
      const list = Array.isArray(data?.reservations)
        ? data.reservations
        : Array.isArray(data)
        ? data
        : [];
      setReservations(list);
    } catch (err) {
      console.error('Failed to load shop reservations:', err);
      setReservations([]);
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async (e) => {
    e?.preventDefault();
    if (!pickupCode.trim()) {
      alert('Kripya customer ka Pickup Code enter karein');
      return;
    }

    try {
      setVerifying(true);
      setVerifyMessage(null);
      const res = await reservationApi.verifyShopReservation({
        pickup_code: pickupCode.trim(),
      });
      setVerifyMessage({
        type: 'success',
        text: `Pickup Code Valid! Saman handover karein. (Order ID: ${res?.id || pickupCode})`,
      });
      setPickupCode('');
      loadShopReservations();
    } catch (err) {
      setVerifyMessage({
        type: 'error',
        text: err.message || 'Galat ya expired pickup code. Kripya dobara check karein.',
      });
    } finally {
      setVerifying(false);
    }
  };

  const filteredReservations = reservations.filter((r) => {
    if (filter === 'ALL') return true;
    return r.status === filter;
  });

  const getStatusBadge = (status) => {
    switch (status) {
      case 'PENDING':
        return (
          <span style={{ background: '#fef3c7', color: '#b45309', padding: '4px 10px', borderRadius: '12px', fontSize: '0.75rem', fontWeight: 700, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
            <Clock size={12} /> Counter Par Aana Hai
          </span>
        );
      case 'FULFILLED':
      case 'COLLECTED':
        return (
          <span style={{ background: '#dcfce7', color: '#15803d', padding: '4px 10px', borderRadius: '12px', fontSize: '0.75rem', fontWeight: 700, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
            <CheckCircle size={12} /> Saman Diya Gaya
          </span>
        );
      case 'EXPIRED':
        return (
          <span style={{ background: '#fee2e2', color: '#b91c1c', padding: '4px 10px', borderRadius: '12px', fontSize: '0.75rem', fontWeight: 700, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
            <XCircle size={12} /> Expire Ho Gaya
          </span>
        );
      default:
        return (
          <span style={{ background: '#f1f5f9', color: '#475569', padding: '4px 10px', borderRadius: '12px', fontSize: '0.75rem', fontWeight: 700 }}>
            {status}
          </span>
        );
    }
  };

  return (
    <AppLayout title="Pickup Counter Verification">
      <div style={{ padding: '1rem', maxWidth: '800px', margin: '0 auto' }}>

        {/* Counter Verification Card */}
        <div style={{ background: 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)', padding: '1.5rem', borderRadius: '16px', border: '1px solid rgba(255,255,255,0.08)', marginBottom: '1.5rem', boxShadow: '0 8px 24px rgba(0,0,0,0.3)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '1rem' }}>
            <div style={{ background: '#3b82f6', color: '#fff', padding: '8px', borderRadius: '10px' }}>
              <ShieldCheck size={24} />
            </div>
            <div>
              <h3 style={{ margin: 0, color: '#f8fafc', fontSize: '1.1rem', fontWeight: 700 }}>Customer Pickup Verify Karein</h3>
              <p style={{ margin: 0, color: '#94a3b8', fontSize: '0.8rem' }}>Grahak ka 6-digit pickup OTP enter karein</p>
            </div>
          </div>

          <form onSubmit={handleVerify} style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
            <input
              type="text"
              placeholder="e.g. 849201"
              value={pickupCode}
              onChange={(e) => setPickupCode(e.target.value.toUpperCase())}
              maxLength={8}
              style={{
                flex: 1,
                minWidth: '200px',
                padding: '0.85rem 1rem',
                borderRadius: '12px',
                background: '#0f172a',
                border: '1px solid #334155',
                color: '#fff',
                fontSize: '1.1rem',
                fontWeight: 700,
                letterSpacing: '2px',
                outline: 'none',
              }}
            />
            <button
              type="submit"
              disabled={verifying}
              style={{
                background: '#2563eb',
                color: 'white',
                border: 'none',
                padding: '0.85rem 1.5rem',
                borderRadius: '12px',
                fontWeight: 700,
                cursor: 'pointer',
                fontSize: '0.95rem',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '8px',
              }}
            >
              <CheckCircle size={18} />
              <span>{verifying ? 'Checking...' : 'Verify & Deliver'}</span>
            </button>
          </form>

          {verifyMessage && (
            <div
              style={{
                marginTop: '1rem',
                padding: '0.75rem 1rem',
                borderRadius: '10px',
                background: verifyMessage.type === 'success' ? 'rgba(34, 197, 94, 0.15)' : 'rgba(239, 68, 68, 0.15)',
                color: verifyMessage.type === 'success' ? '#4ade80' : '#f87171',
                border: `1px solid ${verifyMessage.type === 'success' ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                fontSize: '0.9rem',
                fontWeight: 600,
              }}
            >
              {verifyMessage.text}
            </div>
          )}
        </div>

        {/* Reservations List Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem', flexWrap: 'wrap', gap: '8px' }}>
          <div>
            <h3 style={{ margin: 0, color: '#f8fafc', fontSize: '1.05rem', fontWeight: 700 }}>
              Store Hold Orders ({reservations.length})
            </h3>
            <p style={{ margin: 0, color: '#94a3b8', fontSize: '0.8rem' }}>Grahako dwara hold kiye gaye items</p>
          </div>
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
            <button
              onClick={loadShopReservations}
              style={{
                background: '#1e293b',
                color: '#cbd5e1',
                border: '1px solid #334155',
                padding: '6px 12px',
                borderRadius: '8px',
                cursor: 'pointer',
                fontSize: '0.8rem',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
              }}
            >
              <RefreshCw size={14} className={loading ? 'spin' : ''} />
              <span>Refresh</span>
            </button>
          </div>
        </div>

        {/* Filter Tabs */}
        <div style={{ display: 'flex', gap: '8px', marginBottom: '1rem', overflowX: 'auto', paddingBottom: '4px' }}>
          {['ALL', 'PENDING', 'FULFILLED', 'EXPIRED'].map((tab) => (
            <button
              key={tab}
              onClick={() => setFilter(tab)}
              style={{
                padding: '6px 14px',
                borderRadius: '20px',
                fontSize: '0.8rem',
                fontWeight: 600,
                border: 'none',
                cursor: 'pointer',
                background: filter === tab ? '#2563eb' : '#1e293b',
                color: filter === tab ? '#ffffff' : '#94a3b8',
                transition: 'all 0.2s',
              }}
            >
              {tab === 'ALL' ? 'Sabhi Orders' : tab === 'PENDING' ? 'Pending Holds' : tab === 'FULFILLED' ? 'Completed' : 'Expired'}
            </button>
          ))}
        </div>

        {/* Orders List */}
        {loading ? (
          <div style={{ padding: '3rem', textAlign: 'center', color: '#94a3b8' }}>
            Orders load ho rahe hain...
          </div>
        ) : filteredReservations.length === 0 ? (
          <div style={{ padding: '3rem 1rem', textAlign: 'center', background: '#1e293b', borderRadius: '16px', color: '#94a3b8' }}>
            <ShoppingBag size={40} style={{ opacity: 0.3, marginBottom: '0.5rem' }} />
            <p style={{ margin: 0, fontWeight: 600 }}>Koi order nahi mila</p>
            <p style={{ margin: '4px 0 0 0', fontSize: '0.8rem' }}>Jab customer aapki dukan se hold karega, yahan dikhega.</p>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            {filteredReservations.map((item) => (
              <div
                key={item.id}
                style={{
                  background: '#1e293b',
                  borderRadius: '14px',
                  padding: '1rem',
                  border: '1px solid rgba(255,255,255,0.06)',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '0.6rem',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <div>
                    <h4 style={{ margin: 0, color: '#f8fafc', fontSize: '1rem', fontWeight: 700 }}>
                      {item.product_name || 'Product'} × {item.quantity || 1}
                    </h4>
                    <span style={{ fontSize: '0.75rem', color: '#94a3b8' }}>
                      Order Ref: {item.reservation_number || item.id?.slice(0, 8)}
                    </span>
                  </div>
                  {getStatusBadge(item.status)}
                </div>

                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: '1px solid rgba(255,255,255,0.06)', paddingTop: '0.6rem', fontSize: '0.82rem', color: '#cbd5e1' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <User size={14} style={{ color: '#60a5fa' }} />
                    <span>{item.user_name || 'Customer'}</span>
                    {item.user_phone && (
                      <span style={{ color: '#94a3b8', fontSize: '0.78rem' }}>({item.user_phone})</span>
                    )}
                  </div>
                  <div style={{ fontWeight: 700, color: '#34d399' }}>
                    ₹{item.total_amount || (item.price * item.quantity) || 0}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

      </div>
    </AppLayout>
  );
};
