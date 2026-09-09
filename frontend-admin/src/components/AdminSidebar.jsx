import React from 'react';
import { 
  ShieldCheck, 
  Store, 
  AlertTriangle, 
  Ban, 
  LayoutDashboard, 
  FolderTree, 
  Users, 
  LogOut,
  X
} from 'lucide-react';
import { useAdminAuth } from '../context/AdminAuthContext';

export const AdminSidebar = ({ activeTab, setActiveTab, stats, mobileOpen, onCloseMobile }) => {
  const { adminUser, logout } = useAdminAuth();

  const menuItems = [
    { id: 'dashboard', label: 'Overview', icon: LayoutDashboard },
    { 
      id: 'shops', 
      label: 'Shop Approvals', 
      icon: Store, 
      badge: (stats?.pending_shops || 0) + (stats?.flagged_shops || 0) 
    },
    { 
      id: 'categories', 
      label: 'Categories', 
      icon: FolderTree, 
      badge: stats?.total_categories || 0,
      badgeColor: 'neutral'
    },
    { 
      id: 'users', 
      label: 'Users & Staff', 
      icon: Users, 
      badge: stats?.total_users || 0,
      badgeColor: 'neutral'
    },
    { 
      id: 'reports', 
      label: 'Grievance Desk', 
      icon: AlertTriangle, 
      badge: stats?.pending_reports || 0, 
      badgeColor: 'danger' 
    },
    { 
      id: 'blacklist', 
      label: 'Blacklist Vault', 
      icon: Ban, 
      count: stats?.banned_entities || 0 
    },
  ];

  const handleSelectTab = (id) => {
    setActiveTab(id);
    if (onCloseMobile) {
      onCloseMobile();
    }
  };

  return (
    <aside className={`admin-sidebar ${mobileOpen ? 'mobile-open' : ''}`}>
      <div className="sidebar-brand">
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div className="brand-icon">
            <ShieldCheck size={22} />
          </div>
          <div className="brand-text">
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <h2>ShopMe</h2>
              <span className="brand-badge">Admin</span>
            </div>
            <p style={{ fontSize: '0.72rem', color: '#64748b' }}>Control & Safety OS</p>
          </div>
        </div>

        {onCloseMobile && (
          <button 
            className="mobile-close-btn"
            onClick={onCloseMobile}
            aria-label="Close Navigation"
          >
            <X size={20} />
          </button>
        )}
      </div>

      <nav className="sidebar-menu">
        {menuItems.map((item) => {
          const Icon = item.icon;
          const isActive = activeTab === item.id;
          return (
            <div
              key={item.id}
              className={`menu-item ${isActive ? 'active' : ''}`}
              onClick={() => handleSelectTab(item.id)}
            >
              <div className="menu-item-left">
                <Icon size={18} />
                <span>{item.label}</span>
              </div>
              {item.badge !== undefined && item.badge > 0 && (
                <span 
                  className="sidebar-badge" 
                  style={{ 
                    background: item.badgeColor === 'danger' 
                      ? '#ef4444' 
                      : item.badgeColor === 'neutral' 
                        ? '#334155' 
                        : '#f59e0b',
                    color: item.badgeColor === 'neutral' ? '#cbd5e1' : '#fff'
                  }}
                >
                  {item.badge}
                </span>
              )}
            </div>
          );
        })}
      </nav>

      <div className="sidebar-footer">
        <div className="admin-profile">
          <div className="admin-avatar">
            {adminUser?.full_name ? adminUser.full_name[0].toUpperCase() : 'A'}
          </div>
          <div className="admin-info">
            <div className="admin-name">{adminUser?.full_name || 'Admin Master'}</div>
            <div className="admin-role">Super Admin</div>
          </div>
        </div>
        <button className="btn-logout" onClick={logout} title="Sign Out">
          <LogOut size={18} />
        </button>
      </div>
    </aside>
  );
};

