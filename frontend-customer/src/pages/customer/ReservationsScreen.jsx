/**
 * Customer Reservations (Pickup Orders) Screen
 * 
 * Hinglish Hint:
 * Grahak dwara hold kiye gaye saman aur unke Pickup OTP code:
 * - Counter pickup verification status
 * - Cancel reservation option
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { ShoppingBag, Clock, CheckCircle, XCircle, AlertCircle } from 'lucide-react';
import { reservationApi } from '../../api/reservation.api';
import { useAuth } from '../../context/AuthContext';
import { AppLayout } from '../../components/layout/AppLayout';

export const ReservationsScreen = () => {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  const [reservations, setReservations] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!isAuthenticated) {
      setLoading(false);
      return;
    }
    loadReservations();
  }, [isAuthenticated]);

  const loadReservations = async () => {
    try {
      setLoading(true);
      const data = await reservationApi.listUserReservations();
      const list = Array.isArray(data?.reservations)
        ? data.reservations
        : Array.isArray(data)
        ? data
        : [];
      setReservations(list);
    } catch (err) {
      console.error('Failed to load reservations:', err);
      setReservations([]);
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = async (id) => {
    if (!window.confirm('Kya aap is reservation ko cancel karna chahte hain?')) return;
    try {
      await reservationApi.cancelUserReservation(id);
      await loadReservations();
    } catch (err) {
      alert('Cancel nahi ho saka: ' + err.message);
    }
  };

  const formatExpiry = (expiresAt) => {
    if (!expiresAt) return 'Today';
    try {
      const d = new Date(expiresAt);
      if (isNaN(d.getTime())) return 'Today';
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch (e) {
      return 'Today';
    }
  };

  if (!isAuthenticated) {
    return (
      <AppLayout title="My Pickups" subtitle="Item Reservations" showBack={true}>
        <div className="card" style={{ textAlign: 'center', padding: '32px' }}>
          <ShoppingBag size={40} color="var(--text-muted)" style={{ margin: '0 auto 8px auto' }} />
          <h2 style={{ fontSize: '1.1rem', fontWeight: 700 }}>Login Zaroori Hai</h2>
          <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
            Apne reserved saman aur pickup codes dekhne ke liye login karein.
          </p>
          <button className="btn btn-primary" onClick={() => navigate('/login')}>
            Login Page Par Jayein
          </button>
        </div>
      </AppLayout>
    );
  }

  return (
    <AppLayout title="My Pickups" subtitle="Item Hold & Pickup Codes" showBack={true}>
      <div className="card" style={{ padding: '8px 12px' }}>
        <div style={{ padding: '8px 4px', fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-secondary)' }}>
          AAPKE RESERVED ITEMS ({reservations.length})
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '24px', color: 'var(--text-muted)', fontSize: '0.85rem' }}>
            Reservations load ho rahe hain...
          </div>
        ) : reservations.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '24px', color: 'var(--text-muted)', fontSize: '0.85rem' }}>
            Aapne abhi koi saman hold nahi kiya hai. Kisi bhi dukan me jakar "Hold / Reserve" dabayein.
          </div>
        ) : (
          reservations.map((r) => {
            const displayId =
              r.reservation_number ||
              (typeof r.id === 'string' ? r.id.slice(0, 8) : r.id) ||
              'RES';

            return (
              <div key={r.id || Math.random()} className="list-item" style={{ padding: '12px 0' }}>
                <div>
                  <div style={{ fontWeight: 700, fontSize: '0.9rem' }}>
                    {r.product?.name || `Reservation #${displayId}`}
                  </div>
                  {r.shop?.name && (
                    <div style={{ fontSize: '0.78rem', color: 'var(--color-primary)', fontWeight: 600 }}>
                      🏪 {r.shop.name}
                    </div>
                  )}
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '2px' }}>
                    Qty: {r.quantity || 1} • Expiry: {formatExpiry(r.expires_at)}
                  </div>
                  <div style={{ marginTop: '6px' }}>
                    <span
                      style={{
                        backgroundColor: 'var(--color-primary-light)',
                        color: 'var(--color-primary)',
                        fontWeight: 800,
                        padding: '3px 8px',
                        borderRadius: 'var(--radius-sm)',
                        fontSize: '0.82rem',
                        letterSpacing: '1px',
                        display: 'inline-block',
                      }}
                    >
                      Pickup Code: {r.pickup_code || r.reservation_number || displayId}
                    </span>
                  </div>
                </div>

                <div style={{ textAlign: 'right' }}>
                  <span
                    className={`badge ${
                      r.status === 'verified' || r.status === 'completed'
                        ? 'badge-success'
                        : r.status === 'cancelled'
                        ? 'badge-danger'
                        : 'badge-warning'
                    }`}
                  >
                    {r.status || 'Pending Pickup'}
                  </span>
                  {(r.status === 'active' || r.status === 'pending') && (
                    <div style={{ marginTop: '8px' }}>
                      <button
                        onClick={() => handleCancel(r.id)}
                        style={{
                          background: 'transparent',
                          border: 'none',
                          color: 'var(--color-danger)',
                          fontSize: '0.72rem',
                          fontWeight: 600,
                          cursor: 'pointer',
                        }}
                      >
                        Cancel Karein
                      </button>
                    </div>
                  )}
                </div>
              </div>
            );
          })
        )}
      </div>
    </AppLayout>
  );
};
