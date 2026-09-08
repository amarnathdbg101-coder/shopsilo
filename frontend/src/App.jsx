/**
 * ShopMe Main Application Router & State Wrapper
 * 
 * Hinglish Hint:
 * Yeh file hamari app ka central brain hai:
 * 1. Global AuthContext aur POSContext provide karti hai.
 * 2. Mobile-first routes manage karti hai:
 *    - Public / Customer Routes:
 *        '/'              -> Aas-paas ki dukanien khojna (ExploreShopsScreen)
 *        '/shop/:slug'    -> Dukan ka storefront (StorefrontScreen)
 *        '/reservations'  -> Customer item pickups (ReservationsScreen)
 *        '/login'         -> Login Screen
 *        '/register'      -> Register Screen
 *    - Protected Merchant OS Routes (Sirf Dukaandar ke liye):
 *        '/merchant'           -> Merchant Dashboard
 *        '/merchant/pos'       -> Fast Counter Billing POS
 *        '/merchant/khata'     -> Customer Udhar Book
 *        '/merchant/expenses'  -> Dukan Ke Roz Ke Kharche
 *        '/merchant/inventory' -> Stock & Wholesale Reorder PDF
 *        '/merchant/analytics' -> Real Net Pocket Profit
 */

import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { POSProvider } from './context/POSContext';

// Auth Pages
import { LoginScreen } from './pages/auth/LoginScreen';
import { RegisterScreen } from './pages/auth/RegisterScreen';

// Merchant Pages
import { DashboardScreen } from './pages/merchant/DashboardScreen';
import { POSScreen } from './pages/merchant/POSScreen';
import { KhataScreen } from './pages/merchant/KhataScreen';
import { ExpenseScreen } from './pages/merchant/ExpenseScreen';
import { InventoryScreen } from './pages/merchant/InventoryScreen';
import { AnalyticsScreen } from './pages/merchant/AnalyticsScreen';

// Customer Pages
import { ExploreShopsScreen } from './pages/customer/ExploreShopsScreen';
import { StorefrontScreen } from './pages/customer/StorefrontScreen';
import { ReservationsScreen } from './pages/customer/ReservationsScreen';
import { ProfileScreen } from './pages/merchant/ProfileScreen';

// Protected Route Guard (Login check)
const ProtectedMerchantRoute = ({ children }) => {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', color: 'white' }}>
        App shuru ho raha hai...
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
      <POSProvider>
        <BrowserRouter>
          <Routes>
            {/* Customer & Public Routes */}
            <Route path="/" element={<ExploreShopsScreen />} />
            <Route path="/shop/:slug" element={<StorefrontScreen />} />
            <Route path="/reservations" element={<ReservationsScreen />} />

            {/* Auth Routes */}
            <Route path="/login" element={<LoginScreen />} />
            <Route path="/register" element={<RegisterScreen />} />

            {/* Merchant Dashboard & Dukan OS (Protected) */}
            <Route
              path="/merchant"
              element={
                <ProtectedMerchantRoute>
                  <DashboardScreen />
                </ProtectedMerchantRoute>
              }
            />
            <Route
              path="/merchant/pos"
              element={
                <ProtectedMerchantRoute>
                  <POSScreen />
                </ProtectedMerchantRoute>
              }
            />
            <Route
              path="/merchant/khata"
              element={
                <ProtectedMerchantRoute>
                  <KhataScreen />
                </ProtectedMerchantRoute>
              }
            />
            <Route
              path="/merchant/expenses"
              element={
                <ProtectedMerchantRoute>
                  <ExpenseScreen />
                </ProtectedMerchantRoute>
              }
            />
            <Route
              path="/merchant/inventory"
              element={
                <ProtectedMerchantRoute>
                  <InventoryScreen />
                </ProtectedMerchantRoute>
              }
            />
            <Route
              path="/merchant/analytics"
              element={
                <ProtectedMerchantRoute>
                  <AnalyticsScreen />
                </ProtectedMerchantRoute>
              }
            />

            <Route
              path="/profile"
              element={
                <ProtectedMerchantRoute>
                  <ProfileScreen />
                </ProtectedMerchantRoute>
              }
            />

            {/* Fallback */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </POSProvider>
    </AuthProvider>
  );
}

export default App;
