/**
 * Merchant Bottom Navigation Bar
 * 
 * Hinglish Hint:
 * Dukaandar ke liye dedicated counter bottom bar:
 * - Home (Dashboard overview)
 * - POS (Quick billing counter)
 * - Stock (Inventory & wholesale)
 * - Khata (Udhaar ledger)
 * - Profile (Shop settings)
 */

import React from 'react';
import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  Receipt,
  BookOpen,
  Package,
  User,
} from 'lucide-react';
import { usePOS } from '../../context/POSContext';

export const BottomNav = () => {
  const { itemCount } = usePOS();

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
        to="/merchant/inventory"
        className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
      >
        <Package size={20} />
        <span>Stock</span>
      </NavLink>

      <NavLink
        to="/merchant/khata"
        className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
      >
        <BookOpen size={20} />
        <span>Khata</span>
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
};
