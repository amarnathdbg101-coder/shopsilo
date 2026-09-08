/**
 * Inventory & Stock Management Screen (Enhanced Super-App Version)
 * Inspired by: Blinkit, Zepto, Shopify
 * 
 * Features:
 * - Flexible MongoDB-like Product Attributes (Company, Model, Size, Type, Color, Season, Age Group, Gender & Custom Key-Values)
 * - Visual Category Selector (Emojis ke sath Blinkit jaisa)
 * - Automatic Smart SKU Code Generator (Product name se apne aap unique code banta hai)
 * - Stock Adjust & Low-stock Alerts
 * - 1-Click Wholesale Reorder PDF Sheet
 */

import React, { useState, useEffect } from 'react';
import {
  Package,
  AlertTriangle,
  FileDown,
  Plus,
  Search,
  ArrowUpDown,
  CheckCircle,
  RefreshCw,
  Boxes,
  Check,
  Sliders,
  X,
  Tag,
  Upload,
  Image,
  Edit3,
  Trash2,
} from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { productApi } from '../../api/product.api';
import { inventoryApi } from '../../api/inventory.api';
import { uploadApi } from '../../api/upload.api';
import { generateSmartSKU } from '../../utils/sku';
import { getCategoryEmoji } from '../../utils/categoryMeta';
import { AppLayout } from '../../components/layout/AppLayout';
import { ProductDetailModal } from '../../components/common/ProductDetailModal';
import { getImageUrl } from '../../utils/imageUrl';

export const InventoryScreen = () => {
  const { shop } = useAuth();

  const [activeTab, setActiveTab] = useState('catalog'); // 'catalog' | 'low_stock'
  const [products, setProducts] = useState([]);
  const [lowStockItems, setLowStockItems] = useState([]);
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');

  // Selected product for full detail view modal
  const [inspectedProduct, setInspectedProduct] = useState(null);

  // New product images
  const [productImages, setProductImages] = useState([]);
  const [productImagePreviews, setProductImagePreviews] = useState([]);

  // Stock Adjust Modal
  const [selectedProduct, setSelectedProduct] = useState(null);
  const [adjustmentQty, setAdjustmentQty] = useState('');
  const [adjustNotes, setAdjustNotes] = useState('');
  const [adjustLoading, setAdjustLoading] = useState(false);

  // New Product Modal State
  const [showAddProductModal, setShowAddProductModal] = useState(false);
  const [newProductForm, setNewProductForm] = useState({
    name: '',
    sku: '',
    price: '',
    cost_price: '',
    stock_quantity: '20',
    category_id: '',
    // Flexible attributes (MongoDB inside PostgreSQL)
    attributes: {
      company: '',
      model: '',
      product_type: '',
      size: '',
      color: '',
      gender: '',
      season: '',
      age_group: '',
    },
  });

  // Custom key-value pairs for endless flexibility
  const [customAttributes, setCustomAttributes] = useState([]);
  const [showAdvancedAttributes, setShowAdvancedAttributes] = useState(true);

  const [addProductLoading, setAddProductLoading] = useState(false);
  const [error, setError] = useState('');

  // Edit Product Modal State
  const [editingProduct, setEditingProduct] = useState(null);
  const [editProductForm, setEditProductForm] = useState(null);
  const [editExistingImages, setEditExistingImages] = useState([]);
  const [editNewImages, setEditNewImages] = useState([]);
  const [editNewImagePreviews, setEditNewImagePreviews] = useState([]);
  const [editProductLoading, setEditProductLoading] = useState(false);
  const [editError, setEditError] = useState('');

  // Universal Safe Stock Helper
  const getProductStock = (p) => {
    if (!p) return 0;
    return Number(
      p.stock_quantity ??
      p.inventory?.available_quantity ??
      p.inventory?.quantity ??
      0
    );
  };

  // Handle new product images selection (Add Product Modal)
  const handleProductImageSelect = (e) => {
    const files = Array.from(e.target.files || []);
    if (files.length === 0) return;

    const validFiles = [];
    const validPreviews = [];

    for (const f of files) {
      if (productImages.length + validFiles.length >= 4) {
        alert('Ek product ke liye maximum 4 photos hi jod sakte hain');
        break;
      }
      if (f.size > 2 * 1024 * 1024) {
        alert(`"${f.name}" ka size 2MB se zyada hai. Kripya 2MB se choti photo chunein.`);
        continue;
      }
      const mime = (f.type || '').toLowerCase();
      if (mime && !['image/jpeg', 'image/png', 'image/webp', 'image/jpg'].includes(mime)) {
        alert(`"${f.name}" ka format valid nahi hai. Sirf JPG, PNG, ya WebP allowed hain.`);
        continue;
      }
      validFiles.push(f);
      validPreviews.push(URL.createObjectURL(f));
    }

    setProductImages((prev) => [...prev, ...validFiles]);
    setProductImagePreviews((prev) => [...prev, ...validPreviews]);
    e.target.value = '';
  };

  // Remove photo from Add Product modal
  const handleRemoveProductImage = (idx) => {
    setProductImages((prev) => prev.filter((_, i) => i !== idx));
    setProductImagePreviews((prev) => {
      const removed = prev[idx];
      if (removed && removed.startsWith('blob:')) {
        URL.revokeObjectURL(removed);
      }
      return prev.filter((_, i) => i !== idx);
    });
  };

  const handleOpenEditProduct = (p) => {
    setEditingProduct(p);
    setEditExistingImages(Array.isArray(p.images) ? [...p.images] : []);
    setEditNewImages([]);
    setEditNewImagePreviews([]);
    setEditError('');
    setEditProductForm({
      name: p.name || '',
      sku: p.sku || '',
      price: String(p.price || ''),
      cost_price: String(p.cost_price || ''),
      stock_quantity: String(getProductStock(p)),
      category_id: p.category_id || p.category?.id || (categories[0]?.id || ''),
      description: p.description || '',
      attributes: { ...(p.attributes || {}) },
    });
  };

  const handleEditNewImagesSelect = (e) => {
    const files = Array.from(e.target.files || []);
    if (files.length === 0) return;
    const currentTotal = editExistingImages.length + editNewImages.length;
    const remaining = 4 - currentTotal;
    if (remaining <= 0) {
      alert('Max 4 photos allowed per product');
      return;
    }

    const validFiles = [];
    const validPreviews = [];

    for (const f of files.slice(0, remaining)) {
      if (f.size > 2 * 1024 * 1024) {
        alert(`"${f.name}" ka size 2MB se zyada hai.`);
        continue;
      }
      const mime = (f.type || '').toLowerCase();
      if (mime && !['image/jpeg', 'image/png', 'image/webp', 'image/jpg'].includes(mime)) {
        alert(`"${f.name}" ka format valid nahi hai.`);
        continue;
      }
      validFiles.push(f);
      validPreviews.push(URL.createObjectURL(f));
    }

    setEditNewImages((prev) => [...prev, ...validFiles]);
    setEditNewImagePreviews((prev) => [...prev, ...validPreviews]);
    e.target.value = '';
  };

  const handleRemoveEditExistingImage = (idx) => {
    setEditExistingImages((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleRemoveEditNewImage = (idx) => {
    setEditNewImages((prev) => prev.filter((_, i) => i !== idx));
    setEditNewImagePreviews((prev) => {
      const removed = prev[idx];
      if (removed && removed.startsWith('blob:')) {
        URL.revokeObjectURL(removed);
      }
      return prev.filter((_, i) => i !== idx);
    });
  };

  const handleSaveEditProduct = async (e) => {
    e.preventDefault();
    if (!editingProduct || !editProductForm) return;

    try {
      setEditProductLoading(true);
      setEditError('');

      let uploadedNewUrls = [];
      if (editNewImages.length > 0) {
        try {
          const uploadRes = await uploadApi.uploadProductImages(editNewImages);
          uploadedNewUrls = uploadRes.images || [];
        } catch (uploadErr) {
          console.error('Edit image upload error:', uploadErr);
          if (!window.confirm('Nayi photo upload nahi ho saki: ' + (uploadErr.message || 'Error') + '. Kya aap baaki badlaav save karna chahte hain?')) {
            setEditProductLoading(false);
            return;
          }
        }
      }

      const finalImages = [...editExistingImages, ...uploadedNewUrls].slice(0, 4);

      const updatePayload = {
        name: editProductForm.name.trim(),
        sku: editProductForm.sku.trim(),
        price: Number(editProductForm.price),
        cost_price: Number(editProductForm.cost_price) || 0,
        stock_quantity: Number(editProductForm.stock_quantity) || 0,
        category_id: editProductForm.category_id,
        description: editProductForm.description?.trim(),
        attributes: editProductForm.attributes,
        images: finalImages,
      };

      await productApi.updateProduct(editingProduct.id, updatePayload);
      setEditingProduct(null);
      setEditProductForm(null);
      await loadData();
    } catch (err) {
      setEditError(err.message || 'Product update nahi ho saka');
    } finally {
      setEditProductLoading(false);
    }
  };

  const handleDeleteProduct = async (id) => {
    if (!window.confirm('Kya aap sach me iss product ko catalog se delete karna chahte hain?')) {
      return;
    }
    try {
      await productApi.deleteProduct(id);
      setEditingProduct(null);
      setInspectedProduct(null);
      await loadData();
    } catch (err) {
      alert('Delete asafal: ' + err.message);
    }
  };

  const loadData = async () => {
    try {
      setLoading(true);
      const [prodRes, lowRes, catRes] = await Promise.allSettled([
        shop?.slug ? productApi.listByShopSlug(shop.slug) : Promise.resolve([]),
        inventoryApi.getLowStockAlerts(),
        productApi.getCategories(),
      ]);

      if (prodRes.status === 'fulfilled') {
        setProducts(prodRes.value?.products || prodRes.value || []);
      }
      if (lowRes.status === 'fulfilled') {
        setLowStockItems(lowRes.value?.items || []);
      }
      if (catRes.status === 'fulfilled') {
        const catList = catRes.value || [];
        setCategories(catList);
        if (catList.length > 0 && !newProductForm.category_id) {
          setNewProductForm((prev) => ({ ...prev, category_id: catList[0].id }));
        }
      }
    } catch (err) {
      console.error('Inventory load error:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [shop]);

  // Product naam type karte hi automatic smart SKU banayein
  const handleProductNameChange = (e) => {
    const name = e.target.value;
    setNewProductForm((prev) => ({
      ...prev,
      name,
      sku: generateSmartSKU(name),
    }));
  };

  // Re-generate SKU
  const handleRegenerateSKU = () => {
    setNewProductForm((prev) => ({
      ...prev,
      sku: generateSmartSKU(prev.name),
    }));
  };

  // Attribute change handler
  const handleAttributeChange = (key, val) => {
    setNewProductForm((prev) => ({
      ...prev,
      attributes: {
        ...prev.attributes,
        [key]: val,
      },
    }));
  };

  // Custom key-value pair functions
  const addCustomAttribute = () => {
    setCustomAttributes((prev) => [...prev, { key: '', value: '' }]);
  };

  const updateCustomAttribute = (index, field, val) => {
    setCustomAttributes((prev) => {
      const updated = [...prev];
      updated[index][field] = val;
      return updated;
    });
  };

  const removeCustomAttribute = (index) => {
    setCustomAttributes((prev) => prev.filter((_, i) => i !== index));
  };

  // Stock Adjust karna
  const handleStockAdjust = async (e) => {
    e.preventDefault();
    if (!selectedProduct) return;
    try {
      setAdjustLoading(true);
      await inventoryApi.adjustStock({
        product_id: selectedProduct.id,
        adjustment: Number(adjustmentQty),
        notes: adjustNotes || 'Manual counter stock adjustment',
      });
      setSelectedProduct(null);
      setAdjustmentQty('');
      setAdjustNotes('');
      await loadData();
    } catch (err) {
      alert('Stock update asafal: ' + err.message);
    } finally {
      setAdjustLoading(false);
    }
  };

  const [bulkRestocking, setBulkRestocking] = useState(false);

  // Bulk Restock Helper: adds +qty units to all low-stock or out-of-stock products
  const handleBulkRestock = async (qty = 10) => {
    const targets = (Array.isArray(products) ? products : []).filter(
      (p) => getProductStock(p) <= 5
    );
    if (targets.length === 0) {
      alert('All products are above minimum stock levels. Your inventory looks healthy!');
      return;
    }
    if (!window.confirm(`Bulk Restock: ${targets.length} low-stock/out-of-stock products me +${qty} units add karein?`)) {
      return;
    }

    setBulkRestocking(true);
    try {
      await Promise.all(
        targets.map((p) =>
          inventoryApi.adjustStock({
            product_id: p.id,
            adjustment: qty,
            notes: `Bulk restock (+${qty})`,
          })
        )
      );
      await loadData();
      alert(`Success! ${targets.length} products restocked with +${qty} units.`);
    } catch (err) {
      alert('Bulk restock encountered an error: ' + err.message);
    } finally {
      setBulkRestocking(false);
    }
  };

  const totalStockValue = (Array.isArray(products) ? products : []).reduce(
    (acc, p) => acc + (Number(p.price) || 0) * getProductStock(p),
    0
  );


  // Naya Product Create karna
  const handleAddProduct = async (e) => {
    e.preventDefault();
    if (!newProductForm.category_id) {
      setError('Kripya ek category chuniye');
      return;
    }

    try {
      setAddProductLoading(true);
      setError('');

      // Build final attributes map (clean empty keys)
      const finalAttributes = {};
      Object.entries(newProductForm.attributes).forEach(([k, v]) => {
        if (v && v.trim()) {
          finalAttributes[k] = v.trim();
        }
      });
      customAttributes.forEach((item) => {
        if (item.key && item.key.trim() && item.value && item.value.trim()) {
          finalAttributes[item.key.trim().toLowerCase()] = item.value.trim();
        }
      });

      // Upload product images if selected
      let uploadedImageUrls = [];
      if (productImages.length > 0) {
        try {
          const uploadRes = await uploadApi.uploadProductImages(productImages);
          uploadedImageUrls = uploadRes.images || [];
        } catch (uploadErr) {
          console.error('Product image upload error:', uploadErr);
          if (!window.confirm('Photo upload me samasya aayi: ' + (uploadErr.message || 'Error') + '. Kya aap bina photo ke product save karna chahte hain?')) {
            setAddProductLoading(false);
            return;
          }
        }
      }

      await productApi.createProduct({
        ...newProductForm,
        sku: newProductForm.sku || generateSmartSKU(newProductForm.name),
        price: Number(newProductForm.price),
        cost_price: Number(newProductForm.cost_price) || 0,
        stock_quantity: Number(newProductForm.stock_quantity) || 0,
        category_id: newProductForm.category_id,
        attributes: finalAttributes,
        images: uploadedImageUrls,
      });

      setShowAddProductModal(false);
      setProductImages([]);
      setProductImagePreviews([]);
      setNewProductForm({
        name: '',
        sku: '',
        price: '',
        cost_price: '',
        stock_quantity: '20',
        category_id: categories[0]?.id || '',
        attributes: {
          company: '',
          model: '',
          product_type: '',
          size: '',
          color: '',
          gender: '',
          season: '',
          age_group: '',
        },
      });
      setCustomAttributes([]);
      await loadData();
    } catch (err) {
      setError(err.message || 'Product add nahi ho saka');
    } finally {
      setAddProductLoading(false);
    }
  };

  const filteredProducts = products.filter((p) => {
    const s = searchTerm.toLowerCase();
    const matchesBasic =
      p.name.toLowerCase().includes(s) ||
      p.sku?.toLowerCase().includes(s);

    // Search inside flexible JSONB attributes too (Brand, Model, Color, Size, Type)
    const matchesAttr =
      p.attributes &&
      Object.values(p.attributes).some(
        (val) => typeof val === 'string' && val.toLowerCase().includes(s)
      );

    return matchesBasic || matchesAttr;
  });

  return (
    <AppLayout title="Stock & Catalog" subtitle="Smart Inventory Control">
      {/* Tab Switcher */}
      <div className="tab-pills">
        <button
          className={`tab-pill ${activeTab === 'catalog' ? 'active' : ''}`}
          onClick={() => setActiveTab('catalog')}
        >
          <Boxes size={16} style={{ display: 'inline', verticalAlign: 'middle', marginRight: '4px' }} />
          Sare Products ({products.length})
        </button>
        <button
          className={`tab-pill ${activeTab === 'low_stock' ? 'active' : ''}`}
          onClick={() => setActiveTab('low_stock')}
          style={{ color: lowStockItems.length > 0 ? 'var(--color-danger)' : undefined }}
        >
          <AlertTriangle size={16} style={{ display: 'inline', verticalAlign: 'middle', marginRight: '4px' }} />
          Low Stock ({lowStockItems.length})
        </button>
      </div>

      {activeTab === 'catalog' ? (
        <>
          {/* Inventory Health & Stock Value Bar */}
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              background: 'var(--bg-card)',
              padding: '10px 14px',
              borderRadius: '12px',
              marginBottom: '12px',
              border: '1px solid var(--border-subtle)',
            }}
          >
            <div>
              <div style={{ fontSize: '0.72rem', color: 'var(--text-muted)' }}>Stock Retail Value</div>
              <div style={{ fontSize: '1.05rem', fontWeight: 800, color: 'var(--color-primary)' }}>
                ₹{totalStockValue.toLocaleString('en-IN')}
              </div>
            </div>

            <button
              onClick={() => handleBulkRestock(10)}
              disabled={bulkRestocking}
              style={{
                background: '#10b981',
                color: '#fff',
                border: 'none',
                padding: '6px 14px',
                borderRadius: '8px',
                fontSize: '0.78rem',
                fontWeight: 700,
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
              }}
            >
              <RefreshCw size={13} className={bulkRestocking ? 'spin' : ''} />
              {bulkRestocking ? 'Restocking...' : 'Bulk Restock (+10)'}
            </button>
          </div>

          {/* Top Actions: Search + Naya Product */}
          <div style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
            <div className="search-box" style={{ flex: 1, margin: 0 }}>
              <Search size={18} />
              <input
                type="text"
                placeholder="Product, SKU, Brand ya Size khojein..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
            <button
              onClick={() => {
                setShowAddProductModal(true);
                if (categories.length > 0 && !newProductForm.category_id) {
                  setNewProductForm((p) => ({ ...p, category_id: categories[0].id }));
                }
              }}
              className="btn btn-primary btn-sm"
              style={{ flexShrink: 0, gap: '4px' }}
            >
              <Plus size={16} /> Naya Saman
            </button>
          </div>

          {/* Product Items */}
          <div className="card" style={{ padding: '8px 12px' }}>
            {filteredProducts.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '28px', color: 'var(--text-muted)' }}>
                <Package size={36} style={{ margin: '0 auto 8px auto', opacity: 0.6 }} />
                <div style={{ fontWeight: 700 }}>Koi product nahi mila</div>
                <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                  Upar "+ Naya Saman" button dabakar product jodein.
                </p>
              </div>
            ) : (
              filteredProducts.map((p) => (
                <div
                  key={p.id}
                  className="list-item card-clickable"
                  onClick={() => setInspectedProduct(p)}
                  style={{ padding: '12px 0', alignItems: 'center', display: 'flex', gap: '12px', cursor: 'pointer' }}
                >
                  {/* Product Thumbnail Photo */}
                  <div
                    style={{
                      width: '48px',
                      height: '48px',
                      borderRadius: 'var(--radius-md)',
                      backgroundColor: '#f8fafc',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      overflow: 'hidden',
                      flexShrink: 0,
                      border: '1px solid var(--border-subtle)',
                    }}
                  >
                    {p.images && p.images.length > 0 ? (
                      <img
                        src={getImageUrl(p.images[0])}
                        alt={p.name}
                        style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                      />
                    ) : (
                      <Package size={22} color="var(--text-muted)" style={{ opacity: 0.6 }} />
                    )}
                  </div>

                  <div style={{ flex: 1, minWidth: 0, paddingRight: '8px' }}>
                    <div style={{ fontWeight: 700, fontSize: '0.92rem', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {p.name}
                    </div>
                    <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '2px' }}>
                      SKU: <span style={{ fontFamily: 'monospace', fontWeight: 700 }}>{p.sku || 'N/A'}</span> • Price: <strong>₹{p.price}</strong>
                    </div>

                    {/* Flexible Attributes Badges (Company, Size, Color, Gender, Type) */}
                    {p.attributes && Object.keys(p.attributes).length > 0 && (
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px', marginTop: '6px' }}>
                        {p.attributes.company && (
                          <span className="badge badge-info" style={{ fontSize: '0.7rem' }}>
                            🏷️ {p.attributes.company}
                          </span>
                        )}
                        {p.attributes.product_type && (
                          <span className="badge badge-warning" style={{ fontSize: '0.7rem' }}>
                            {p.attributes.product_type}
                          </span>
                        )}
                        {p.attributes.size && (
                          <span className="badge badge-muted" style={{ fontSize: '0.7rem' }}>
                            Size: {p.attributes.size}
                          </span>
                        )}
                        {p.attributes.color && (
                          <span className="badge badge-muted" style={{ fontSize: '0.7rem' }}>
                            🎨 {p.attributes.color}
                          </span>
                        )}
                        {p.attributes.gender && (
                          <span className="badge badge-muted" style={{ fontSize: '0.7rem' }}>
                            👤 {p.attributes.gender}
                          </span>
                        )}
                      </div>
                    )}
                  </div>

                  <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '8px', flexShrink: 0 }}>
                    <div style={{ textAlign: 'right' }}>
                      <div
                        style={{
                          fontWeight: 800,
                          fontSize: '0.95rem',
                          color: getProductStock(p) <= 5 ? 'var(--color-danger)' : 'var(--color-success)',
                        }}
                      >
                        {getProductStock(p)} pcs
                      </div>
                      <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)' }}>Stock</div>
                    </div>
                    <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleOpenEditProduct(p);
                        }}
                        className="btn btn-secondary btn-sm"
                        style={{ padding: '5px 8px', fontSize: '0.75rem', display: 'flex', alignItems: 'center', gap: '3px' }}
                        title="Edit Details & Photos"
                      >
                        <Edit3 size={12} /> Edit
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setSelectedProduct(p);
                        }}
                        className="btn btn-secondary btn-sm"
                        style={{ padding: '5px 8px', fontSize: '0.75rem' }}
                      >
                        Stock
                      </button>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </>
      ) : (
        /* Low Stock Tab */
        <div>
          <div
            className="card"
            style={{
              backgroundColor: 'var(--color-warning-light)',
              border: '1px solid #fde68a',
              padding: '16px',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px' }}>
              <AlertTriangle size={26} color="var(--color-warning)" style={{ flexShrink: 0 }} />
              <div>
                <div style={{ fontWeight: 800, fontSize: '0.95rem', color: '#92400e' }}>
                  Wholesale Reorder PDF Sheet
                </div>
                <div style={{ fontSize: '0.8rem', color: '#b45309', margin: '4px 0 12px 0' }}>
                  Aapke dukan ke {lowStockItems.length} saman khatam hone wale hain. 
                  Is PDF sheet ko download karke seedhe wholesale supplier ko bhej sakte hain.
                </div>
                <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
                  <a
                    href={inventoryApi.getReorderSheetUrl()}
                    target="_blank"
                    rel="noreferrer"
                    className="btn btn-sm"
                    style={{
                      backgroundColor: '#92400e',
                      color: '#ffffff',
                      fontWeight: 700,
                      textDecoration: 'none',
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '6px',
                    }}
                  >
                    <FileDown size={16} /> Reorder Sheet PDF
                  </a>

                  <button
                    onClick={() => handleBulkRestock(10)}
                    disabled={bulkRestocking}
                    style={{
                      backgroundColor: '#10b981',
                      color: '#ffffff',
                      border: 'none',
                      padding: '6px 12px',
                      borderRadius: 'var(--radius-md)',
                      fontWeight: 700,
                      fontSize: '0.78rem',
                      cursor: 'pointer',
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '6px',
                    }}
                  >
                    <RefreshCw size={14} className={bulkRestocking ? 'spin' : ''} />
                    {bulkRestocking ? 'Restocking...' : '1-Click Restock (+10)'}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div className="card" style={{ padding: '8px 12px' }}>
            <div style={{ padding: '8px 4px', fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-secondary)' }}>
              KHATAM HONE WALE PRODUCTS ({lowStockItems.length})
            </div>

            {lowStockItems.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '24px', color: 'var(--color-success)', fontSize: '0.85rem' }}>
                <CheckCircle size={32} style={{ margin: '0 auto 6px auto', display: 'block' }} />
                Sabhi products ka stock accha hai! Koi item khatam nahi ho raha.
              </div>
            ) : (
              lowStockItems.map((item) => (
                <div
                  key={item.product_id}
                  className="list-item"
                  style={{ padding: '10px 0' }}
                >
                  <div>
                    <div style={{ fontWeight: 700, fontSize: '0.88rem', color: 'var(--text-primary)' }}>
                      {item.name}
                    </div>
                    <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                      SKU: {item.sku} • Price: ₹{item.price}
                    </div>
                  </div>

                  <div style={{ textAlign: 'right' }}>
                    <div style={{ fontWeight: 800, fontSize: '0.95rem', color: 'var(--color-danger)' }}>
                      {item.current_stock} bacha hai
                    </div>
                    <div style={{ fontSize: '0.72rem', color: '#b45309', fontWeight: 600 }}>
                      Suggested: +{item.suggested_reorder_qty} pcs
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {/* Stock Adjust Modal */}
      {selectedProduct && (
        <div className="modal-backdrop" onClick={() => setSelectedProduct(null)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <h3 style={{ fontSize: '1.15rem', fontWeight: 800, marginBottom: '4px' }}>
              Stock Adjust: {selectedProduct.name}
            </h3>
            <p style={{ fontSize: '0.82rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
              Current Stock: <strong>{getProductStock(selectedProduct)} pcs</strong>
            </p>

            <form onSubmit={handleStockAdjust}>
              <div className="form-group">
                <label className="form-label">
                  Kitna Stock Badhana ya Ghatana Hai? (+50 ya -5)
                </label>
                <input
                  type="number"
                  required
                  className="form-input"
                  placeholder="e.g. +20 naya maal aaya ya -2 damage hua"
                  value={adjustmentQty}
                  onChange={(e) => setAdjustmentQty(e.target.value)}
                />
              </div>

              <div className="form-group">
                <label className="form-label">Vajah / Notes</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder="e.g. Supplier se stock aaya"
                  value={adjustNotes}
                  onChange={(e) => setAdjustNotes(e.target.value)}
                />
              </div>

              <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
                <button type="submit" className="btn btn-primary btn-block" disabled={adjustLoading}>
                  {adjustLoading ? 'Update ho raha hai...' : 'Stock Save Karein'}
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setSelectedProduct(null)}
                >
                  Radd
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Product Modal with Flexible Attributes (MongoDB in PostgreSQL) */}
      {showAddProductModal && (
        <div className="modal-backdrop" onClick={() => setShowAddProductModal(false)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()}>
            <div className="sheet-handle" />
            <h3 style={{ fontSize: '1.25rem', fontWeight: 800, marginBottom: '4px' }}>
              Naya Product Add Karein
            </h3>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '14px' }}>
              Product details & flexible specifications bharein.
            </p>

            {error && (
              <div style={{ color: 'var(--color-danger)', fontSize: '0.82rem', marginBottom: '10px' }}>
                {error}
              </div>
            )}

            <form onSubmit={handleAddProduct}>
              {/* Product Name */}
              <div className="form-group">
                <label className="form-label">Product Ka Naam</label>
                <input
                  type="text"
                  required
                  className="form-input"
                  placeholder="e.g. Peter England Slim Fit Shirt (Blue) ya Fortune Oil 1L"
                  value={newProductForm.name}
                  onChange={handleProductNameChange}
                />
              </div>

              {/* Smart Auto-Generated SKU Code */}
              <div className="form-group">
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                  <label className="form-label" style={{ margin: 0 }}>
                    SKU / Barcode (Automatic Generated)
                  </label>
                  <button
                    type="button"
                    onClick={handleRegenerateSKU}
                    style={{
                      border: 'none',
                      background: 'transparent',
                      color: 'var(--color-primary)',
                      fontSize: '0.75rem',
                      fontWeight: 700,
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                    }}
                  >
                    <RefreshCw size={12} /> Naya Code
                  </button>
                </div>
                <div className="sku-generator-box">
                  <span className="sku-code-text">
                    {newProductForm.sku || 'Naam likhte hi banega'}
                  </span>
                  <input
                    type="text"
                    required
                    style={{
                      border: 'none',
                      background: 'transparent',
                      fontSize: '0.85rem',
                      width: '80px',
                      color: 'var(--text-secondary)',
                      outline: 'none',
                      textAlign: 'right',
                    }}
                    value={newProductForm.sku}
                    onChange={(e) => setNewProductForm({ ...newProductForm, sku: e.target.value })}
                  />
                </div>
              </div>

              {/* Product Photos Upload (Max 4) */}
              <div className="form-group">
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                  <label className="form-label" style={{ margin: 0 }}>
                    Product Ki Photos (Max 4)
                  </label>
                  <span style={{ fontSize: '0.72rem', color: 'var(--text-muted)' }}>
                    {productImages.length}/4 selected
                  </span>
                </div>

                {productImagePreviews.length > 0 && (
                  <div style={{ display: 'flex', gap: '8px', marginBottom: '8px', flexWrap: 'wrap' }}>
                    {productImagePreviews.map((url, idx) => (
                      <div key={idx} style={{ position: 'relative' }}>
                        <img
                          src={url}
                          alt={`Product Preview ${idx + 1}`}
                          style={{
                            width: '60px',
                            height: '60px',
                            borderRadius: 'var(--radius-sm)',
                            objectFit: 'cover',
                            border: '1px solid var(--border-subtle)',
                          }}
                        />
                        <button
                          type="button"
                          onClick={() => handleRemoveProductImage(idx)}
                          style={{
                            position: 'absolute',
                            top: '-5px',
                            right: '-5px',
                            backgroundColor: 'var(--color-danger)',
                            color: '#ffffff',
                            borderRadius: '50%',
                            width: '18px',
                            height: '18px',
                            border: 'none',
                            cursor: 'pointer',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                          }}
                          title="Hataayein"
                        >
                          <X size={11} />
                        </button>
                      </div>
                    ))}
                  </div>
                )}

                {productImages.length < 4 && (
                  <div>
                    <label
                      htmlFor="modal-product-images-input"
                      className="btn btn-secondary btn-sm"
                      style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
                    >
                      <Upload size={14} /> Photos Chunein (Gallery/Camera)
                    </label>
                    <input
                      id="modal-product-images-input"
                      type="file"
                      multiple
                      accept="image/jpeg,image/png,image/webp"
                      onChange={handleProductImageSelect}
                      style={{ display: 'none' }}
                    />
                  </div>
                )}
              </div>

              {/* Visual Category Selector (Blinkit Style Pill Grid) */}
              <div className="form-group">
                <label className="form-label">Category Chuniye</label>
                <div className="category-pill-grid">
                  {categories.map((cat) => {
                    const isSelected = newProductForm.category_id === cat.id;
                    const emoji = getCategoryEmoji(cat.slug || cat.name);
                    return (
                      <button
                        key={cat.id}
                        type="button"
                        className={`category-pill-btn ${isSelected ? 'active' : ''}`}
                        onClick={() => setNewProductForm({ ...newProductForm, category_id: cat.id })}
                      >
                        <span style={{ fontSize: '1.2rem' }}>{emoji}</span>
                        <span style={{ flex: 1, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {cat.name}
                        </span>
                        {isSelected && <Check size={14} color="var(--color-primary)" />}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Price & Cost Price */}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
                <div className="form-group">
                  <label className="form-label">Selling Price (₹)</label>
                  <input
                    type="number"
                    step="0.01"
                    required
                    className="form-input"
                    placeholder="999"
                    value={newProductForm.price}
                    onChange={(e) => setNewProductForm({ ...newProductForm, price: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Cost Price (Kharid ₹)</label>
                  <input
                    type="number"
                    step="0.01"
                    className="form-input"
                    placeholder="750"
                    value={newProductForm.cost_price}
                    onChange={(e) => setNewProductForm({ ...newProductForm, cost_price: e.target.value })}
                  />
                </div>
              </div>

              {/* Stock Quantity */}
              <div className="form-group">
                <label className="form-label">Shuruaati Stock (Units/Pcs)</label>
                <input
                  type="number"
                  className="form-input"
                  placeholder="20"
                  value={newProductForm.stock_quantity}
                  onChange={(e) => setNewProductForm({ ...newProductForm, stock_quantity: e.target.value })}
                />
              </div>

              {/* 🚀 FLEXIBLE PRODUCT ATTRIBUTES SECTION (MongoDB inside PostgreSQL) */}
              <div
                style={{
                  backgroundColor: 'var(--bg-surface-subtle)',
                  borderRadius: 'var(--radius-md)',
                  padding: '12px',
                  marginBottom: '16px',
                  border: '1.5px solid var(--border-subtle)',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '10px',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <Sliders size={16} color="var(--color-primary)" />
                    <span style={{ fontWeight: 800, fontSize: '0.85rem' }}>
                      Product Specifications & Features
                    </span>
                  </div>
                  <span style={{ fontSize: '0.72rem', color: 'var(--color-primary)', fontWeight: 700 }}>
                    Flexible JSONB
                  </span>
                </div>

                {/* Company / Brand & Model */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
                  <div className="form-group" style={{ marginBottom: '8px' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Company / Brand</label>
                    <input
                      type="text"
                      className="form-input"
                      style={{ padding: '8px 10px', fontSize: '0.85rem' }}
                      placeholder="e.g. Peter England / Samsung / Tata"
                      value={newProductForm.attributes.company}
                      onChange={(e) => handleAttributeChange('company', e.target.value)}
                    />
                  </div>

                  <div className="form-group" style={{ marginBottom: '8px' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Model / Variety</label>
                    <input
                      type="text"
                      className="form-input"
                      style={{ padding: '8px 10px', fontSize: '0.85rem' }}
                      placeholder="e.g. Slim Fit / M34 / Banarasi"
                      value={newProductForm.attributes.model}
                      onChange={(e) => handleAttributeChange('model', e.target.value)}
                    />
                  </div>
                </div>

                {/* Type & Size */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
                  <div className="form-group" style={{ marginBottom: '8px' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Type (Shirt, Saree, Oil)</label>
                    <input
                      type="text"
                      className="form-input"
                      style={{ padding: '8px 10px', fontSize: '0.85rem' }}
                      placeholder="e.g. Shirt, Saree, Jeans, Kurti"
                      value={newProductForm.attributes.product_type}
                      onChange={(e) => handleAttributeChange('product_type', e.target.value)}
                    />
                  </div>

                  <div className="form-group" style={{ marginBottom: '8px' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Size (S, M, L, XL, 1L, etc.)</label>
                    <input
                      type="text"
                      className="form-input"
                      style={{ padding: '8px 10px', fontSize: '0.85rem' }}
                      placeholder="e.g. XL, 38, Free Size, 1L"
                      value={newProductForm.attributes.size}
                      onChange={(e) => handleAttributeChange('size', e.target.value)}
                    />
                  </div>
                </div>

                {/* Color & Gender */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
                  <div className="form-group" style={{ marginBottom: '8px' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Color (Rang)</label>
                    <input
                      type="text"
                      className="form-input"
                      style={{ padding: '8px 10px', fontSize: '0.85rem' }}
                      placeholder="e.g. Navy Blue, Maroon, Black"
                      value={newProductForm.attributes.color}
                      onChange={(e) => handleAttributeChange('color', e.target.value)}
                    />
                  </div>

                  <div className="form-group" style={{ marginBottom: '8px' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Gender</label>
                    <select
                      className="form-select"
                      style={{ padding: '8px 10px', fontSize: '0.85rem' }}
                      value={newProductForm.attributes.gender}
                      onChange={(e) => handleAttributeChange('gender', e.target.value)}
                    >
                      <option value="">-- Choose Gender --</option>
                      <option value="Men">Men (Purush)</option>
                      <option value="Women">Women (Mahila)</option>
                      <option value="Boys">Boys (Ladke)</option>
                      <option value="Girls">Girls (Ladkiyan)</option>
                      <option value="Kids">Kids (Bacche)</option>
                      <option value="Unisex">Unisex (Sabhi)</option>
                    </select>
                  </div>
                </div>

                {/* Season & Age Group */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
                  <div className="form-group" style={{ marginBottom: '8px' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Season (Mausam)</label>
                    <select
                      className="form-select"
                      style={{ padding: '8px 10px', fontSize: '0.85rem' }}
                      value={newProductForm.attributes.season}
                      onChange={(e) => handleAttributeChange('season', e.target.value)}
                    >
                      <option value="">-- Choose Season --</option>
                      <option value="Summer">Summer (Garmi)</option>
                      <option value="Winter">Winter (Sardi)</option>
                      <option value="Monsoon">Monsoon (Barish)</option>
                      <option value="All Season">All Season (Hamesha)</option>
                    </select>
                  </div>

                  <div className="form-group" style={{ marginBottom: '8px' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Age Group</label>
                    <input
                      type="text"
                      className="form-input"
                      style={{ padding: '8px 10px', fontSize: '0.85rem' }}
                      placeholder="e.g. Adults / 5-10 yrs / Teens"
                      value={newProductForm.attributes.age_group}
                      onChange={(e) => handleAttributeChange('age_group', e.target.value)}
                    />
                  </div>
                </div>

                {/* Custom Key-Value Dynamic Attributes List */}
                {customAttributes.map((attr, idx) => (
                  <div key={idx} style={{ display: 'flex', gap: '6px', alignItems: 'center', marginBottom: '6px' }}>
                    <input
                      type="text"
                      className="form-input"
                      style={{ flex: 1, padding: '7px 9px', fontSize: '0.8rem' }}
                      placeholder="Field Name (e.g. Fabric)"
                      value={attr.key}
                      onChange={(e) => updateCustomAttribute(idx, 'key', e.target.value)}
                    />
                    <input
                      type="text"
                      className="form-input"
                      style={{ flex: 1, padding: '7px 9px', fontSize: '0.8rem' }}
                      placeholder="Value (e.g. Pure Silk)"
                      value={attr.value}
                      onChange={(e) => updateCustomAttribute(idx, 'value', e.target.value)}
                    />
                    <button
                      type="button"
                      onClick={() => removeCustomAttribute(idx)}
                      style={{
                        border: 'none',
                        background: 'transparent',
                        color: 'var(--color-danger)',
                        cursor: 'pointer',
                        padding: '4px',
                      }}
                    >
                      <X size={16} />
                    </button>
                  </div>
                ))}

                <button
                  type="button"
                  onClick={addCustomAttribute}
                  style={{
                    border: '1px dashed var(--color-primary)',
                    background: 'transparent',
                    color: 'var(--color-primary)',
                    borderRadius: 'var(--radius-sm)',
                    padding: '6px 10px',
                    fontSize: '0.75rem',
                    fontWeight: 700,
                    cursor: 'pointer',
                    width: '100%',
                    marginTop: '4px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: '4px',
                  }}
                >
                  <Plus size={14} /> + Aur Custom Field Jodein (Warranty, Material, Pattern)
                </button>
              </div>

              <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
                <button type="submit" className="btn btn-primary btn-block btn-lg" disabled={addProductLoading}>
                  {addProductLoading ? 'Product ban raha hai...' : 'Product Catalog Me Jodein'}
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setShowAddProductModal(false)}
                >
                  Radd
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Product Detail Inspection Modal */}
      {inspectedProduct && (
        <ProductDetailModal
          product={inspectedProduct}
          onClose={() => setInspectedProduct(null)}
          onEditProduct={(p) => handleOpenEditProduct(p)}
          onAdjustStock={(p) => {
            setSelectedProduct(p);
            setAdjustmentQty('');
            setAdjustNotes('');
          }}
          isMerchant={true}
        />
      )}

      {/* Edit Product Modal with Full Photos & Details Editing */}
      {editingProduct && editProductForm && (
        <div className="modal-backdrop" onClick={() => setEditingProduct(null)}>
          <div className="bottom-sheet" onClick={(e) => e.stopPropagation()} style={{ maxHeight: '90vh', overflowY: 'auto' }}>
            <div className="sheet-handle" />
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
              <div>
                <h3 style={{ fontSize: '1.25rem', fontWeight: 800, margin: 0 }}>
                  Product Edit Karein
                </h3>
                <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', margin: '2px 0 0 0' }}>
                  SKU: {editingProduct.sku}
                </p>
              </div>
              <button
                type="button"
                onClick={() => handleDeleteProduct(editingProduct.id)}
                style={{
                  backgroundColor: 'rgba(239, 68, 68, 0.1)',
                  color: 'var(--color-danger)',
                  border: 'none',
                  padding: '6px 10px',
                  borderRadius: 'var(--radius-md)',
                  fontSize: '0.78rem',
                  fontWeight: 700,
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                }}
              >
                <Trash2 size={14} /> Delete
              </button>
            </div>

            {editError && (
              <div style={{ color: 'var(--color-danger)', backgroundColor: '#fee2e2', padding: '8px 12px', borderRadius: 'var(--radius-sm)', fontSize: '0.82rem', marginBottom: '12px' }}>
                {editError}
              </div>
            )}

            <form onSubmit={handleSaveEditProduct}>
              {/* Product Name */}
              <div className="form-group">
                <label className="form-label">Product Ka Naam</label>
                <input
                  type="text"
                  required
                  className="form-input"
                  value={editProductForm.name}
                  onChange={(e) => setEditProductForm({ ...editProductForm, name: e.target.value })}
                />
              </div>

              {/* Product Photos Section */}
              <div className="form-group">
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                  <label className="form-label" style={{ margin: 0 }}>
                    Product Photos ({editExistingImages.length + editNewImages.length}/4)
                  </label>
                  <span style={{ fontSize: '0.72rem', color: 'var(--text-muted)' }}>
                    Max 4 photos
                  </span>
                </div>

                {/* Previews Grid */}
                <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', marginBottom: '8px' }}>
                  {/* Existing Saved Photos */}
                  {editExistingImages.map((url, idx) => (
                    <div key={`existing-${idx}`} style={{ position: 'relative' }}>
                      <img
                        src={getImageUrl(url)}
                        alt={`Photo ${idx + 1}`}
                        style={{
                          width: '64px',
                          height: '64px',
                          borderRadius: 'var(--radius-sm)',
                          objectFit: 'cover',
                          border: '1px solid var(--border-subtle)',
                        }}
                      />
                      <button
                        type="button"
                        onClick={() => handleRemoveEditExistingImage(idx)}
                        style={{
                          position: 'absolute',
                          top: '-5px',
                          right: '-5px',
                          backgroundColor: 'var(--color-danger)',
                          color: '#ffffff',
                          borderRadius: '50%',
                          width: '18px',
                          height: '18px',
                          border: 'none',
                          cursor: 'pointer',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                        title="Hataayein"
                      >
                        <X size={11} />
                      </button>
                    </div>
                  ))}

                  {/* Newly Added Photos */}
                  {editNewImagePreviews.map((url, idx) => (
                    <div key={`new-${idx}`} style={{ position: 'relative' }}>
                      <img
                        src={url}
                        alt={`New ${idx + 1}`}
                        style={{
                          width: '64px',
                          height: '64px',
                          borderRadius: 'var(--radius-sm)',
                          objectFit: 'cover',
                          border: '2px solid var(--color-primary)',
                        }}
                      />
                      <button
                        type="button"
                        onClick={() => handleRemoveEditNewImage(idx)}
                        style={{
                          position: 'absolute',
                          top: '-5px',
                          right: '-5px',
                          backgroundColor: 'var(--color-danger)',
                          color: '#ffffff',
                          borderRadius: '50%',
                          width: '18px',
                          height: '18px',
                          border: 'none',
                          cursor: 'pointer',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                        title="Hataayein"
                      >
                        <X size={11} />
                      </button>
                    </div>
                  ))}
                </div>

                {/* Upload Button */}
                {editExistingImages.length + editNewImages.length < 4 && (
                  <div>
                    <label
                      htmlFor="edit-product-images-input"
                      className="btn btn-secondary btn-sm"
                      style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
                    >
                      <Upload size={14} /> Nayi Photo Jodein (Camera / Gallery)
                    </label>
                    <input
                      id="edit-product-images-input"
                      type="file"
                      multiple
                      accept="image/jpeg,image/png,image/webp"
                      onChange={handleEditNewImagesSelect}
                      style={{ display: 'none' }}
                    />
                  </div>
                )}
              </div>

              {/* Price & Cost Price */}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
                <div className="form-group">
                  <label className="form-label">Selling Price (₹)</label>
                  <input
                    type="number"
                    step="0.01"
                    required
                    className="form-input"
                    value={editProductForm.price}
                    onChange={(e) => setEditProductForm({ ...editProductForm, price: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Cost Price (Kharid ₹)</label>
                  <input
                    type="number"
                    step="0.01"
                    className="form-input"
                    value={editProductForm.cost_price}
                    onChange={(e) => setEditProductForm({ ...editProductForm, cost_price: e.target.value })}
                  />
                </div>
              </div>

              {/* Stock Quantity */}
              <div className="form-group">
                <label className="form-label">Total Stock Quantity (Pieces/Units)</label>
                <input
                  type="number"
                  min="0"
                  required
                  className="form-input"
                  value={editProductForm.stock_quantity}
                  onChange={(e) => setEditProductForm({ ...editProductForm, stock_quantity: e.target.value })}
                />
              </div>

              {/* Category */}
              <div className="form-group">
                <label className="form-label">Category</label>
                <select
                  className="form-select"
                  value={editProductForm.category_id}
                  onChange={(e) => setEditProductForm({ ...editProductForm, category_id: e.target.value })}
                >
                  {categories.map((cat) => (
                    <option key={cat.id} value={cat.id}>
                      {cat.name}
                    </option>
                  ))}
                </select>
              </div>

              <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
                <button type="submit" className="btn btn-primary btn-block btn-lg" disabled={editProductLoading}>
                  {editProductLoading ? 'Save ho raha hai...' : 'Badlav Save Karein'}
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setEditingProduct(null)}
                >
                  Radd
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </AppLayout>
  );
};
