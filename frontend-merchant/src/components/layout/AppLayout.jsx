/**
 * Main App Layout Shell
 * 
 * Hinglish Hint:
 * Har screen ko ek structured mobile app container me wrap karta hai:
 * Header + Content + Bottom Tab Navigation
 */

import React from 'react';
import { AppHeader } from './AppHeader';
import { BottomNav } from './BottomNav';

export const AppLayout = ({
  children,
  title,
  subtitle,
  showBack = false,
  hideNav = false,
}) => {
  return (
    <div className="app-container">
      <AppHeader title={title} subtitle={subtitle} showBack={showBack} />
      <main className={`app-content ${hideNav ? 'no-bottom-nav' : ''}`}>
        {children}
      </main>
      {!hideNav && <BottomNav />}
    </div>
  );
};
