/**
 * Merchant Dashboard Screen (Shopkeeper Home)
 * 
 * Hinglish Hint:
 * Dukaandar ka main control room:
 * - Dukan ki live status (Online/Offline) switch
 * - Aaj ki bikri, pending udhar aur low stock alert ka snapshot
 * - POS, Khata, Kharche, Stock aur Analytics ke quick tap buttons
 * - Agar shop abhi nahi bani hai toh 1-minute Shop Setup form
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Receipt,
  BookOpen,
  Wallet,
  Package,
  TrendingUp,
  QrCode,
  Power,
  PlusCircle,
  AlertTriangle,
  Store,
  ChevronRight,
  ExternalLink,
} from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { shopApi } from '../../api/shop.api';
import { posApi } from '../../api/pos.api';
import { khataApi } from '../../api/khata.api';
import { inventoryApi } from '../../api/inventory.api';
import { AppLayout } from '../../components/layout/AppLayout';

export const DashboardScreen = () => {
  const navigate = useNavigate();
  const { shop, refreshShop, user } = useAuth();

  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    todaySales: 0,
    totalUdhar: 0,
    lowStockCount: 0,
  });

  const [showQRModal, setShowQRModal] = useState(false);
  const [qrCodeData, setQrCodeData] = useState(null);

  // Shop Setup Form State (agar user ki koi shop nahi hai)
  const [newShop, setNewShop] = useState({
    name: '',
    category: 'General Store / Kirana',
    address: '',
    city: 'Darbhanga',
    phone: user?.phone || '',
    description: '',
  });
  const [creatingShop, setCreatingShop] = useState(false);
  const [setupError, setSetupError] = useState('');

  // Dashboard Metrics Load karna
  const loadDashboardData = async () => {
    try {
      setLoading(true);
      // Run in parallel for fast loading
      const [posSummary, khataSummary, lowStock] = await Promise.allSettled([
        posApi.getDailySummary(),
        khataApi.getSummary(),
        inventoryApi.getLowStockAlerts(),
      ]);

      setStats({
        todaySales: posSummary.status === 'fulfilled' ? posSummary.value?.total_revenue || 0 : 0,
        totalUdhar: khataSummary.status === 'fulfilled' ? khataSummary.value?.total_outstanding_amount || 0 : 0,
        lowStockCount: lowStock.status === 'fulfilled' ? lowStock.value?.total_low_stock_items || 0 : 0,
      });
    } catch (err) {
      console.error('Error fetching dashboard stats:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (shop) {
      loadDashboardData();
    } else {
      setLoading(false);
    }
  }, [shop]);

  // Shop Online/Offline toggle karna
  const handleToggleStatus = async () => {
    try {
      await shopApi.toggleShopStatus();
      await refreshShop();
    } catch (err) {
      alert('Status change asafal: ' + err.message);
    }
  };

  // Shop QR Code dekhna
  const handleOpenQR = async () => {
    try {
      setShowQRModal(true);
      const data = await shopApi.getMyShopQR();
      setQrCodeData(data);
    } catch (err) {
      console.error('Failed to get QR code', err);
    }
  };

  // Nayi Shop create karna
  const handleCreateShop = async (e) => {
    e.preventDefault();
    setCreatingShop(true);
    setSetupError('');
    try {
      await shopApi.createShop(newShop);
      await refreshShop();
    } catch (err) {
      setSetupError(err.message || 'Dukan create nahi ho saki');
    } finally {
      setCreatingShop(false);
    }
  };

  // Agar Shopkeeper ne abhi tak dukan register nahi ki hai:
  if (!shop) {
    return (
      <AppLayout title="Shop Setup" subtitle="Apni Dukan Register Karein">
        <div className="card" style={{ marginTop: '10px' }}>
          <div style={{ textAlign: 'center', marginBottom: '16px' }}>
            <div
              style={{
                width: '52px',
                height: '52px',
                borderRadius: '50%',
                backgroundColor: 'var(--color-primary-light)',
                color: 'var(--color-primary)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                margin: '0 auto 8px auto',
              }}
            >
              <Store size={28} />
            </div>
            <h2 style={{ fontSize: '1.2rem', fontWeight: 700 }}>Dukan Ki Details Bharein</h2>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
              Aapka POS terminal aur online storefront turant tayar ho jayega.
            </p>
          </div>

          {setupError && (
            <div
              style={{
                backgroundColor: 'var(--color-danger-light)',
                color: 'var(--color-danger)',
                padding: '10px',
                borderRadius: 'var(--radius-md)',
                fontSize: '0.85rem',
                marginBottom: '12px',
              }}
            >
              {setupError}
            </div>
          )}

          <form onSubmit={handleCreateShop}>
            <div className="form-group">
              <label className="form-label">Dukan Ka Naam</label>
              <input
                type="text"
                required
                className="form-input"
                placeholder="e.g. Gupta General Store"
                value={newShop.name}
                onChange={(e) => setNewShop({ ...newShop, name: e.target.value })}
              />
            </div>

            <div className="form-group">
              <label className="form-label">Category</label>
              <select
                className="form-select"
                value={newShop.category}
                onChange={(e) => setNewShop({ ...newShop, category: e.target.value })}
              >
                <option value="General Store / Kirana">General Store / Kirana</option>
                <option value="Electronics & Mobile">Electronics & Mobile</option>
                <option value="Clothing & Fashion">Clothing & Fashion</option>
                <option value="Pharmacy / Medical">Pharmacy / Medical</option>
                <option value="Bakery & Dairy">Bakery & Dairy</option>
                <option value="Hardware & Tools">Hardware & Tools</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">Dukan Ka Pura Pata (Address)</label>
              <input
                type="text"
                required
                className="form-input"
                placeholder="Shop No. 4, Main Bazar Road"
                value={newShop.address}
                onChange={(e) => setNewShop({ ...newShop, address: e.target.value })}
              />
            </div>

            <div className="form-group">
              <label className="form-label">City / Shahar</label>
              <input
                type="text"
                className="form-input"
                placeholder="e.g. Darbhanga"
                value={newShop.city}
                onChange={(e) => setNewShop({ ...newShop, city: e.target.value })}
              />
            </div>

            <button
              type="submit"
              className="btn btn-primary btn-block btn-lg"
              disabled={creatingShop}
              style={{ marginTop: '8px' }}
            >
              {creatingShop ? 'Tayar ho raha hai...' : 'Dukan Shuru Karein'}
            </button>
          </form>
        </div>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      {/* Top Banner Card with Live Status Toggle */}
      <div
        className="card"
        style={{
          background: 'linear-gradient(135deg, #1e1b4b 0%, #312e81 100%)',
          color: '#ffffff',
          border: 'none',
          padding: '18px',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <div style={{ fontSize: '0.75rem', opacity: 0.8, textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              Merchant Terminal
            </div>
            <h1 style={{ fontSize: '1.35rem', fontWeight: 800, marginTop: '2px' }}>
              {shop.name}
            </h1>
            <div style={{ fontSize: '0.8rem', opacity: 0.85, marginTop: '2px' }}>
              {shop.category} • {shop.city || 'Local'}
            </div>
          </div>

          <button
            onClick={handleToggleStatus}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              backgroundColor: shop.is_active ? 'rgba(16, 185, 129, 0.25)' : 'rgba(239, 68, 68, 0.25)',
              border: `1px solid ${shop.is_active ? '#10b981' : '#ef4444'}`,
              color: '#ffffff',
              padding: '6px 12px',
              borderRadius: 'var(--radius-full)',
              fontSize: '0.75rem',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            <Power size={14} color={shop.is_active ? '#10b981' : '#ef4444'} />
            <span>{shop.is_active ? 'Online (Khuli)' : 'Offline'}</span>
          </button>
        </div>

        {/* Quick QR & Storefront Link */}
        <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
          <button
            onClick={handleOpenQR}
            style={{
              flex: 1,
              backgroundColor: 'rgba(255, 255, 255, 0.12)',
              border: 'none',
              color: 'white',
              padding: '8px 12px',
              borderRadius: 'var(--radius-md)',
              fontSize: '0.8rem',
              fontWeight: 500,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '6px',
              cursor: 'pointer',
            }}
          >
            <QrCode size={16} />
            <span>Dukan QR Code</span>
          </button>

          <a
            href={`/shop/${shop.slug}`}
            target="_blank"
            rel="noreferrer"
            style={{
              flex: 1,
              backgroundColor: 'rgba(255, 255, 255, 0.12)',
              border: 'none',
              color: 'white',
              padding: '8px 12px',
              borderRadius: 'var(--radius-md)',
              fontSize: '0.8rem',
              fontWeight: 500,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '6px',
              textDecoration: 'none',
            }}
          >
            <ExternalLink size={16} />
            <span>Storefront Dekhein</span>
          </a>
        </div>
      </div>

      {/* Snapshot Metrics Grid */}
      <div className="stat-grid">
        <div className="stat-card">
          <div className="stat-label">Aaj Ki Bikri (Sales)</div>
          <div className="stat-value" style={{ color: 'var(--color-success)' }}>
            ₹{stats.todaySales.toLocaleString('en-IN')}
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-label">Baki Udhar (Khata)</div>
          <div className="stat-value" style={{ color: 'var(--color-danger)' }}>
            ₹{stats.totalUdhar.toLocaleString('en-IN')}
          </div>
        </div>
      </div>

      {/* Low Stock Warning Banner if items > 0 */}
      {stats.lowStockCount > 0 && (
        <div
          onClick={() => navigate('/merchant/inventory')}
          style={{
            backgroundColor: 'var(--color-warning-light)',
            border: '1px solid #fde68a',
            padding: '12px 14px',
            borderRadius: 'var(--radius-md)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: '16px',
            cursor: 'pointer',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <AlertTriangle size={20} color="var(--color-warning)" />
            <div>
              <div style={{ fontSize: '0.85rem', fontWeight: 700, color: '#92400e' }}>
                {stats.lowStockCount} Products Ka Stock Kam Hai!
              </div>
              <div style={{ fontSize: '0.75rem', color: '#b45309' }}>
                1-Click Wholesale Reorder Sheet download karein
              </div>
            </div>
          </div>
          <ChevronRight size={18} color="#92400e" />
        </div>
      )}

      {/* Main Feature Navigation (Big Touch Cards) */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {/* POS Counter Billing */}
        <div
          className="card card-clickable"
          onClick={() => navigate('/merchant/pos')}
          style={{ display: 'flex', alignItems: 'center', gap: '14px', padding: '16px', margin: 0 }}
        >
          <div
            style={{
              width: '46px',
              height: '46px',
              borderRadius: 'var(--radius-md)',
              backgroundColor: 'var(--color-primary-light)',
              color: 'var(--color-primary)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Receipt size={24} />
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 700, fontSize: '0.95rem' }}>Counter POS Billing</div>
            <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
              Barcode scan, tezi se parchi/bill banayein aur print karein
            </div>
          </div>
          <ChevronRight size={18} color="var(--text-muted)" />
        </div>

        {/* Khata Book */}
        <div
          className="card card-clickable"
          onClick={() => navigate('/merchant/khata')}
          style={{ display: 'flex', alignItems: 'center', gap: '14px', padding: '16px', margin: 0 }}
        >
          <div
            style={{
              width: '46px',
              height: '46px',
              borderRadius: 'var(--radius-md)',
              backgroundColor: 'var(--color-danger-light)',
              color: 'var(--color-danger)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <BookOpen size={24} />
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 700, fontSize: '0.95rem' }}>Customer Khata Book</div>
            <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
              Grahak udhar, jama aur passbook hisab-kitab
            </div>
          </div>
          <ChevronRight size={18} color="var(--text-muted)" />
        </div>

        {/* Expense Tracker */}
        <div
          className="card card-clickable"
          onClick={() => navigate('/merchant/expenses')}
          style={{ display: 'flex', alignItems: 'center', gap: '14px', padding: '16px', margin: 0 }}
        >
          <div
            style={{
              width: '46px',
              height: '46px',
              borderRadius: 'var(--radius-md)',
              backgroundColor: 'var(--color-warning-light)',
              color: 'var(--color-warning)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Wallet size={24} />
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 700, fontSize: '0.95rem' }}>Dukan Ke Roz Ke Kharche</div>
            <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
              Chai, bijli, dukan rent, staff salary entry
            </div>
          </div>
          <ChevronRight size={18} color="var(--text-muted)" />
        </div>

        {/* Stock & Catalog */}
        <div
          className="card card-clickable"
          onClick={() => navigate('/merchant/inventory')}
          style={{ display: 'flex', alignItems: 'center', gap: '14px', padding: '16px', margin: 0 }}
        >
          <div
            style={{
              width: '46px',
              height: '46px',
              borderRadius: 'var(--radius-md)',
              backgroundColor: 'var(--color-success-light)',
              color: 'var(--color-success)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Package size={24} />
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 700, fontSize: '0.95rem' }}>Stock & Saman Catalog</div>
            <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
              Stock check, naya maal add karna aur alert
            </div>
          </div>
          <ChevronRight size={18} color="var(--text-muted)" />
        </div>

        {/* Real Profit Intelligence */}
        <div
          className="card card-clickable"
          onClick={() => navigate('/merchant/analytics')}
          style={{ display: 'flex', alignItems: 'center', gap: '14px', padding: '16px', margin: 0 }}
        >
          <div
            style={{
              width: '46px',
              height: '46px',
              borderRadius: 'var(--radius-md)',
              backgroundColor: '#f3e8ff',
              color: '#7e22ce',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <TrendingUp size={24} />
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 700, fontSize: '0.95rem' }}>Munafa & Profit Intelligence</div>
            <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
              Sales minus Kharche = Real Pocket Profit
            </div>
          </div>
          <ChevronRight size={18} color="var(--text-muted)" />
        </div>
      </div>

      {/* QR Code Modal */}
      {showQRModal && (
        <div className="modal-backdrop" onClick={() => setShowQRModal(false)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <div style={{ textAlign: 'center', padding: '10px' }}>
              <h3 style={{ fontSize: '1.1rem', fontWeight: 700 }}>{shop.name} QR Code</h3>
              <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
                Isko counter par lagayein taaki grahak dukan dekh sakein
              </p>

              {qrCodeData?.qr_code_image || qrCodeData?.qr_image ? (
                <img
                  src={qrCodeData.qr_code_image || qrCodeData.qr_image}
                  alt="Shop QR"
                  style={{ width: '200px', height: '200px', margin: '0 auto', borderRadius: '12px' }}
                />
              ) : (
                <div
                  style={{
                    width: '180px',
                    height: '180px',
                    margin: '0 auto',
                    backgroundColor: 'var(--bg-surface-subtle)',
                    borderRadius: '12px',
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: '8px',
                  }}
                >
                  <QrCode size={80} color="var(--color-primary)" />
                  <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                    {shop.slug}
                  </span>
                </div>
              )}

              <button
                className="btn btn-secondary btn-block"
                style={{ marginTop: '20px' }}
                onClick={() => setShowQRModal(false)}
              >
                Band Karein
              </button>
            </div>
          </div>
        </div>
      )}
    </AppLayout>
  );
};
