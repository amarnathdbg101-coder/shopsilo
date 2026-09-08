/**
 * Customer SideDrawer Component
 * 
 * Hinglish Hint:
 * Grahak app ka slide-out menu bar:
 * - Aas-paas ki dukaanein
 * - Meri Bookings & Holds
 * - Mera Account / Profile
 * - Support & Logout
 */

import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  X,
  Store,
  ShoppingBag,
  User,
  LogOut,
  ChevronRight,
  ShieldCheck,
} from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { getImageUrl } from '../../utils/imageUrl';
import { ThemeLanguageBar } from '../common/ThemeLanguageBar';

export const SideDrawer = ({ isOpen, onClose }) => {
  const { user, isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && isOpen) onClose();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  const handleLogout = () => {
    if (window.confirm('Kya aap sure hain ki aap logout karna chahte hain?')) {
      logout();
      onClose();
      navigate('/login');
    }
  };

  const handleNavigate = (path) => {
    navigate(path);
    onClose();
  };

  return (
    <>
      <div
        className={`drawer-backdrop ${isOpen ? 'active' : ''}`}
        onClick={onClose}
        aria-hidden={!isOpen}
      />

      <aside className={`side-drawer ${isOpen ? 'open' : ''}`}>
        <div className="drawer-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', minWidth: 0 }}>
            <div
              style={{
                width: '42px',
                height: '42px',
                borderRadius: '50%',
                background: user?.avatar_url ? 'transparent' : 'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)',
                color: '#ffffff',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontWeight: 800,
                fontSize: '1.1rem',
                boxShadow: '0 4px 12px rgba(59, 130, 246, 0.35)',
                flexShrink: 0,
                overflow: 'hidden',
              }}
            >
              {user?.avatar_url ? (
                <img
                  src={getImageUrl(user.avatar_url)}
                  alt={user?.name || 'Customer'}
                  style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                />
              ) : (
                user?.name?.charAt(0)?.toUpperCase() || 'G'
              )}
            </div>
            <div style={{ minWidth: 0, flex: 1 }}>
              <div style={{ fontWeight: 800, fontSize: '0.92rem', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {user ? user.name : 'Namaste, Guest'}
              </div>
              <div style={{ fontSize: '0.74rem', color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {user ? user.phone || user.email : 'Local Shopping Me Swagat Hai'}
              </div>
              <div style={{ marginTop: '2px' }}>
                <span
                  style={{
                    backgroundColor: '#dbeafe',
                    color: '#1e40af',
                    fontSize: '0.68rem',
                    fontWeight: 700,
                    padding: '2px 7px',
                    borderRadius: 'var(--radius-full)',
                    display: 'inline-block',
                  }}
                >
                  🛍️ Grahak
                </span>
              </div>
            </div>
          </div>

          <button onClick={onClose} className="drawer-close-btn" title="Menu Band Karein">
            <X size={20} />
          </button>
        </div>

        <div className="drawer-content">
          <div className="drawer-section-title">MARKETPLACE & ORDERS</div>
          <div className="drawer-links-group">
            <button className="drawer-link-btn" onClick={() => handleNavigate('/')}>
              <div className="drawer-icon-bubble" style={{ background: '#e0e7ff', color: '#4338ca' }}>
                <Store size={18} />
              </div>
              <div style={{ flex: 1, textAlign: 'left' }}>
                <div className="drawer-link-title">Aas-Paas Ki Dukaanein</div>
                <div className="drawer-link-sub">Explore nearby verified shops</div>
              </div>
              <ChevronRight size={16} color="var(--text-muted)" />
            </button>

            <button className="drawer-link-btn" onClick={() => handleNavigate('/reservations')}>
              <div className="drawer-icon-bubble" style={{ background: '#dcfce7', color: '#15803d' }}>
                <ShoppingBag size={18} />
              </div>
              <div style={{ flex: 1, textAlign: 'left' }}>
                <div className="drawer-link-title">My Pickups & Holds</div>
                <div className="drawer-link-sub">Reserved items & pickup OTP codes</div>
              </div>
              <ChevronRight size={16} color="var(--text-muted)" />
            </button>

            <button className="drawer-link-btn" onClick={() => handleNavigate(isAuthenticated ? '/profile' : '/login')}>
              <div className="drawer-icon-bubble" style={{ background: '#f1f5f9', color: '#334155' }}>
                <User size={18} />
              </div>
              <div style={{ flex: 1, textAlign: 'left' }}>
                <div className="drawer-link-title">My Profile & Account</div>
                <div className="drawer-link-sub">{isAuthenticated ? 'Details & photo update' : 'Login ya register karein'}</div>
              </div>
              <ChevronRight size={16} color="var(--text-muted)" />
            </button>
          </div>

          <div style={{ marginTop: '20px' }}>
            <div className="drawer-section-title">THEME & LANGUAGE</div>
            <ThemeLanguageBar />
          </div>
        </div>

        <div className="drawer-footer">
          {isAuthenticated ? (
            <button className="btn btn-outline btn-block" onClick={handleLogout} style={{ gap: '8px', color: 'var(--color-danger)', borderColor: 'rgba(239, 68, 68, 0.3)' }}>
              <LogOut size={16} />
              <span>Logout Karein</span>
            </button>
          ) : (
            <button className="btn btn-primary btn-block" onClick={() => handleNavigate('/login')}>
              <span>Login / Account Banayein</span>
            </button>
          )}
          <div style={{ textAlign: 'center', fontSize: '0.68rem', color: 'var(--text-muted)', marginTop: '10px' }}>
            ShopMe Customer Portal • v1.0
          </div>
        </div>
      </aside>
    </>
  );
};
