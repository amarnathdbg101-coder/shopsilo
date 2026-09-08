/**
 * ShopMe Customer Web Application Router
 * 
 * Ported from Flutter APK (QuickPick) routes:
 * - '/'             -> Nearby discovery with GPS & Radius slider
 * - '/deals'        -> Live deals & promotional offers feed
 * - '/saved'        -> Saved items & favorite shops
 * - '/shop/:slug'   -> Dukan ka storefront, live items aur pickup booking
 * - '/reservations' -> Grahak ke hold kiye huye item pickup codes (OTP)
 * - '/profile'      -> Grahak ka profile aur account settings
 * - '/login'        -> Customer Login
 * - '/register'     -> Customer Signup
 */

import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { LocationProvider } from './context/LocationContext';
import { SavedProvider } from './context/SavedContext';

// Auth Pages
import { LoginScreen } from './pages/auth/LoginScreen';
import { RegisterScreen } from './pages/auth/RegisterScreen';

// Customer Pages
import { ExploreShopsScreen } from './pages/customer/ExploreShopsScreen';
import { DealsScreen } from './pages/customer/DealsScreen';
import { SavedScreen } from './pages/customer/SavedScreen';
import { StorefrontScreen } from './pages/customer/StorefrontScreen';
import { ReservationsScreen } from './pages/customer/ReservationsScreen';
import { CustomerProfileScreen } from './pages/customer/CustomerProfileScreen';

// Protected Route Guard for Customer Profile / Orders
const ProtectedCustomerRoute = ({ children }) => {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', color: 'white' }}>
        Loading...
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return children;
};

function App() {
  return (
    <AuthProvider>
      <LocationProvider>
        <SavedProvider>
          <BrowserRouter>
            <Routes>
              {/* Public Store Discovery, Deals & Shopping */}
              <Route path="/" element={<ExploreShopsScreen />} />
              <Route path="/deals" element={<DealsScreen />} />
              <Route path="/saved" element={<SavedScreen />} />
              <Route path="/shop/:slug" element={<StorefrontScreen />} />

              {/* Customer Pickups & Orders */}
              <Route path="/reservations" element={<ReservationsScreen />} />

              {/* Customer Profile */}
              <Route
                path="/profile"
                element={
                  <ProtectedCustomerRoute>
                    <CustomerProfileScreen />
                  </ProtectedCustomerRoute>
                }
              />

              {/* Auth Routes */}
              <Route path="/login" element={<LoginScreen />} />
              <Route path="/register" element={<RegisterScreen />} />

              {/* Fallback */}
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </BrowserRouter>
        </SavedProvider>
      </LocationProvider>
    </AuthProvider>
  );
}

export default App;
