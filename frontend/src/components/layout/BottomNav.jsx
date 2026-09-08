/**
 * Bottom Navigation Bar (Super-App Navigation)
 * 
 * Hinglish Hint:
 * Mobile app ka main navigation:
 * - Merchant ke liye: Home, POS, Khata, Stock, Profile (User Section)
 * - Customer ke liye: Shops, Pickup Orders, Profile
 */

import React from 'react';
import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  Receipt,
  BookOpen,
  Package,
  User,
  Store,
  ShoppingBag,
} from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { usePOS } from '../../context/POSContext';

export const BottomNav = () => {
  const { isMerchant, isAuthenticated, user } = useAuth();
  const { itemCount } = usePOS();

  if (isMerchant) {
    return (
      <nav className="bottom-nav">
        <NavLink
          to="/merchant"
          end
          className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
        >
          <LayoutDashboard size={20} />
          <span>Home</span>
        </NavLink>

        <NavLink
          to="/merchant/pos"
          className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
        >
          <Receipt size={20} />
          <span>POS</span>
          {itemCount > 0 && <span className="nav-badge">{itemCount}</span>}
        </NavLink>

        <NavLink
          to="/merchant/khata"
          className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
        >
          <BookOpen size={20} />
          <span>Khata</span>
        </NavLink>

        <NavLink
          to="/merchant/inventory"
          className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
        >
          <Package size={20} />
          <span>Stock</span>
        </NavLink>

        <NavLink
          to="/profile"
          className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
        >
          <User size={20} />
          <span>Profile</span>
        </NavLink>
      </nav>
    );
  }

  // Customer / Visitor View (Clean, essential navigation - Pickup accessible via Side Menu)
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
        to={isAuthenticated ? "/profile" : "/login"}
        className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
      >
        <User size={20} />
        <span>{isAuthenticated ? 'Profile' : 'Login'}</span>
      </NavLink>
    </nav>
  );
};
