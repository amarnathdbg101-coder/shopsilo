/**
 * ThemeLanguageBar Component
 * 
 * Hinglish Hint:
 * Top navbar, side drawer aur profile screen me ek-click me:
 * 1. Dark Mode aur Normal Light Mode switch karne ka button
 * 2. Hindi, English aur Hinglish bhasha badalne ke chips
 */

import React from 'react';
import { Sun, Moon, Globe } from 'lucide-react';
import { useTheme } from '../../context/ThemeContext';
import { useLanguage } from '../../context/LanguageContext';

export const ThemeLanguageBar = ({ compact = false }) => {
  const { theme, isDark, toggleTheme } = useTheme();
  const { language, setLanguage, t } = useLanguage();

  const langOptions = [
    { code: 'hi', label: 'हिंदी' },
    { code: 'en', label: 'English' },
    { code: 'hinglish', label: 'Hinglish' },
  ];

  if (compact) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        {/* Theme Toggle */}
        <button
          onClick={toggleTheme}
          style={{
            background: 'rgba(255, 255, 255, 0.15)',
            border: '1px solid rgba(255, 255, 255, 0.25)',
            borderRadius: 'var(--radius-full)',
            width: '34px',
            height: '34px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: 'pointer',
            color: '#ffffff',
            transition: 'all 0.2s ease',
          }}
          title={isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
        >
          {isDark ? <Sun size={17} color="#fbbf24" /> : <Moon size={17} />}
        </button>

        {/* Language Selector */}
        <div style={{ display: 'flex', background: 'rgba(255,255,255,0.12)', borderRadius: 'var(--radius-full)', padding: '2px' }}>
          {langOptions.map((opt) => (
            <button
              key={opt.code}
              onClick={() => setLanguage(opt.code)}
              style={{
                border: 'none',
                background: language === opt.code ? '#ffffff' : 'transparent',
                color: language === opt.code ? '#0f172a' : '#ffffff',
                fontWeight: 700,
                fontSize: '0.72rem',
                padding: '4px 8px',
                borderRadius: 'var(--radius-full)',
                cursor: 'pointer',
                transition: 'all 0.15s ease',
              }}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div
      style={{
        backgroundColor: 'var(--bg-card)',
        border: '1px solid var(--border-subtle)',
        borderRadius: '16px',
        padding: '14px',
        display: 'flex',
        flexDirection: 'column',
        gap: '12px',
      }}
    >
      {/* Theme Switcher Row */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          {isDark ? <Moon size={18} color="#818cf8" /> : <Sun size={18} color="#f59e0b" />}
          <div>
            <div style={{ fontWeight: 800, fontSize: '0.88rem', color: 'var(--text-primary)' }}>
              {t('theme')}: {isDark ? t('dark_mode') : t('light_mode')}
            </div>
            <div style={{ fontSize: '0.74rem', color: 'var(--text-secondary)' }}>
              {isDark ? 'Raat ke samay aakhon ke liye aaramdayak' : 'Standard clean daylight interface'}
            </div>
          </div>
        </div>

        <button
          onClick={toggleTheme}
          className="btn btn-secondary btn-sm"
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
            fontWeight: 700,
            borderRadius: 'var(--radius-full)',
            padding: '6px 14px',
          }}
        >
          {isDark ? <Sun size={15} color="#f59e0b" /> : <Moon size={15} />}
          <span>{isDark ? 'Normal Light' : 'Dark Mode'}</span>
        </button>
      </div>

      <div style={{ height: '1px', background: 'var(--border-subtle)' }} />

      {/* Language Switcher Row */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '8px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Globe size={18} color="var(--color-primary)" />
          <div>
            <div style={{ fontWeight: 800, fontSize: '0.88rem', color: 'var(--text-primary)' }}>
              {t('language')}
            </div>
            <div style={{ fontSize: '0.74rem', color: 'var(--text-secondary)' }}>
              Apni pasand ki bhasha chunein
            </div>
          </div>
        </div>

        <div style={{ display: 'flex', gap: '6px' }}>
          {langOptions.map((opt) => (
            <button
              key={opt.code}
              onClick={() => setLanguage(opt.code)}
              style={{
                border: language === opt.code ? '1.5px solid var(--color-primary)' : '1px solid var(--border-subtle)',
                background: language === opt.code ? 'var(--color-primary)' : 'var(--bg-surface-subtle)',
                color: language === opt.code ? '#ffffff' : 'var(--text-primary)',
                fontWeight: 800,
                fontSize: '0.78rem',
                padding: '6px 12px',
                borderRadius: 'var(--radius-full)',
                cursor: 'pointer',
                transition: 'all 0.15s ease',
              }}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
};
