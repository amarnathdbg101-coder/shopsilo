/**
 * Analytics & Profit Intelligence Screen
 * 
 * Hinglish Hint:
 * Dukaandar ko uski dukan ka ASAL MUNAFA (Real Pocket Profit) dikhata hai:
 * - Kul Bikri (Revenue) minus Kharid Cost = Gross Profit
 * - Gross Profit minus Dukan ke Roz ke Kharche = Real Pocket Profit (Asal Bachat)
 * - Best-Profitable Saman vs Dead-Stock (Purana pada maal)
 */

import React, { useState, useEffect } from 'react';
import {
  TrendingUp,
  Percent,
  CircleDollarSign,
  AlertCircle,
  Flame,
  Archive,
  ArrowUpRight,
  ArrowDownRight,
} from 'lucide-react';
import { analyticsApi } from '../../api/analytics.api';
import { AppLayout } from '../../components/layout/AppLayout';

export const AnalyticsScreen = () => {
  const [profitData, setProfitData] = useState(null);
  const [matrixData, setMatrixData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadAnalytics();
  }, []);

  const loadAnalytics = async () => {
    try {
      setLoading(true);
      const [profit, matrix] = await Promise.allSettled([
        analyticsApi.getMonthlyProfit(),
        analyticsApi.getProductMatrix(),
      ]);

      if (profit.status === 'fulfilled') setProfitData(profit.value);
      if (matrix.status === 'fulfilled') setMatrixData(matrix.value);
    } catch (err) {
      console.error('Analytics fetch error:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppLayout title="Munafa & Intelligence" subtitle="Dukan Ki Asal Kamayi">
      {/* Real Pocket Profit Hero Card */}
      <div
        className="card"
        style={{
          background: 'linear-gradient(135deg, #064e3b 0%, #047857 100%)',
          color: '#ffffff',
          border: 'none',
          padding: '18px',
        }}
      >
        <div style={{ fontSize: '0.75rem', opacity: 0.9, textTransform: 'uppercase' }}>
          Real Pocket Profit (Asal Munafa)
        </div>
        <div style={{ fontSize: '1.8rem', fontWeight: 900, marginTop: '4px' }}>
          ₹{(profitData?.net_pocket_profit || 0).toLocaleString('en-IN')}
        </div>
        <div style={{ fontSize: '0.8rem', opacity: 0.9, marginTop: '6px' }}>
          Margin: <strong>{(profitData?.average_margin_pct || 0).toFixed(1)}%</strong> • {profitData?.total_completed_sales || 0} Bills Cleared
        </div>
      </div>

      {/* Financial Formula Breakdown */}
      <div className="card" style={{ padding: '14px' }}>
        <div style={{ fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-secondary)', marginBottom: '10px' }}>
          HISAAB-KITAAB KA FORMULA
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', fontSize: '0.85rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--text-secondary)' }}>Kul Bikri (Revenue):</span>
            <span style={{ fontWeight: 700 }}>+₹{(profitData?.total_revenue || 0).toLocaleString('en-IN')}</span>
          </div>

          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--text-secondary)' }}>Maal Ki Kharid Cost:</span>
            <span style={{ fontWeight: 700, color: 'var(--color-danger)' }}>
              -₹{(profitData?.total_cost || 0).toLocaleString('en-IN')}
            </span>
          </div>

          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--text-secondary)' }}>Dukan Ke Roz Ke Kharche:</span>
            <span style={{ fontWeight: 700, color: 'var(--color-danger)' }}>
              -₹{(profitData?.total_expenses || 0).toLocaleString('en-IN')}
            </span>
          </div>

          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              borderTop: '1px solid var(--border-subtle)',
              paddingTop: '8px',
              marginTop: '4px',
            }}
          >
            <span style={{ fontWeight: 800 }}>Pocket Profit (Asal Bachat):</span>
            <span style={{ fontWeight: 900, color: 'var(--color-success)', fontSize: '1.05rem' }}>
              ₹{(profitData?.net_pocket_profit || 0).toLocaleString('en-IN')}
            </span>
          </div>
        </div>
      </div>

      {/* Best Selling / Profitable Items */}
      {matrixData?.best_profitable && matrixData.best_profitable.length > 0 && (
        <div className="card" style={{ padding: '12px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '10px' }}>
            <Flame size={18} color="var(--color-warning)" />
            <span style={{ fontWeight: 800, fontSize: '0.88rem' }}>SABSE ZYADA MUNAFA DENE WALE PRODUCTS</span>
          </div>

          {matrixData.best_profitable.slice(0, 5).map((item) => (
            <div
              key={item.product_id}
              className="list-item"
              style={{ padding: '8px 0' }}
            >
              <div>
                <div style={{ fontWeight: 700, fontSize: '0.85rem' }}>{item.name}</div>
                <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>
                  {item.total_sold_qty} bika • Margin: {item.profit_margin_pct?.toFixed(0)}%
                </div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <span style={{ fontWeight: 800, color: 'var(--color-success)', fontSize: '0.9rem' }}>
                  +₹{item.total_profit}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Dead Stock / Purana Pada Maal */}
      {matrixData?.old_dead_stock && matrixData.old_dead_stock.length > 0 && (
        <div className="card" style={{ padding: '12px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '10px' }}>
            <Archive size={18} color="var(--text-secondary)" />
            <span style={{ fontWeight: 800, fontSize: '0.88rem' }}>DEAD STOCK (PURANA PADA MAAL)</span>
          </div>
          <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginBottom: '8px' }}>
            In samano ko discount ya offer lagakar jaldi nikalne ki koshish karein.
          </p>

          {matrixData.old_dead_stock.slice(0, 5).map((item) => (
            <div
              key={item.product_id}
              className="list-item"
              style={{ padding: '8px 0' }}
            >
              <div>
                <div style={{ fontWeight: 700, fontSize: '0.85rem' }}>{item.name}</div>
                <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>
                  Stock: {item.current_stock} pcs • {item.days_in_stock} dino se pada hai
                </div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <span className="badge badge-warning">Fast Clear Karein</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </AppLayout>
  );
};
