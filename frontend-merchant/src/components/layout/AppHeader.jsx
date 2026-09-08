/**
 * Top App Header Component (Modern Glassy Style)
 * 
 * Hinglish Hint:
 * Top navigation bar:
 * - Dukan ka naam aur live status (Online/Offline)
 * - Back button
 * - User Avatar bubble (Tap karne par profile screen khulti hai)
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Menu } from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { SideDrawer } from './SideDrawer';
import { getImageUrl } from '../../utils/imageUrl';

export const AppHeader = ({ title, subtitle, showBack = false }) => {
  const navigate = useNavigate();
  const { user, shop, isMerchant } = useAuth();
  const [drawerOpen, setDrawerOpen] = useState(false);

  return (
    <>
      <header className="app-header">
        <div className="header-left">
          {showBack ? (
            <button
              onClick={() => navigate(-1)}
              style={{
                background: 'transparent',
                border: 'none',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                padding: '6px',
                borderRadius: 'var(--radius-sm)',
                color: 'var(--text-primary)',
              }}
              title="Peeche Jayein"
            >
              <ArrowLeft size={20} />
            </button>
          ) : (
            <button
              onClick={() => setDrawerOpen(true)}
              style={{
                background: 'transparent',
                border: 'none',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                padding: '6px',
                borderRadius: 'var(--radius-sm)',
                color: 'var(--text-primary)',
              }}
              title="Side Menu Kholein"
            >
              <Menu size={22} />
            </button>
          )}

          <div>
            <div className="header-title">
              {title || (shop ? shop.name : 'ShopMe')}
            </div>
            {subtitle ? (
              <div className="header-subtitle">{subtitle}</div>
            ) : shop ? (
              <div className="header-subtitle" style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
                <span
                  style={{
                    width: '7px',
                    height: '7px',
                    borderRadius: '50%',
                    backgroundColor: shop.is_active ? 'var(--color-success)' : 'var(--color-danger)',
                    display: 'inline-block',
                  }}
                />
                <span style={{ fontWeight: 600, color: shop.is_active ? '#065f46' : '#991b1b' }}>
                  {shop.is_active ? 'Online (Khuli Hai)' : 'Offline (Band)'}
                </span>
              </div>
            ) : user ? (
              <div className="header-subtitle">Namaste, {user.full_name}</div>
            ) : null}
          </div>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          {/* Quick Menu Button when back button is active */}
          {showBack && (
            <button
              onClick={() => setDrawerOpen(true)}
              style={{
                background: 'transparent',
                border: 'none',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                padding: '6px',
                color: 'var(--text-secondary)',
              }}
              title="Menu"
            >
              <Menu size={20} />
            </button>
          )}

          {user ? (
            <button
              onClick={() => navigate('/profile')}
              style={{
                width: '34px',
                height: '34px',
                borderRadius: '50%',
                background: user.avatar_url ? 'transparent' : 'linear-gradient(135deg, #4f46e5 0%, #4338ca 100%)',
                color: '#ffffff',
                border: 'none',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontWeight: 800,
                fontSize: '0.85rem',
                boxShadow: '0 2px 8px rgba(79, 70, 229, 0.3)',
                overflow: 'hidden',
                padding: 0,
              }}
              title="Profile & Settings"
            >
              {user.avatar_url ? (
                <img
                  src={getImageUrl(user.avatar_url)}
                  alt={user?.name || user?.full_name}
                  style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                />
              ) : (
                (user?.name || user?.full_name)?.charAt(0)?.toUpperCase() || 'U'
              )}
            </button>
          ) : (
            <button
              onClick={() => navigate('/login')}
              className="btn btn-primary btn-sm"
            >
              Login
            </button>
          )}
        </div>
      </header>

      {/* Side Navigation Drawer */}
      <SideDrawer isOpen={drawerOpen} onClose={() => setDrawerOpen(false)} />
    </>
  );
};
