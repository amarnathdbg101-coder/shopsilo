/**
 * Customer App Bottom Navigation Bar
 * 
 * Ported from Flutter APK (QuickPick) tabs:
 * 1. Dukaanein (Explore Nearby Shops)
 * 2. Deals (Live Offers & Discounts Near You)
 * 3. Saved (Wishlist Products & Shops)
 * 4. Bookings (Customer Reservations & Pickups)
 * 5. Profile (Account & Switch to Merchant Mode)
 */

import React from 'react';
import { NavLink } from 'react-router-dom';
import { Store, Tag, Heart, ShoppingBag, User } from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { useSaved } from '../../context/SavedContext';

export const BottomNav = () => {
  const { isAuthenticated } = useAuth();
  const { savedProducts, savedShops } = useSaved();
  const totalSaved = savedProducts.length + savedShops.length;

  return (
    <nav className="bottom-nav">
      <NavLink
        to="/"
        end
        className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
      >
        <Store size={20} />
        <span>Dukaanein</span>
      </NavLink>

      <NavLink
        to="/deals"
        className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
      >
        <Tag size={20} />
        <span>Deals</span>
      </NavLink>

      <NavLink
        to="/saved"
        className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
        style={{ position: 'relative' }}
      >
        <Heart size={20} />
        <span>Saved</span>
        {totalSaved > 0 && (
          <span
            style={{
              position: 'absolute',
              top: '4px',
              right: '18px',
              background: '#ef4444',
              color: '#fff',
              fontSize: '0.65rem',
              fontWeight: 800,
              width: '16px',
              height: '16px',
              borderRadius: '50%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            {totalSaved > 9 ? '9+' : totalSaved}
          </span>
        )}
      </NavLink>

      <NavLink
        to="/reservations"
        className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
      >
        <ShoppingBag size={20} />
        <span>Bookings</span>
      </NavLink>

      <NavLink
        to={isAuthenticated ? "/profile" : "/login"}
        className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
      >
        <User size={20} />
        <span>{isAuthenticated ? 'Profile' : 'Login'}</span>
      </NavLink>
    </nav>
  );
};
