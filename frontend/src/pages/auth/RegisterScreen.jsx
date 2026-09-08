/**
 * Register Screen
 * 
 * Hinglish Hint:
 * Naye Dukaandar ya Grahak ke registration ke liye form:
 * - Full Name, Phone, Email aur Password
 * - Success hone par auto login aur dashboard redirect
 */

import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { UserPlus, ArrowRight, AlertCircle } from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { AppLayout } from '../../components/layout/AppLayout';

export const RegisterScreen = () => {
  const navigate = useNavigate();
  const { register } = useAuth();

  const [formData, setFormData] = useState({
    full_name: '',
    email: '',
    phone: '',
    password: '',
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (e) => {
    setFormData((prev) => ({ ...prev, [e.target.name]: e.target.value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await register(formData);
      // Auto-redirect to merchant onboarding/dashboard
      navigate('/merchant');
    } catch (err) {
      setError(err.message || 'Registration asafal raha, kripya details check karein');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppLayout title="ShopMe" subtitle="Naya Khata Banayein" hideNav={true} showBack={true}>
      <div style={{ paddingTop: '10px' }}>
        <div style={{ textAlign: 'center', marginBottom: '24px' }}>
          <div
            style={{
              width: '56px',
              height: '56px',
              backgroundColor: 'var(--color-primary-light)',
              color: 'var(--color-primary)',
              borderRadius: 'var(--radius-lg)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              margin: '0 auto 10px auto',
            }}
          >
            <UserPlus size={30} />
          </div>
          <h1 style={{ fontSize: '1.4rem', fontWeight: 800 }}>Account Banayein</h1>
          <p style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>
            Apni dukan ko digital banayein 2 minute me
          </p>
        </div>

        {error && (
          <div
            style={{
              backgroundColor: 'var(--color-danger-light)',
              color: 'var(--color-danger)',
              padding: '12px',
              borderRadius: 'var(--radius-md)',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              fontSize: '0.85rem',
              marginBottom: '16px',
            }}
          >
            <AlertCircle size={18} />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label">Aapka Pura Naam</label>
            <input
              type="text"
              name="full_name"
              required
              className="form-input"
              placeholder="e.g. Ramesh Kumar"
              value={formData.full_name}
              onChange={handleChange}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Mobile Number</label>
            <input
              type="tel"
              name="phone"
              required
              className="form-input"
              placeholder="e.g. 9876543210"
              value={formData.phone}
              onChange={handleChange}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Email Address</label>
            <input
              type="email"
              name="email"
              required
              className="form-input"
              placeholder="ramesh@dukaan.com"
              value={formData.email}
              onChange={handleChange}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Password (Minimum 6 akshar)</label>
            <input
              type="password"
              name="password"
              required
              minLength={6}
              className="form-input"
              placeholder="••••••••"
              value={formData.password}
              onChange={handleChange}
            />
          </div>

          <button
            type="submit"
            className="btn btn-primary btn-block btn-lg"
            disabled={loading}
            style={{ marginTop: '12px' }}
          >
            {loading ? 'Account ban raha hai...' : (
              <>
                Register Karein <ArrowRight size={18} />
              </>
            )}
          </button>
        </form>

        <div style={{ textAlign: 'center', marginTop: '20px', fontSize: '0.85rem' }}>
          <span style={{ color: 'var(--text-secondary)' }}>Pehle se account hai? </span>
          <Link
            to="/login"
            style={{
              color: 'var(--color-primary)',
              fontWeight: 600,
              textDecoration: 'none',
            }}
          >
            Login Karein
          </Link>
        </div>
      </div>
    </AppLayout>
  );
};
