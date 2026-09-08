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
import { ThemeProvider } from './context/ThemeContext';
import { LanguageProvider } from './context/LanguageContext';

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
import { OffersScreen } from './pages/merchant/OffersScreen';

import { ProfileScreen } from './pages/merchant/ProfileScreen';
import { PickupVerifyScreen } from './pages/merchant/PickupVerifyScreen';

// Protected Route Guard (Merchant Only Check)
const ProtectedMerchantRoute = ({ children }) => {
  const { isAuthenticated, loading, isMerchant } = useAuth();

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', color: 'white' }}>
        Dukan OS load ho raha hai...
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!isMerchant) {
    return (
      <div style={{ padding: '3rem 1.5rem', textAlign: 'center', color: '#f8fafc' }}>
        <h2 style={{ fontSize: '1.25rem', fontWeight: 600, color: '#f87171', marginBottom: '0.5rem' }}>Access Restricted</h2>
        <p style={{ color: '#94a3b8', fontSize: '0.9rem', marginBottom: '1.5rem' }}>Yeh portal sirf registered Dukaandaar (Shop Owner) ke liye hai.</p>
        <button 
          onClick={() => { localStorage.clear(); window.location.href = '/login'; }}
          style={{ background: '#2563eb', color: '#fff', border: 'none', padding: '0.6rem 1.2rem', borderRadius: '8px', cursor: 'pointer', fontWeight: 600 }}
        >
          Shop Owner Account se Login Karein
        </button>
      </div>
    );
  }

  return children;
};

function App() {
  return (
    <ThemeProvider>
      <LanguageProvider>
        <AuthProvider>
          <POSProvider>
            <BrowserRouter>
              <Routes>
                {/* Merchant OS Root: Default to Merchant Dashboard/POS */}
                <Route path="/" element={<Navigate to="/merchant" replace />} />

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
                <Route path="/pos" element={<Navigate to="/merchant/pos" replace />} />
                <Route
                  path="/merchant/khata"
                  element={
                    <ProtectedMerchantRoute>
                      <KhataScreen />
                    </ProtectedMerchantRoute>
                  }
                />
                <Route path="/khata" element={<Navigate to="/merchant/khata" replace />} />
                <Route
                  path="/merchant/expenses"
                  element={
                    <ProtectedMerchantRoute>
                      <ExpenseScreen />
                    </ProtectedMerchantRoute>
                  }
                />
                <Route path="/expenses" element={<Navigate to="/merchant/expenses" replace />} />
                <Route
                  path="/merchant/inventory"
                  element={
                    <ProtectedMerchantRoute>
                      <InventoryScreen />
                    </ProtectedMerchantRoute>
                  }
                />
                <Route path="/inventory" element={<Navigate to="/merchant/inventory" replace />} />
                <Route
                  path="/merchant/analytics"
                  element={
                    <ProtectedMerchantRoute>
                      <AnalyticsScreen />
                    </ProtectedMerchantRoute>
                  }
                />
                <Route path="/analytics" element={<Navigate to="/merchant/analytics" replace />} />

                <Route
                  path="/merchant/offers"
                  element={
                    <ProtectedMerchantRoute>
                      <OffersScreen />
                    </ProtectedMerchantRoute>
                  }
                />
                <Route path="/offers" element={<Navigate to="/merchant/offers" replace />} />

                <Route
                  path="/merchant/pickups"
                  element={
                    <ProtectedMerchantRoute>
                      <PickupVerifyScreen />
                    </ProtectedMerchantRoute>
                  }
                />
                <Route path="/reservations" element={<Navigate to="/merchant/pickups" replace />} />

                <Route
                  path="/profile"
                  element={
                    <ProtectedMerchantRoute>
                      <ProfileScreen />
                    </ProtectedMerchantRoute>
                  }
                />
                <Route path="/merchant/profile" element={<Navigate to="/profile" replace />} />

                {/* Fallback */}
                <Route path="*" element={<Navigate to="/merchant" replace />} />
              </Routes>
            </BrowserRouter>
          </POSProvider>
        </AuthProvider>
      </LanguageProvider>
    </ThemeProvider>
  );
}

export default App;
