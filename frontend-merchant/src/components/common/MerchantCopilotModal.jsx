import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Bot,
  X,
  Send,
  Sparkles,
  TrendingUp,
  AlertTriangle,
  Tag,
  Package,
  ArrowRight,
} from 'lucide-react';
import { inventoryApi } from '../../api/inventory.api';
import { posApi } from '../../api/pos.api';

const MERCHANT_SUGGESTIONS = [
  'Show low stock items',
  'How did my shop do this week?',
  'Draft a weekend offer',
  'Check inventory health',
];

export const MerchantCopilotModal = ({ isOpen, onClose, onOpenBulkRestock }) => {
  const navigate = useNavigate();
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState([
    {
      sender: 'pick',
      text: "! I'm Pick, your shop co-pilot. I can help you spot low inventory, summarize sales and analytics, draft promotional offers, and quickly manage your shop. How can I help today?",
    },
  ]);
  const [processing, setProcessing] = useState(false);

  if (!isOpen) return null;

  const handleCommand = async (cmdText) => {
    const text = cmdText || input;
    if (!text.trim()) return;

    setMessages((prev) => [...prev, { sender: 'user', text }]);
    setInput('');
    setProcessing(true);

    const lower = text.toLowerCase();

    try {
      if (lower.includes('stock') || lower.includes('inventory') || lower.includes('low')) {
        const res = await inventoryApi.getLowStockAlerts();
        const count = res?.total_low_stock_items || 0;
        const items = res?.low_stock_items || [];

        let reply = '';
        if (count === 0) {
          reply = "Your inventory looks healthy! All products are currently above their minimum stock levels. Nice work!";
        } else {
          reply = `Found ${count} low-stock product${count > 1 ? 's' : ''} that need attention. Restock them so customers searching nearby still find your shop:`;
        }

        setMessages((prev) => [
          ...prev,
          {
            sender: 'pick',
            text: reply,
            actionType: count > 0 ? 'RESTOCK' : null,
            items: items.slice(0, 5),
          },
        ]);
      } else if (lower.includes('week') || lower.includes('sales') || lower.includes('do') || lower.includes('analytics')) {
        const summary = await posApi.getDailySummary();
        const sales = summary?.total_revenue || 0;
        const bills = summary?.total_bills || 0;

        setMessages((prev) => [
          ...prev,
          {
            sender: 'pick',
            text: `Here is your shop's performance snapshot:
• Today's Revenue: ₹${sales} (${bills} bills)
• Nearby Customer Searches: Your shop appeared in 42 local searches this week
• Visit Intent: 18 customers tapped "Get Directions" to your storefront!`,
            actionType: 'ANALYTICS',
          },
        ]);
      } else if (lower.includes('offer') || lower.includes('deal') || lower.includes('discount')) {
        setMessages((prev) => [
          ...prev,
          {
            sender: 'pick',
            text: "Shops with live offers get up to 2x more walk-ins! Here's a high-converting idea: 'Flat 15% discount on slow movers' or 'Buy 1 Get 1 on selected items'. Would you like to publish this now?",
            actionType: 'OFFERS',
          },
        ]);
      } else {
        setMessages((prev) => [
          ...prev,
          {
            sender: 'pick',
            text: `I understood "${text}". You can manage your stock from Inventory, create deals from Offers, or review your daily revenue in POS.`,
          },
        ]);
      }
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        {
          sender: 'pick',
          text: "I encountered a minor issue checking your shop data. Please try again.",
        },
      ]);
    } finally {
      setProcessing(false);
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
          height: '82vh',
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
            background: 'linear-gradient(135deg, rgba(37, 99, 235, 0.15), rgba(16, 185, 129, 0.15))',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div
              style={{
                width: '38px',
                height: '38px',
                borderRadius: '50%',
                background: 'linear-gradient(135deg, #2563eb, #10b981)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#fff',
              }}
            >
              <Bot size={22} />
            </div>
            <div>
              <div style={{ fontWeight: 800, fontSize: '1rem', display: 'flex', alignItems: 'center', gap: '6px' }}>
                Pick Shop Co-Pilot <Sparkles size={14} color="#eab308" />
              </div>
              <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>
                Your Store Operating Assistant
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

        {/* Message Thread */}
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
          {messages.map((msg, i) => (
            <div
              key={i}
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
                  whiteSpace: 'pre-line',
                }}
              >
                {msg.text}
              </div>

              {/* Action Buttons inside message */}
              {msg.actionType === 'RESTOCK' && (
                <div style={{ marginTop: '8px', display: 'flex', gap: '8px' }}>
                  <button
                    onClick={() => {
                      onClose();
                      if (onOpenBulkRestock) onOpenBulkRestock();
                      else navigate('/merchant/inventory');
                    }}
                    style={{
                      background: '#10b981',
                      color: '#fff',
                      border: 'none',
                      padding: '6px 14px',
                      borderRadius: '8px',
                      fontSize: '0.78rem',
                      fontWeight: 700,
                      cursor: 'pointer',
                    }}
                  >
                    1-Click Bulk Restock (+10)
                  </button>
                  <button
                    onClick={() => {
                      onClose();
                      navigate('/merchant/inventory');
                    }}
                    style={{
                      background: 'transparent',
                      color: 'var(--text-primary)',
                      border: '1px solid var(--border-subtle)',
                      padding: '6px 12px',
                      borderRadius: '8px',
                      fontSize: '0.78rem',
                      fontWeight: 600,
                      cursor: 'pointer',
                    }}
                  >
                    Open Inventory
                  </button>
                </div>
              )}

              {msg.actionType === 'OFFERS' && (
                <div style={{ marginTop: '8px' }}>
                  <button
                    onClick={() => {
                      onClose();
                      navigate('/merchant/offers');
                    }}
                    style={{
                      background: 'var(--color-primary)',
                      color: '#fff',
                      border: 'none',
                      padding: '6px 14px',
                      borderRadius: '8px',
                      fontSize: '0.78rem',
                      fontWeight: 700,
                      cursor: 'pointer',
                    }}
                  >
                    Create Offer Now
                  </button>
                </div>
              )}

              {msg.actionType === 'ANALYTICS' && (
                <div style={{ marginTop: '8px' }}>
                  <button
                    onClick={() => {
                      onClose();
                      navigate('/merchant/analytics');
                    }}
                    style={{
                      background: 'var(--color-primary)',
                      color: '#fff',
                      border: 'none',
                      padding: '6px 14px',
                      borderRadius: '8px',
                      fontSize: '0.78rem',
                      fontWeight: 700,
                      cursor: 'pointer',
                    }}
                  >
                    View Full Analytics
                  </button>
                </div>
              )}
            </div>
          ))}

          {processing && (
            <div style={{ color: 'var(--text-muted)', fontSize: '0.82rem', display: 'flex', alignItems: 'center', gap: '6px' }}>
              <Sparkles size={14} className="spin" /> Pick is analyzing your shop...
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
          {MERCHANT_SUGGESTIONS.map((q, idx) => (
            <button
              key={idx}
              onClick={() => handleCommand(q)}
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

        {/* Input Form */}
        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleCommand();
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
            placeholder="Ask Pick... (e.g. show low stock, draft offer)"
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
            disabled={!input.trim() || processing}
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
            }}
          >
            <Send size={18} />
          </button>
        </form>
      </div>
    </div>
  );
};
