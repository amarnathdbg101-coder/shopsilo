/**
 * Smart Automatic SKU Generator
 * 
 * Hinglish Hint:
 * Product ke naam se automatic unique aur standard SKU code banata hai:
 * - "Tata Salt 1kg" -> "TS-1KG-849"
 * - "Amul Butter 100g" -> "AB-100G-312"
 * - "Parle-G Biscuit" -> "PB-942"
 */

export const generateSmartSKU = (productName = '') => {
  if (!productName || !productName.trim()) {
    return `PROD-${Math.random().toString(36).substring(2, 6).toUpperCase()}`;
  }

  const clean = productName.trim();

  // Words split
  const words = clean.split(/\s+/).filter(Boolean);

  // Initial letters (max 4)
  let prefix = words
    .map((w) => w.replace(/[^a-zA-Z0-9]/g, ''))
    .filter((w) => w.length > 0)
    .map((w) => w[0].toUpperCase())
    .slice(0, 4)
    .join('');

  if (!prefix) prefix = 'SKU';

  // Check if any word has weight/size like 1kg, 500g, 1L, 2L, xl, xxl
  const sizeMatch = clean.match(/(\d+\.?\d*\s*(?:kg|g|gm|l|ltr|ml|pack|pcs|pc))/i);
  let sizeTag = '';
  if (sizeMatch) {
    sizeTag = '-' + sizeMatch[1].replace(/\s+/g, '').toUpperCase();
  }

  // Random 3 character alphanumeric suffix for 100% uniqueness
  const randomSuffix = Math.random().toString(36).substring(2, 5).toUpperCase();

  return `${prefix}${sizeTag}-${randomSuffix}`;
};
