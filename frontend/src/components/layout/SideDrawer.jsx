/**
 * SideDrawer Component (Slide-out Side Navigation Menu)
 * 
 * Hinglish Hint:
 * Mobile app ka sliding side menu bar:
 * - Dukaandar aur grahak dono ke liye accessible.
 * - Bottom bar se hataaye gaye secondary features (Analytics, Daily Expenses,
 *   Customer Pickup Holds, Public Storefront Preview, Shop Profile & Settings)
 *   yahan categorized aur clean tarike se milte hain.
 */

import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  X,
  LayoutDashboard,
  Receipt,
  BookOpen,
  Package,
  TrendingUp,
  Wallet,
  ShoppingBag,
  Store,
  User,
  LogOut,
  ExternalLink,
  ChevronRight,
  ShieldCheck,
} from 'lucide-react';
import { useAuth } from '../../context/AuthContext';

export const SideDrawer = ({ isOpen, onClose }) => {
  const { user, shop, isMerchant, isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();

  // Close drawer on Escape key press
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  // Prevent background scrolling when drawer is open
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
      {/* Backdrop Overlay */}
      <div
        className={`drawer-backdrop ${isOpen ? 'active' : ''}`}
        onClick={onClose}
        aria-hidden={!isOpen}
      />

      {/* Drawer Container */}
      <aside className={`side-drawer ${isOpen ? 'open' : ''}`}>
        {/* Drawer Header */}
        <div className="drawer-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', minWidth: 0 }}>
            <div
              style={{
                width: '42px',
                height: '42px',
                borderRadius: '50%',
                background: user?.avatar_url ? 'transparent' : 'linear-gradient(135deg, #4f46e5 0%, #312e81 100%)',
                color: '#ffffff',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontWeight: 800,
                fontSize: '1.1rem',
                boxShadow: '0 4px 12px rgba(79, 70, 229, 0.35)',
                flexShrink: 0,
                overflow: 'hidden',
              }}
            >
              {user?.avatar_url ? (
                <img
                  src={user.avatar_url}
                  alt={user.full_name}
                  style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                />
              ) : (
                user?.full_name?.charAt(0)?.toUpperCase() || 'S'
              )}
            </div>
            <div style={{ minWidth: 0, flex: 1 }}>
              <div style={{ fontWeight: 800, fontSize: '0.92rem', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {user ? user.full_name : 'Namaste, Guest'}
              </div>
              <div style={{ fontSize: '0.74rem', color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {user ? user.phone || user.email : 'Dukan / Shopping Me Swagat Hai'}
              </div>
              <div style={{ marginTop: '2px' }}>
                <span
                  style={{
                    backgroundColor: isMerchant ? '#e0e7ff' : '#f1f5f9',
                    color: isMerchant ? '#4338ca' : '#475569',
                    fontSize: '0.68rem',
                    fontWeight: 700,
                    padding: '2px 7px',
                    borderRadius: 'var(--radius-full)',
                    display: 'inline-block',
                  }}
                >
                  {isMerchant ? '🏪 Dukaandar' : '🛍️ Grahak'}
                </span>
              </div>
            </div>
          </div>

          <button
            onClick={onClose}
            className="drawer-close-btn"
            title="Menu Band Karein"
          >
            <X size={20} />
          </button>
        </div>

        {/* Shop Live Status Card (If merchant) */}
        {shop && (
          <div className="drawer-shop-card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div style={{ minWidth: 0, flex: 1, marginRight: '8px' }}>
                <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.5px', fontWeight: 700 }}>
                  Aapki Dukan
                </div>
                <div style={{ fontWeight: 700, fontSize: '0.9rem', color: 'var(--text-primary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {shop.name}
                </div>
              </div>
              <span
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '5px',
                  fontSize: '0.72rem',
                  fontWeight: 700,
                  padding: '3px 8px',
                  borderRadius: 'var(--radius-full)',
                  backgroundColor: shop.is_active ? 'rgba(16, 185, 129, 0.15)' : 'rgba(239, 68, 68, 0.15)',
                  color: shop.is_active ? '#065f46' : '#991b1b',
                  flexShrink: 0,
                }}
              >
                <span
                  style={{
                    width: '6px',
                    height: '6px',
                    borderRadius: '50%',
                    backgroundColor: shop.is_active ? '#10b981' : '#ef4444',
                  }}
                />
                {shop.is_active ? 'Khuli Hai' : 'Band'}
              </span>
            </div>
          </div>
        )}

        {/* Drawer Body Links */}
        <div className="drawer-content">
          {isMerchant ? (
            <>
              {/* Primary Counter Navigation */}
              <div className="drawer-section-title">MAIN BILLING & COUNTER</div>
              <div className="drawer-links-group">
                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/merchant')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#e0e7ff', color: '#4338ca' }}>
                    <LayoutDashboard size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Dashboard Overview</div>
                    <div className="drawer-link-sub">Aaj ki bikri aur summary</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>

                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/merchant/pos')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#dcfce7', color: '#15803d' }}>
                    <Receipt size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Fast POS Billing</div>
                    <div className="drawer-link-sub">Turant bill aur receipt banayein</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>

                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/merchant/inventory')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#ffedd5', color: '#c2410c' }}>
                    <Package size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Stock & Catalog</div>
                    <div className="drawer-link-sub">Products, low stock alerts</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>

                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/merchant/khata')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#fee2e2', color: '#b91c1c' }}>
                    <BookOpen size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Customer Khata</div>
                    <div className="drawer-link-sub">Udhar ledger & WhatsApp reminder</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>
              </div>

              {/* Secondary / Shop Operations Menu (Infrequent Features) */}
              <div className="drawer-section-title" style={{ marginTop: '16px' }}>
                DUKAN OPERATIONS & PROFIT
              </div>
              <div className="drawer-links-group">
                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/merchant/analytics')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#ecfdf5', color: '#047857' }}>
                    <TrendingUp size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Pocket Profit & Sales</div>
                    <div className="drawer-link-sub">Net munafa aur product matrix</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>

                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/merchant/expenses')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#fef3c7', color: '#b45309' }}>
                    <Wallet size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Dukan Ke Kharche</div>
                    <div className="drawer-link-sub">Rent, bijli, chai aur daily kharche</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>

                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/reservations')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#f3e8ff', color: '#7e22ce' }}>
                    <ShoppingBag size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Pickup Reservations</div>
                    <div className="drawer-link-sub">Grahako ke hold kiye saman</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>

                {shop?.slug && (
                  <button
                    className="drawer-link-btn"
                    onClick={() => handleNavigate(`/shop/${shop.slug}`)}
                  >
                    <div className="drawer-icon-bubble" style={{ background: '#e0f2fe', color: '#0369a1' }}>
                      <Store size={18} />
                    </div>
                    <div style={{ flex: 1, textAlign: 'left' }}>
                      <div className="drawer-link-title">Storefront Preview</div>
                      <div className="drawer-link-sub">Grahak aapki dukan kaise dekhte hain</div>
                    </div>
                    <ExternalLink size={16} color="var(--text-muted)" />
                  </button>
                )}

                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/profile')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#f1f5f9', color: '#334155' }}>
                    <User size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Shop Settings & Profile</div>
                    <div className="drawer-link-sub">Dukan QR code, timings, details</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>
              </div>
            </>
          ) : (
            <>
              {/* Customer View */}
              <div className="drawer-section-title">MARKETPLACE & ORDERS</div>
              <div className="drawer-links-group">
                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#e0e7ff', color: '#4338ca' }}>
                    <Store size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Aas-Paas Ki Dukaanein</div>
                    <div className="drawer-link-sub">Explore nearby local shops</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>

                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/reservations')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#dcfce7', color: '#15803d' }}>
                    <ShoppingBag size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">My Pickups & Holds</div>
                    <div className="drawer-link-sub">Reserved saman & pickup codes</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>

                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate(isAuthenticated ? '/profile' : '/login')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#f1f5f9', color: '#334155' }}>
                    <User size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">My Account</div>
                    <div className="drawer-link-sub">{isAuthenticated ? 'Profile & points' : 'Login ya register karein'}</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>

                <button
                  className="drawer-link-btn"
                  onClick={() => handleNavigate('/register')}
                >
                  <div className="drawer-icon-bubble" style={{ background: '#fef3c7', color: '#b45309' }}>
                    <ShieldCheck size={18} />
                  </div>
                  <div style={{ flex: 1, textAlign: 'left' }}>
                    <div className="drawer-link-title">Dukaandar Banein</div>
                    <div className="drawer-link-sub">Apni dukan ko digital banayein</div>
                  </div>
                  <ChevronRight size={16} color="var(--text-muted)" />
                </button>
              </div>
            </>
          )}
        </div>

        {/* Drawer Footer */}
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
            ShopMe OS • Production Ready v1.0
          </div>
        </div>
      </aside>
    </>
  );
};
