import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Bot,
  X,
  Send,
  Sparkles,
  Search,
  MapPin,
  Store,
  Navigation,
  CheckCircle2,
  AlertCircle,
  ExternalLink,
} from 'lucide-react';
import { productApi } from '../../api/product.api';
import { shopApi } from '../../api/shop.api';
import { useLocation } from '../../context/LocationContext';
import { calculateDistanceKm, formatDistance } from '../../utils/distance';

const SUGGESTED_QUERIES = [
  'Cheapest headphones',
  'Type C Cable available nearby',
  'Pharmacy open near me',
  'Compare prices for Paracetamol',
  'In stock groceries',
];

export const CustomerCopilotModal = ({ isOpen, onClose }) => {
  const navigate = useNavigate();
  const { coords } = useLocation();

  const [input, setInput] = useState('');
  const [messages, setMessages] = useState([
    {
      sender: 'pick',
      text: "Hi! I'm Pick, your shopping co-pilot. I search products across nearby physical shops, compare prices, check stock and opening status, and open directions. What are you looking for today?",
    },
  ]);
  const [searching, setSearching] = useState(false);

  if (!isOpen) return null;

  const handleQuery = async (queryText) => {
    const text = queryText || input;
    if (!text.trim()) return;

    // Add user message
    const userMsg = { sender: 'user', text };
    setMessages((prev) => [...prev, userMsg]);
    setInput('');
    setSearching(true);

    try {
      // Search products & shops
      const [prodRes, shopRes] = await Promise.allSettled([
        productApi.listPublicProducts({ q: text, limit: 6 }),
        shopApi.listPublicShops({ q: text, limit: 4 }),
      ]);

      const products = prodRes.status === 'fulfilled' ? prodRes.value?.products || [] : [];
      const shops = shopRes.status === 'fulfilled' ? shopRes.value?.shops || shopRes.value || [] : [];

      let responseText = '';
      if (products.length === 0 && shops.length === 0) {
        responseText = `I couldn't find any items or shops matching "${text}" near you right now. Try searching for broader terms like "mobile", "medicine", or "grocery".`;
      } else if (products.length > 0) {
        responseText = `Found ${products.length} matching product${products.length > 1 ? 's' : ''} in stock at nearby local shops:`;
      } else {
        responseText = `Found ${shops.length} shop${shops.length > 1 ? 's' : ''} matching "${text}" near you:`;
      }

      setMessages((prev) => [
        ...prev,
        {
          sender: 'pick',
          text: responseText,
          products,
          shops: products.length === 0 ? shops : [],
        },
      ]);
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        {
          sender: 'pick',
          text: "Sorry, I had trouble searching nearby shops right now. Please try again in a moment.",
        },
      ]);
    } finally {
      setSearching(false);
    }
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.7)',
        backdropFilter: 'blur(6px)',
        zIndex: 1000,
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'flex-end',
      }}
    >
      <div
        style={{
          background: 'var(--bg-card)',
          width: '100%',
          maxWidth: '520px',
          margin: '0 auto',
          height: '85vh',
          borderTopLeftRadius: '24px',
          borderTopRightRadius: '24px',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          boxShadow: '0 -8px 30px rgba(0,0,0,0.5)',
          border: '1px solid var(--border-subtle)',
        }}
      >
        {/* Header */}
        <div
          style={{
            padding: '16px 20px',
            borderBottom: '1px solid var(--border-subtle)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            background: 'linear-gradient(135deg, rgba(37, 99, 235, 0.1), rgba(168, 85, 247, 0.1))',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div
              style={{
                width: '38px',
                height: '38px',
                borderRadius: '50%',
                background: 'linear-gradient(135deg, #2563eb, #7c3aed)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#fff',
                boxShadow: '0 2px 10px rgba(37, 99, 235, 0.4)',
              }}
            >
              <Bot size={22} />
            </div>
            <div>
              <div style={{ fontWeight: 800, fontSize: '1rem', display: 'flex', alignItems: 'center', gap: '6px' }}>
                Pick Co-Pilot <Sparkles size={14} color="#eab308" />
              </div>
              <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>
                Your Local Shopping Assistant
              </div>
            </div>
          </div>

          <button
            onClick={onClose}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--text-muted)',
              cursor: 'pointer',
              padding: '6px',
            }}
          >
            <X size={20} />
          </button>
        </div>

        {/* Messages Body */}
        <div
          style={{
            flex: 1,
            overflowY: 'auto',
            padding: '16px',
            display: 'flex',
            flexDirection: 'column',
            gap: '14px',
          }}
        >
          {messages.map((msg, index) => (
            <div
              key={index}
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: msg.sender === 'user' ? 'flex-end' : 'flex-start',
              }}
            >
              <div
                style={{
                  maxWidth: '85%',
                  padding: '12px 16px',
                  borderRadius: msg.sender === 'user' ? '18px 18px 4px 18px' : '18px 18px 18px 4px',
                  background: msg.sender === 'user' ? 'var(--color-primary)' : 'rgba(255,255,255,0.06)',
                  color: msg.sender === 'user' ? '#fff' : 'var(--text-primary)',
                  fontSize: '0.88rem',
                  lineHeight: 1.45,
                  border: msg.sender === 'user' ? 'none' : '1px solid var(--border-subtle)',
                }}
              >
                {msg.text}
              </div>

              {/* Product Cards Result */}
              {msg.products && msg.products.length > 0 && (
                <div style={{ width: '100%', marginTop: '10px', display: 'flex', flexDirection: 'column', gap: '10px' }}>
                  {msg.products.map((p) => {
                    const distKm = calculateDistanceKm(
                      coords?.lat,
                      coords?.lng,
                      p.shop_latitude,
                      p.shop_longitude
                    );

                    return (
                      <div
                        key={p.id}
                        className="card"
                        style={{
                          margin: 0,
                          padding: '12px',
                          borderRadius: '12px',
                          border: '1px solid var(--border-subtle)',
                        }}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                          <div>
                            <div style={{ fontWeight: 700, fontSize: '0.9rem' }}>{p.name}</div>
                            <div style={{ display: 'flex', alignItems: 'baseline', gap: '6px', marginTop: '3px' }}>
                              <span style={{ fontWeight: 800, color: 'var(--color-primary)', fontSize: '0.95rem' }}>
                                ₹{p.price}
                              </span>
                              {p.mrp && p.mrp > p.price && (
                                <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)', textDecoration: 'line-through' }}>
                                  ₹{p.mrp}
                                </span>
                              )}
                              {p.mrp && p.mrp > p.price && (
                                <span style={{ fontSize: '0.7rem', color: '#10b981', fontWeight: 600 }}>
                                  Below MRP
                                </span>
                              )}
                            </div>
                            <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '4px', display: 'flex', alignItems: 'center', gap: '4px' }}>
                              <Store size={12} /> {p.shop_name || p.shop_slug || 'Local Shop'}
                              {distKm != null && <span>• {formatDistance(distKm)}</span>}
                            </div>
                          </div>

                          {(() => {
                            const inStock = Number(p.stock_quantity ?? p.inventory?.available_quantity ?? p.inventory?.quantity ?? 0) > 0;
                            return (
                              <span
                                style={{
                                  fontSize: '0.72rem',
                                  padding: '3px 8px',
                                  borderRadius: '6px',
                                  background: inStock ? 'rgba(16, 185, 129, 0.15)' : 'rgba(239, 68, 68, 0.15)',
                                  color: inStock ? '#10b981' : '#ef4444',
                                  fontWeight: 600,
                                }}
                              >
                                {inStock ? 'In Stock' : 'Out of Stock'}
                              </span>
                            );
                          })()}
                        </div>

                        {/* Actions */}
                        <div style={{ display: 'flex', gap: '8px', marginTop: '10px' }}>
                          {p.shop_latitude && p.shop_longitude && (
                            <a
                              href={`https://maps.google.com/?q=${p.shop_latitude},${p.shop_longitude}`}
                              target="_blank"
                              rel="noreferrer"
                              style={{
                                flex: 1,
                                background: 'rgba(59, 130, 246, 0.1)',
                                color: '#3b82f6',
                                padding: '6px',
                                borderRadius: '8px',
                                fontSize: '0.75rem',
                                fontWeight: 600,
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                gap: '4px',
                                textDecoration: 'none',
                              }}
                            >
                              <Navigation size={12} /> Directions
                            </a>
                          )}
                          {p.shop_slug && (
                            <button
                              onClick={() => {
                                onClose();
                                navigate(`/shop/${p.shop_slug}`);
                              }}
                              style={{
                                flex: 1,
                                background: 'var(--color-primary)',
                                color: '#fff',
                                border: 'none',
                                padding: '6px',
                                borderRadius: '8px',
                                fontSize: '0.75rem',
                                fontWeight: 600,
                                cursor: 'pointer',
                              }}
                            >
                              View Shop
                            </button>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          ))}

          {searching && (
            <div style={{ color: 'var(--text-muted)', fontSize: '0.82rem', display: 'flex', alignItems: 'center', gap: '6px' }}>
              <Sparkles size={14} className="spin" /> Searching nearby physical inventory...
            </div>
          )}
        </div>

        {/* Suggestion Chips */}
        <div
          style={{
            display: 'flex',
            gap: '8px',
            overflowX: 'auto',
            padding: '8px 16px',
            borderTop: '1px solid var(--border-subtle)',
            background: 'var(--bg-app)',
            scrollbarWidth: 'none',
          }}
        >
          {SUGGESTED_QUERIES.map((q, idx) => (
            <button
              key={idx}
              onClick={() => handleQuery(q)}
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border-subtle)',
                color: 'var(--text-secondary)',
                fontSize: '0.75rem',
                fontWeight: 600,
                padding: '5px 12px',
                borderRadius: '16px',
                whiteSpace: 'nowrap',
                cursor: 'pointer',
              }}
            >
              {q}
            </button>
          ))}
        </div>

        {/* Input Bar */}
        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleQuery();
          }}
          style={{
            padding: '12px 16px',
            borderTop: '1px solid var(--border-subtle)',
            display: 'flex',
            gap: '10px',
            background: 'var(--bg-card)',
          }}
        >
          <input
            type="text"
            placeholder="Ask Pick... (e.g. cheapest headphones, nearby pharmacy)"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            style={{
              flex: 1,
              background: 'var(--bg-app)',
              border: '1px solid var(--border-subtle)',
              borderRadius: '24px',
              padding: '10px 16px',
              fontSize: '0.88rem',
              color: 'var(--text-primary)',
              outline: 'none',
            }}
          />
          <button
            type="submit"
            disabled={!input.trim() || searching}
            style={{
              width: '42px',
              height: '42px',
              borderRadius: '50%',
              background: 'var(--color-primary)',
              color: '#fff',
              border: 'none',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: input.trim() ? 'pointer' : 'not-allowed',
              opacity: input.trim() ? 1 : 0.6,
            }}
          >
            <Send size={18} />
          </button>
        </form>
      </div>
    </div>
  );
};
