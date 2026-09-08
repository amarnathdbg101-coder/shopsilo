/**
 * Customer Explore & Marketplace Screen
 * 
 * Features ported from Flutter APK (QuickPick):
 * - GPS Auto-detect & Radius Selector (1km, 3km, 5km, 10km, All)
 * - Open Now filter & Live OPEN/CLOSED shop badges
 * - Distance calculation in km/m based on device GPS
 * - In-stock product search & price comparison ("Below MRP")
 * - 1-Click Call Shop & GPS Directions
 * - "Pick" AI Shopping Co-Pilot widget
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Store,
  Search,
  MapPin,
  Navigation,
  Phone,
  Crosshair,
  Clock,
  Sparkles,
  Bot,
  Package,
  Heart,
  Filter,
  CheckCircle,
  Eye,
  ArrowRight,
} from 'lucide-react';
import { shopApi } from '../../api/shop.api';
import { productApi } from '../../api/product.api';
import { reservationApi } from '../../api/reservation.api';
import { AppLayout } from '../../components/layout/AppLayout';
import { useLocation } from '../../context/LocationContext';
import { useSaved } from '../../context/SavedContext';
import { calculateDistanceKm, formatDistance } from '../../utils/distance';
import { getImageUrl } from '../../utils/imageUrl';
import { CustomerCopilotModal } from '../../components/common/CustomerCopilotModal';
import { ProductDetailModal } from '../../components/common/ProductDetailModal';
import { ShopDetailModal } from '../../components/common/ShopDetailModal';

const CATEGORIES = ['All', 'Electronics', 'Kirana & Grocery', 'Pharmacy', 'Fashion', 'Home & Kitchen'];
const RADIUS_OPTIONS = [
  { label: '1 km', value: 1 },
  { label: '3 km', value: 3 },
  { label: '5 km', value: 5 },
  { label: '10 km', value: 10 },
  { label: 'All', value: 999 },
];

export const ExploreShopsScreen = () => {
  const navigate = useNavigate();
  const { coords, locationName, radiusKm, setRadiusKm, detectLocation, isDetecting } = useLocation();
  const { isProductSaved, toggleSaveProduct, isShopSaved, toggleSaveShop } = useSaved();

  const [shops, setShops] = useState([]);
  const [products, setProducts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [searchMode, setSearchMode] = useState('shops'); // 'shops' | 'products'
  const [selectedCategory, setSelectedCategory] = useState('All');
  const [openNowOnly, setOpenNowOnly] = useState(false);
  const [inStockOnly, setInStockOnly] = useState(false);
  const [isCopilotOpen, setIsCopilotOpen] = useState(false);
  const [inspectedProduct, setInspectedProduct] = useState(null);
  const [inspectedShop, setInspectedShop] = useState(null);

  const handleReserveFromModal = async ({ product, quantity, hold_hours, notes }) => {
    try {
      const res = await reservationApi.createReservation({
        product_id: product.id,
        quantity,
        hold_hours,
        notes,
      });
      alert(`Item safaltapoorvak reserve ho gaya! Pickup Code: ${res.pickup_code || res.reservation_number}`);
      setInspectedProduct(null);
    } catch (err) {
      alert(err.message || 'Reservation fail ho gaya');
    }
  };

  useEffect(() => {
    loadData();
  }, [selectedCategory, radiusKm]);

  const loadData = async () => {
    try {
      setLoading(true);
      const params = {
        lat: coords?.lat,
        lng: coords?.lng,
        radius_km: radiusKm < 999 ? radiusKm : undefined,
      };
      if (selectedCategory !== 'All') {
        params.category = selectedCategory;
      }

      const [shopData, prodData] = await Promise.allSettled([
        shopApi.listPublicShops(params),
        productApi.listProducts({ limit: 50 }),
      ]);

      let shopList = shopData.status === 'fulfilled'
        ? Array.isArray(shopData.value?.shops) ? shopData.value.shops : (Array.isArray(shopData.value) ? shopData.value : [])
        : [];

      // If strict radius returned 0 shops, fallback to all shops so customer never sees an empty screen
      if (shopList.length === 0 && params.radius_km) {
        try {
          const fallbackShops = await shopApi.listPublicShops({
            lat: coords?.lat,
            lng: coords?.lng,
            category: selectedCategory !== 'All' ? selectedCategory : undefined,
          });
          shopList = Array.isArray(fallbackShops?.shops) ? fallbackShops.shops : (Array.isArray(fallbackShops) ? fallbackShops : []);
        } catch (e) {
          console.warn('Fallback shops fetch error:', e);
        }
      }
      setShops(shopList);

      const prodList = prodData.status === 'fulfilled'
        ? Array.isArray(prodData.value?.products) ? prodData.value.products : []
        : [];
      setProducts(prodList);
    } catch (err) {
      console.error('Failed to load marketplace data:', err);
    } finally {
      setLoading(false);
    }
  };

  // Filter shops
  const filteredShops = shops.filter((s) => {
    const matchesSearch =
      (s.name || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (s.category || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (s.city || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (s.address || '').toLowerCase().includes(searchTerm.toLowerCase());

    const matchesOpen = openNowOnly ? Boolean(s.is_active) : true;
    return matchesSearch && matchesOpen;
  });

  // Filter products for product-search mode
  const filteredProducts = products.filter((p) => {
    const matchesSearch =
      (p.name || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (p.description || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (p.category_name || '').toLowerCase().includes(searchTerm.toLowerCase());

    const prodStock = Number(p.stock_quantity ?? p.inventory?.available_quantity ?? p.inventory?.quantity ?? 0);
    const matchesStock = inStockOnly ? prodStock > 0 : true;
    return matchesSearch && matchesStock;
  });

  return (
    <AppLayout title="QuickPick Local" subtitle="Find In-Stock Products Around You">
      {/* Location Bar & GPS Button */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          background: 'var(--bg-card)',
          padding: '10px 14px',
          borderRadius: '12px',
          marginBottom: '12px',
          border: '1px solid var(--border-subtle)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0 }}>
          <MapPin size={16} color="var(--color-primary)" style={{ flexShrink: 0 }} />
          <div style={{ minWidth: 0 }}>
            <div style={{ fontSize: '0.72rem', color: 'var(--text-muted)' }}>Location</div>
            <div style={{ fontSize: '0.85rem', fontWeight: 700, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {locationName}
            </div>
          </div>
        </div>

        <button
          onClick={detectLocation}
          disabled={isDetecting}
          style={{
            background: 'rgba(37, 99, 235, 0.1)',
            color: 'var(--color-primary)',
            border: 'none',
            padding: '6px 12px',
            borderRadius: '8px',
            fontSize: '0.75rem',
            fontWeight: 600,
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            cursor: 'pointer',
          }}
        >
          <Crosshair size={13} className={isDetecting ? 'spin' : ''} />
          {isDetecting ? 'Detecting...' : 'Detect GPS'}
        </button>
      </div>

      {/* Radius Filter Chips */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          marginBottom: '12px',
          overflowX: 'auto',
          scrollbarWidth: 'none',
        }}
      >
        <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>
          Radius:
        </span>
        {RADIUS_OPTIONS.map((opt) => (
          <button
            key={opt.value}
            onClick={() => setRadiusKm(opt.value)}
            style={{
              padding: '4px 10px',
              borderRadius: '16px',
              border: 'none',
              fontSize: '0.75rem',
              fontWeight: 600,
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              background: radiusKm === opt.value ? 'var(--color-primary)' : 'var(--bg-card)',
              color: radiusKm === opt.value ? '#fff' : 'var(--text-secondary)',
              border: '1px solid var(--border-subtle)',
            }}
          >
            {opt.label}
          </button>
        ))}

        <label
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '4px',
            fontSize: '0.75rem',
            color: openNowOnly ? 'var(--color-primary)' : 'var(--text-secondary)',
            marginLeft: 'auto',
            whiteSpace: 'nowrap',
            cursor: 'pointer',
            fontWeight: 600,
          }}
        >
          <input
            type="checkbox"
            checked={openNowOnly}
            onChange={(e) => setOpenNowOnly(e.target.checked)}
            style={{ cursor: 'pointer' }}
          />
          Open Now
        </label>
      </div>

      {/* Search Bar & Switcher */}
      <div style={{ marginBottom: '12px' }}>
        <div className="search-box" style={{ marginBottom: '8px' }}>
          <Search size={18} />
          <input
            type="text"
            placeholder={searchMode === 'shops' ? "Search shop name, area, or category..." : "Search products in local stock..."}
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>

        {/* Search Mode Toggle (Shops vs Products) */}
        <div style={{ display: 'flex', gap: '6px' }}>
          <button
            onClick={() => setSearchMode('shops')}
            style={{
              flex: 1,
              padding: '6px',
              borderRadius: '8px',
              border: 'none',
              fontSize: '0.78rem',
              fontWeight: 600,
              cursor: 'pointer',
              background: searchMode === 'shops' ? 'var(--color-primary)' : 'var(--bg-card)',
              color: searchMode === 'shops' ? '#fff' : 'var(--text-secondary)',
            }}
          >
            Dukaanein ({filteredShops.length})
          </button>
          <button
            onClick={() => setSearchMode('products')}
            style={{
              flex: 1,
              padding: '6px',
              borderRadius: '8px',
              border: 'none',
              fontSize: '0.78rem',
              fontWeight: 600,
              cursor: 'pointer',
              background: searchMode === 'products' ? 'var(--color-primary)' : 'var(--bg-card)',
              color: searchMode === 'products' ? '#fff' : 'var(--text-secondary)',
            }}
          >
            Products Nearby ({filteredProducts.length})
          </button>
        </div>
      </div>

      {/* Category Pills */}
      <div
        style={{
          display: 'flex',
          gap: '8px',
          overflowX: 'auto',
          paddingBottom: '6px',
          marginBottom: '14px',
          scrollbarWidth: 'none',
        }}
      >
        {CATEGORIES.map((cat) => (
          <button
            key={cat}
            onClick={() => setSelectedCategory(cat)}
            style={{
              padding: '5px 12px',
              borderRadius: '16px',
              border: 'none',
              fontSize: '0.75rem',
              fontWeight: 600,
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              background: selectedCategory === cat ? 'var(--color-primary)' : 'var(--bg-card)',
              color: selectedCategory === cat ? '#fff' : 'var(--text-secondary)',
            }}
          >
            {cat}
          </button>
        ))}
      </div>

      {/* Content Feed */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px', color: 'var(--text-muted)' }}>
          Aas-paas ki physical dukaanein load ho rahi hain...
        </div>
      ) : searchMode === 'shops' ? (
        /* SHOPS LIST */
        filteredShops.length === 0 ? (
          <div className="card" style={{ textAlign: 'center', padding: '36px', borderRadius: '16px' }}>
            <Store size={44} color="var(--text-muted)" style={{ margin: '0 auto 10px auto', opacity: 0.6 }} />
            <div style={{ fontWeight: 800, fontSize: '1.05rem' }}>Koi Nazdeeki Dukan Nahi Mili</div>
            <p style={{ fontSize: '0.84rem', color: 'var(--text-secondary)', marginTop: '4px' }}>
              Upar diye gaye Radius ko badhayein ya "All" chunein.
            </p>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            {filteredShops.map((s) => {
              const distKm = calculateDistanceKm(coords?.lat, coords?.lng, s.latitude, s.longitude);
              const saved = isShopSaved(s.id);
              const hasBanner = Array.isArray(s.banners) && s.banners.length > 0 && s.banners[0];

              return (
                <div
                  key={s.id}
                  className="card card-clickable"
                  onClick={() => setInspectedShop(s)}
                  style={{
                    margin: 0,
                    padding: '18px',
                    borderRadius: '18px',
                    border: '1.5px solid var(--border-subtle)',
                    boxShadow: '0 6px 20px rgba(0,0,0,0.05)',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '14px',
                    transition: 'transform 0.15s ease, box-shadow 0.15s ease',
                    overflow: 'hidden',
                  }}
                >
                  {/* Optional Banner Image Preview if Shop has banner */}
                  {hasBanner && (
                    <div
                      style={{
                        margin: '-18px -18px 0 -18px',
                        height: '110px',
                        overflow: 'hidden',
                        position: 'relative',
                      }}
                    >
                      <img
                        src={getImageUrl(s.banners[0])}
                        alt={s.name}
                        style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                        onError={(e) => { e.currentTarget.parentElement.style.display = 'none'; }}
                      />
                      <div
                        style={{
                          position: 'absolute',
                          inset: 0,
                          background: 'linear-gradient(to bottom, rgba(0,0,0,0.05) 0%, rgba(0,0,0,0.45) 100%)',
                        }}
                      />
                    </div>
                  )}

                  {/* Top Row: Shop Logo, Name, Verified Badges */}
                  <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-start' }}>
                    <div
                      style={{
                        width: '74px',
                        height: '74px',
                        borderRadius: '16px',
                        backgroundColor: 'var(--color-primary-light)',
                        color: 'var(--color-primary)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        flexShrink: 0,
                        overflow: 'hidden',
                        border: '1.5px solid var(--border-subtle)',
                        boxShadow: '0 4px 12px rgba(0,0,0,0.08)',
                      }}
                    >
                      {s.logo_url ? (
                        <img
                          src={getImageUrl(s.logo_url)}
                          alt={s.name}
                          style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                          onError={(e) => { e.currentTarget.style.display = 'none'; }}
                        />
                      ) : (
                        <Store size={36} />
                      )}
                    </div>

                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '8px' }}>
                        <div>
                          <h3 style={{ fontWeight: 800, fontSize: '1.2rem', margin: 0, color: 'var(--text-primary)', lineHeight: 1.25 }}>
                            {s.name}
                          </h3>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', alignItems: 'center', marginTop: '4px' }}>
                            <span style={{ fontSize: '0.76rem', fontWeight: 700, color: 'var(--text-secondary)', background: 'var(--bg-app)', padding: '2px 8px', borderRadius: '6px' }}>
                              🏪 {s.category || 'General Store'}
                            </span>
                            <span style={{ fontSize: '0.72rem', fontWeight: 700, color: '#2563eb', background: 'rgba(37, 99, 235, 0.1)', padding: '2px 8px', borderRadius: '6px' }}>
                              🛡️ Verified Partner
                            </span>
                          </div>
                        </div>

                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleSaveShop(s);
                          }}
                          style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '4px', flexShrink: 0 }}
                          title="Favorite shop"
                        >
                          <Heart size={22} color={saved ? '#ef4444' : 'var(--text-muted)'} fill={saved ? '#ef4444' : 'none'} />
                        </button>
                      </div>

                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', alignItems: 'center', marginTop: '6px' }}>
                        <span
                          style={{
                            fontSize: '0.74rem',
                            fontWeight: 700,
                            padding: '3px 9px',
                            borderRadius: '6px',
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: '5px',
                            background: s.is_active ? 'rgba(16, 185, 129, 0.12)' : 'rgba(239, 68, 68, 0.12)',
                            color: s.is_active ? '#065f46' : '#991b1b',
                          }}
                        >
                          <span style={{ width: '7px', height: '7px', borderRadius: '50%', background: s.is_active ? '#10b981' : '#ef4444' }} />
                          {s.is_active ? 'OPEN NOW' : 'CURRENTLY CLOSED'}
                        </span>

                        <span style={{ fontSize: '0.76rem', color: 'var(--text-secondary)', display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                          <Clock size={13} color="var(--text-muted)" />
                          {s.opening_time || '09:00'} - {s.closing_time || '21:00'}
                        </span>
                      </div>

                      <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '5px', marginTop: '6px' }}>
                        <MapPin size={14} color="var(--color-primary)" style={{ flexShrink: 0 }} />
                        <span style={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {[s.address, s.city].filter(Boolean).join(', ') || 'Local Store Location'}
                        </span>
                        {distKm != null && (
                          <span style={{ fontWeight: 800, color: 'var(--color-primary)', flexShrink: 0 }}>
                            • {formatDistance(distKm)}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Feature Highlights Bar */}
                  <div
                    style={{
                      display: 'flex',
                      flexWrap: 'wrap',
                      gap: '8px',
                      padding: '8px 12px',
                      background: 'var(--bg-app)',
                      borderRadius: '10px',
                      fontSize: '0.75rem',
                      color: 'var(--text-secondary)',
                      alignItems: 'center',
                    }}
                  >
                    <span>⚡ Counter Pickup in 30 Mins</span>
                    <span>•</span>
                    <span>✓ Realtime Stock</span>
                    {s.phone && (
                      <>
                        <span>•</span>
                        <span>💬 WhatsApp Support</span>
                      </>
                    )}
                  </div>

                  {/* Action Buttons */}
                  <div
                    style={{
                      display: 'flex',
                      gap: '8px',
                      paddingTop: '10px',
                      borderTop: '1px solid var(--border-subtle)',
                      flexWrap: 'wrap',
                    }}
                    onClick={(e) => e.stopPropagation()}
                  >
                    <button
                      onClick={() => setInspectedShop(s)}
                      className="btn btn-secondary btn-sm"
                      style={{
                        flex: 1,
                        minWidth: '110px',
                        fontSize: '0.8rem',
                        fontWeight: 700,
                        display: 'inline-flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        gap: '5px',
                        padding: '9px 10px',
                        borderRadius: '10px',
                      }}
                    >
                      <Eye size={14} /> Dukan Details
                    </button>

                    {s.phone && (
                      <a
                        href={`tel:${s.phone}`}
                        className="btn btn-secondary btn-sm"
                        style={{
                          flex: 0.8,
                          minWidth: '85px',
                          fontSize: '0.8rem',
                          fontWeight: 700,
                          display: 'inline-flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          gap: '5px',
                          textDecoration: 'none',
                          padding: '9px 10px',
                          borderRadius: '10px',
                        }}
                      >
                        <Phone size={14} /> Call
                      </a>
                    )}

                    <button
                      onClick={() => navigate(`/shop/${s.slug}`)}
                      className="btn btn-primary btn-sm"
                      style={{
                        flex: 1.4,
                        minWidth: '140px',
                        fontSize: '0.84rem',
                        fontWeight: 700,
                        padding: '9px 12px',
                        borderRadius: '10px',
                        display: 'inline-flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        gap: '6px',
                      }}
                    >
                      <span>Storefront & Items</span>
                      <ArrowRight size={15} />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )
      ) : (
        /* PRODUCTS LIST (PRICE COMPARISON & STOCK) */
        filteredProducts.length === 0 ? (
          <div className="card" style={{ textAlign: 'center', padding: '36px', borderRadius: '16px' }}>
            <Package size={44} color="var(--text-muted)" style={{ margin: '0 auto 10px auto', opacity: 0.6 }} />
            <div style={{ fontWeight: 800, fontSize: '1.05rem' }}>Koi Product Nahi Mila</div>
            <p style={{ fontSize: '0.84rem', color: 'var(--text-secondary)', marginTop: '4px' }}>
              Dusre product name ya category se search karein.
            </p>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            {filteredProducts.map((p) => {
              const saved = isProductSaved(p.id);
              const stock = Number(p.stock_quantity ?? p.inventory?.available_quantity ?? p.inventory?.quantity ?? 0);
              const inStock = stock > 0;
              const hasDiscount = p.compare_price && p.compare_price > p.price;
              const discountPct = hasDiscount ? Math.round(((p.compare_price - p.price) / p.compare_price) * 100) : 0;
              const brand = p.attributes?.brand || p.attributes?.company;

              return (
                <div
                  key={p.id}
                  className="card card-clickable"
                  onClick={() => setInspectedProduct(p)}
                  style={{
                    margin: 0,
                    padding: '18px',
                    borderRadius: '18px',
                    border: '1.5px solid var(--border-subtle)',
                    boxShadow: '0 6px 20px rgba(0,0,0,0.05)',
                    display: 'flex',
                    gap: '18px',
                    alignItems: 'center',
                    cursor: 'pointer',
                    transition: 'transform 0.15s ease, box-shadow 0.15s ease',
                  }}
                >
                  {/* Big High-Res Product Image with Overlay Status */}
                  <div
                    style={{
                      width: '116px',
                      height: '116px',
                      borderRadius: '16px',
                      backgroundColor: '#ffffff',
                      overflow: 'hidden',
                      flexShrink: 0,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      border: '1.5px solid var(--border-subtle)',
                      boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
                      position: 'relative',
                    }}
                  >
                    {p.images && p.images[0] ? (
                      <img
                        src={getImageUrl(p.images[0])}
                        alt={p.name}
                        style={{ width: '100%', height: '100%', objectFit: 'contain', padding: '4px' }}
                        onError={(e) => { e.currentTarget.style.display = 'none'; }}
                      />
                    ) : (
                      <Package size={40} color="var(--text-muted)" style={{ opacity: 0.4 }} />
                    )}

                    {/* Stock Pill on image */}
                    <span
                      style={{
                        position: 'absolute',
                        bottom: '5px',
                        left: '5px',
                        right: '5px',
                        textAlign: 'center',
                        fontSize: '0.64rem',
                        fontWeight: 800,
                        padding: '2px 4px',
                        borderRadius: '6px',
                        backgroundColor: inStock ? 'rgba(16, 185, 129, 0.95)' : 'rgba(239, 68, 68, 0.95)',
                        color: '#ffffff',
                        boxShadow: '0 2px 4px rgba(0,0,0,0.15)',
                      }}
                    >
                      {inStock ? `${stock} in stock` : 'Out of stock'}
                    </span>
                  </div>

                  {/* Middle: Details */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '8px' }}>
                      <h4 style={{ fontWeight: 800, fontSize: '1.12rem', margin: 0, color: 'var(--text-primary)', lineHeight: 1.3 }}>
                        {p.name}
                      </h4>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleSaveProduct(p);
                        }}
                        style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '2px', flexShrink: 0 }}
                        title="Save product"
                      >
                        <Heart size={20} color={saved ? '#ef4444' : 'var(--text-muted)'} fill={saved ? '#ef4444' : 'none'} />
                      </button>
                    </div>

                    {/* Category & Brand & Attribute Chips Upfront */}
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', alignItems: 'center', marginTop: '6px' }}>
                      {brand && (
                        <span style={{ fontSize: '0.74rem', background: 'rgba(37, 99, 235, 0.09)', color: '#2563eb', padding: '2px 8px', borderRadius: '6px', fontWeight: 700 }}>
                          {brand}
                        </span>
                      )}
                      {p.category_name && (
                        <span style={{ fontSize: '0.74rem', background: 'var(--bg-app)', color: 'var(--text-secondary)', padding: '2px 8px', borderRadius: '6px', fontWeight: 600 }}>
                          {p.category_name}
                        </span>
                      )}
                      {p.attributes?.size && (
                        <span style={{ fontSize: '0.72rem', background: '#f1f5f9', color: 'var(--text-muted)', padding: '2px 6px', borderRadius: '4px', fontWeight: 600 }}>
                          Size: {p.attributes.size}
                        </span>
                      )}
                      {p.attributes?.weight && (
                        <span style={{ fontSize: '0.72rem', background: '#f1f5f9', color: 'var(--text-muted)', padding: '2px 6px', borderRadius: '4px', fontWeight: 600 }}>
                          {p.attributes.weight}
                        </span>
                      )}
                    </div>

                    {/* Pricing & Discount Upfront */}
                    <div style={{ display: 'flex', alignItems: 'baseline', gap: '8px', marginTop: '8px' }}>
                      <span style={{ fontWeight: 900, color: 'var(--color-primary)', fontSize: '1.25rem' }}>
                        ₹{p.price}
                      </span>
                      {hasDiscount && (
                        <>
                          <span style={{ fontSize: '0.82rem', color: 'var(--text-muted)', textDecoration: 'line-through' }}>
                            ₹{p.compare_price}
                          </span>
                          <span style={{ fontSize: '0.72rem', color: '#15803d', fontWeight: 800, background: '#dcfce7', padding: '2px 7px', borderRadius: '6px' }}>
                            Save ₹{p.compare_price - p.price} ({discountPct}% OFF)
                          </span>
                        </>
                      )}
                    </div>

                    {/* Seller Shop info & Stock */}
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-muted)', marginTop: '6px', display: 'flex', alignItems: 'center', gap: '5px', flexWrap: 'wrap' }}>
                      <Store size={14} color="var(--color-primary)" />
                      <span style={{ fontWeight: 700, color: 'var(--text-primary)' }}>{p.shop_name || 'Verified Shop'}</span>
                      {p.shop_city && <span>• {p.shop_city}</span>}
                      {inStock && (
                        <span style={{ color: '#10b981', fontWeight: 700 }}>
                          • Counter pickup ready
                        </span>
                      )}
                    </div>

                    {/* Action buttons */}
                    <div style={{ marginTop: '12px', display: 'flex', gap: '8px', flexWrap: 'wrap' }} onClick={(e) => e.stopPropagation()}>
                      <button
                        onClick={() => setInspectedProduct(p)}
                        className="btn btn-primary btn-sm"
                        style={{
                          fontSize: '0.8rem',
                          padding: '7px 14px',
                          borderRadius: '8px',
                          fontWeight: 700,
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '6px',
                        }}
                      >
                        <Eye size={14} /> View Details & Hold
                      </button>
                      {p.shop_slug && (
                        <button
                          onClick={() => navigate(`/shop/${p.shop_slug}`)}
                          className="btn btn-secondary btn-sm"
                          style={{ fontSize: '0.78rem', padding: '7px 12px', borderRadius: '8px', fontWeight: 600 }}
                        >
                          Visit Storefront →
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )
      )}

      {/* Product Detail Modal */}
      {inspectedProduct && (
        <ProductDetailModal
          product={inspectedProduct}
          onClose={() => setInspectedProduct(null)}
          onReserve={handleReserveFromModal}
        />
      )}

      {/* Shop Detail Modal */}
      {inspectedShop && (
        <ShopDetailModal
          shop={inspectedShop}
          onClose={() => setInspectedShop(null)}
        />
      )}

      {/* Floating "Ask Pick (AI Shopping Co-Pilot)" Button */}
      <button
        onClick={() => setIsCopilotOpen(true)}
        style={{
          position: 'fixed',
          bottom: '80px',
          right: '20px',
          background: 'linear-gradient(135deg, #2563eb, #7c3aed)',
          color: '#fff',
          border: 'none',
          borderRadius: '30px',
          padding: '10px 18px',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          fontSize: '0.88rem',
          fontWeight: 700,
          cursor: 'pointer',
          boxShadow: '0 6px 20px rgba(37, 99, 235, 0.45)',
          zIndex: 90,
          transition: 'transform 0.2s',
        }}
        onMouseEnter={(e) => (e.currentTarget.style.transform = 'scale(1.05)')}
        onMouseLeave={(e) => (e.currentTarget.style.transform = 'scale(1)')}
      >
        <Bot size={18} />
        <span>Ask Pick (AI)</span>
      </button>

      {/* Co-Pilot Modal */}
      <CustomerCopilotModal
        isOpen={isCopilotOpen}
        onClose={() => setIsCopilotOpen(false)}
      />
    </AppLayout>
  );
};
