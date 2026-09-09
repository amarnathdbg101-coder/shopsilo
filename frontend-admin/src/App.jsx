import React, { useState, useEffect } from 'react';
import { useAdminAuth } from './context/AdminAuthContext';
import { LoginScreen } from './pages/LoginScreen';
import { AdminSidebar } from './components/AdminSidebar';
import { DashboardOverview } from './pages/DashboardOverview';
import { ShopManagement } from './pages/ShopManagement';
import { CategoryManagement } from './pages/CategoryManagement';
import { UserManagement } from './pages/UserManagement';
import { ReportsDesk } from './pages/ReportsDesk';
import { BlacklistVault } from './pages/BlacklistVault';
import { adminApi } from './api/admin.api';
import { RefreshCw, Menu } from 'lucide-react';

export const AppContent = () => {
  const { adminUser, token, loading } = useAdminAuth();
  const [activeTab, setActiveTab] = useState('dashboard');
  const [stats, setStats] = useState(null);
  const [refreshing, setRefreshing] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  // Filter pass-throughs when navigating from dashboard
  const [shopFilter, setShopFilter] = useState('all');
  const [reportFilter, setReportFilter] = useState('pending');

  const fetchStats = async () => {
    try {
      setRefreshing(true);
      const data = await adminApi.getStats();
      setStats(data);
    } catch (err) {
      console.error('Failed to load stats:', err);
    } finally {
      setRefreshing(false);
    }
  };

  useEffect(() => {
    if (adminUser) {
      fetchStats();
      // Auto refresh stats every 30 seconds
      const interval = setInterval(fetchStats, 30000);
      return () => clearInterval(interval);
    }
  }, [adminUser]);

  const handleNavigateTab = (tab, filter) => {
    if (tab === 'shops' && filter) {
      setShopFilter(filter);
    }
    if (tab === 'reports' && filter) {
      setReportFilter(filter);
    }
    setActiveTab(tab);
  };

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#090d16', color: '#94a3b8' }}>
        <RefreshCw size={28} className="spin" />
      </div>
    );
  }

  if (!token || !adminUser) {
    return <LoginScreen />;
  }

  const getTabTitle = () => {
    switch (activeTab) {
      case 'dashboard': return 'Command Center Overview';
      case 'shops': return 'Shop Approvals & Moderation';
      case 'categories': return 'Product Categories Management';
      case 'users': return 'Registered Users & Merchants';
      case 'reports': return 'IT Rules 2021 Grievance Desk';
      case 'blacklist': return 'Ban Evasion Blacklist Vault';
      default: return 'Admin Console';
    }
  };

  return (
    <div className="admin-app">
      {/* Mobile Drawer Backdrop */}
      {mobileOpen && (
        <div 
          className="sidebar-backdrop" 
          onClick={() => setMobileOpen(false)}
        />
      )}

      <AdminSidebar
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        stats={stats}
        mobileOpen={mobileOpen}
        onCloseMobile={() => setMobileOpen(false)}
      />

      <div className="admin-main">
        <header className="admin-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
            <button
              className="mobile-menu-toggle"
              onClick={() => setMobileOpen(!mobileOpen)}
              aria-label="Toggle navigation menu"
            >
              <Menu size={22} />
            </button>

            <div className="header-title">
              <h1>{getTabTitle()}</h1>
            </div>
          </div>

          <div className="header-meta">
            <div className="live-indicator">
              <span className="pulse-dot"></span>
              <span>LIVE CLUSTER</span>
            </div>

            <button
              className="btn btn-outline btn-sm"
              onClick={fetchStats}
              title="Refresh Telemetry"
              style={{ padding: '6px 12px' }}
            >
              <RefreshCw size={14} className={refreshing ? 'spin' : ''} />
              <span>Refresh</span>
            </button>
          </div>
        </header>

        <main className="admin-page-content">
          {activeTab === 'dashboard' && (
            <DashboardOverview stats={stats} onNavigateTab={handleNavigateTab} />
          )}

          {activeTab === 'shops' && (
            <ShopManagement initialFilter={shopFilter} onRefreshStats={fetchStats} />
          )}

          {activeTab === 'categories' && (
            <CategoryManagement onRefreshStats={fetchStats} />
          )}

          {activeTab === 'users' && (
            <UserManagement onRefreshStats={fetchStats} />
          )}

          {activeTab === 'reports' && (
            <ReportsDesk initialStatus={reportFilter} onRefreshStats={fetchStats} />
          )}

          {activeTab === 'blacklist' && (
            <BlacklistVault onRefreshStats={fetchStats} />
          )}
        </main>
      </div>
    </div>
  );
};

