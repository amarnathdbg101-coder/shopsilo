/**
 * Category Visual Metadata (Emojis & Themes)
 * 
 * Hinglish Hint:
 * Blinkit / Zepto jaisa rich look dene ke liye har category ka icon aur emoji:
 */

export const getCategoryEmoji = (slugOrName = '') => {
  const str = String(slugOrName || '').toLowerCase();
  if (str.includes('kirana') || str.includes('grocery')) return '🛒';
  if (str.includes('snack') || str.includes('drink') || str.includes('beverage')) return '🍪';
  if (str.includes('dairy') || str.includes('milk') || str.includes('egg') || str.includes('bread')) return '🥛';
  if (str.includes('care') || str.includes('beauty') || str.includes('soap')) return '🧼';
  if (str.includes('fruit') || str.includes('veg')) return '🍎';
  if (str.includes('electr') || str.includes('mobile')) return '📱';
  if (str.includes('cloth') || str.includes('fashion') || str.includes('wear')) return '👕';
  if (str.includes('pharm') || str.includes('med') || str.includes('health')) return '💊';
  if (str.includes('clean') || str.includes('house')) return '🧹';
  return '🏪';
};
