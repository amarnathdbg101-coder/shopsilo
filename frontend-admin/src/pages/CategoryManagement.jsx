import React, { useState, useEffect } from 'react';
import { 
  FolderTree, 
  Plus, 
  Search, 
  Edit2, 
  Trash2, 
  RefreshCw, 
  CheckCircle2, 
  XCircle, 
  Tag, 
  Box, 
  X,
  ExternalLink,
  Image as ImageIcon
} from 'lucide-react';
import { adminApi } from '../api/admin.api';

export const CategoryManagement = ({ onRefreshStats }) => {
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  
  // Modal state
  const [showModal, setShowModal] = useState(false);
  const [editingCategory, setEditingCategory] = useState(null);
  const [formData, setFormData] = useState({
    name: '',
    slug: '',
    description: '',
    image_url: '',
    is_active: true,
  });
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');

  useEffect(() => {
    fetchCategories();
  }, []);

  const fetchCategories = async () => {
    setLoading(true);
    try {
      const data = await adminApi.getCategories();
      setCategories(data || []);
    } catch (err) {
      console.error('Failed to load categories:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenCreate = () => {
    setEditingCategory(null);
    setFormData({
      name: '',
      slug: '',
      description: '',
      image_url: '',
      is_active: true,
    });
    setFormError('');
    setShowModal(true);
  };

  const handleOpenEdit = (cat) => {
    setEditingCategory(cat);
    setFormData({
      name: cat.name || '',
      slug: cat.slug || '',
      description: cat.description || '',
      image_url: cat.image_url || '',
      is_active: cat.is_active !== undefined ? cat.is_active : true,
    });
    setFormError('');
    setShowModal(true);
  };

  const handleNameChange = (e) => {
    const val = e.target.value;
    if (!editingCategory) {
      // Auto slugify if creating
      const autoSlug = val
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '');
      setFormData((prev) => ({ ...prev, name: val, slug: autoSlug }));
    } else {
      setFormData((prev) => ({ ...prev, name: val }));
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!formData.name.trim()) {
      setFormError('Category name is required.');
      return;
    }

    setSubmitting(true);
    setFormError('');
    try {
      if (editingCategory) {
        await adminApi.updateCategory(editingCategory.id, {
          name: formData.name.trim(),
          slug: formData.slug.trim(),
          description: formData.description.trim(),
          image_url: formData.image_url.trim(),
          is_active: formData.is_active,
        });
      } else {
        await adminApi.createCategory({
          name: formData.name.trim(),
          slug: formData.slug.trim(),
          description: formData.description.trim(),
          image_url: formData.image_url.trim(),
        });
      }
      setShowModal(false);
      await fetchCategories();
      if (onRefreshStats) onRefreshStats();
    } catch (err) {
      setFormError(err.message || 'Operation failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (cat) => {
    if (!window.confirm(`Are you sure you want to delete category "${cat.name}"? Products in this category will become uncategorized.`)) {
      return;
    }
    try {
      await adminApi.deleteCategory(cat.id);
      await fetchCategories();
      if (onRefreshStats) onRefreshStats();
    } catch (err) {
      alert(err.message);
    }
  };

  const filteredCategories = categories.filter((c) => {
    const q = searchQuery.toLowerCase();
    return (
      (c.name || '').toLowerCase().includes(q) ||
      (c.slug || '').toLowerCase().includes(q) ||
      (c.description || '').toLowerCase().includes(q)
    );
  });

  return (
    <div>
      <div className="content-card">
        <div className="card-header-bar">
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <div className="search-input-wrap">
              <Search size={16} className="search-icon" />
              <input
                type="text"
                placeholder="Search categories..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
            <span style={{ fontSize: '0.85rem', color: '#94a3b8' }}>
              {filteredCategories.length} categories
            </span>
          </div>

          <button className="btn btn-primary" onClick={handleOpenCreate}>
            <Plus size={16} /> Add Category
          </button>
        </div>

        {loading ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#94a3b8' }}>
            <RefreshCw size={24} className="spin" style={{ margin: '0 auto 12px' }} />
            Loading catalog categories...
          </div>
        ) : filteredCategories.length === 0 ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#64748b' }}>
            <FolderTree size={36} style={{ margin: '0 auto 12px', opacity: 0.4 }} />
            <p>No categories found.</p>
          </div>
        ) : (
          <div className="table-responsive">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>Category</th>
                  <th>Slug</th>
                  <th>Description</th>
                  <th>Active Products</th>
                  <th>Status</th>
                  <th style={{ textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filteredCategories.map((c) => (
                  <tr key={c.id}>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <div style={{
                          width: '40px',
                          height: '40px',
                          borderRadius: '10px',
                          background: '#1e293b',
                          border: '1px solid #334155',
                          overflow: 'hidden',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          flexShrink: 0
                        }}>
                          {c.image_url ? (
                            <img 
                              src={c.image_url} 
                              alt={c.name} 
                              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                              onError={(e) => { e.target.style.display = 'none'; }} 
                            />
                          ) : (
                            <Tag size={18} color="#94a3b8" />
                          )}
                        </div>
                        <div>
                          <div style={{ fontWeight: '700', color: '#fff', fontSize: '0.95rem' }}>{c.name}</div>
                          <div style={{ fontSize: '0.75rem', color: '#64748b' }}>ID: {c.id.slice(0, 8)}...</div>
                        </div>
                      </div>
                    </td>
                    <td>
                      <code style={{ fontSize: '0.78rem', color: '#a5b4fc', background: '#1e293b', padding: '3px 8px', borderRadius: '4px' }}>
                        /{c.slug}
                      </code>
                    </td>
                    <td style={{ maxWidth: '280px', color: '#94a3b8', fontSize: '0.85rem' }}>
                      {c.description ? (
                        <div style={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {c.description}
                        </div>
                      ) : (
                        <span style={{ color: '#475569' }}>No description</span>
                      )}
                    </td>
                    <td>
                      <span style={{ 
                        display: 'inline-flex', 
                        alignItems: 'center', 
                        gap: '6px', 
                        fontWeight: '700', 
                        color: c.product_count > 0 ? '#60a5fa' : '#64748b',
                        fontSize: '0.85rem'
                      }}>
                        <Box size={14} /> {c.product_count || 0}
                      </span>
                    </td>
                    <td>
                      <span className={`status-pill ${c.is_active ? 'active' : 'suspended'}`}>
                        {c.is_active ? 'Active' : 'Hidden'}
                      </span>
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                        <button
                          className="btn btn-outline btn-sm"
                          onClick={() => handleOpenEdit(c)}
                          title="Edit Category"
                        >
                          <Edit2 size={14} /> Edit
                        </button>
                        <button
                          className="btn btn-danger btn-sm"
                          onClick={() => handleDelete(c)}
                          title="Delete Category"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create / Edit Modal */}
      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <div className="brand-icon" style={{ width: '32px', height: '32px' }}>
                  <FolderTree size={18} />
                </div>
                <h3 style={{ fontSize: '1.15rem' }}>
                  {editingCategory ? 'Edit Category' : 'Create New Category'}
                </h3>
              </div>
              <button className="btn-logout" onClick={() => setShowModal(false)}>
                <X size={20} />
              </button>
            </div>

            <form onSubmit={handleSubmit}>
              <div className="modal-body">
                {formError && (
                  <div style={{
                    background: 'rgba(239, 68, 68, 0.1)',
                    border: '1px solid #ef4444',
                    color: '#f87171',
                    padding: '10px 14px',
                    borderRadius: '8px',
                    marginBottom: '16px',
                    fontSize: '0.85rem'
                  }}>
                    {formError}
                  </div>
                )}

                <div className="form-group">
                  <label className="form-label">Category Name *</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="e.g. Grocery & Staples, Fashion, Electronics"
                    value={formData.name}
                    onChange={handleNameChange}
                    required
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">URL Slug</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="e.g. grocery-staples"
                    value={formData.slug}
                    onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                  />
                  <span style={{ fontSize: '0.72rem', color: '#64748b', marginTop: '4px', display: 'block' }}>
                    Used in web links: /categories/{formData.slug || 'slug'}
                  </span>
                </div>

                <div className="form-group">
                  <label className="form-label">Description (Optional)</label>
                  <textarea
                    className="form-textarea"
                    rows="3"
                    placeholder="Short description of products in this category..."
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Image or Icon URL (Optional)</label>
                  <input
                    type="url"
                    className="form-input"
                    placeholder="https://images.example.com/grocery.jpg"
                    value={formData.image_url}
                    onChange={(e) => setFormData({ ...formData, image_url: e.target.value })}
                  />
                  {formData.image_url && (
                    <div style={{ marginTop: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <span style={{ fontSize: '0.72rem', color: '#94a3b8' }}>Preview:</span>
                      <img 
                        src={formData.image_url} 
                        alt="Preview" 
                        style={{ width: '36px', height: '36px', borderRadius: '6px', objectFit: 'cover', border: '1px solid #334155' }}
                        onError={(e) => { e.target.style.display = 'none'; }}
                      />
                    </div>
                  )}
                </div>

                {editingCategory && (
                  <div className="form-group" style={{ display: 'flex', alignItems: 'center', gap: '10px', marginTop: '12px' }}>
                    <input
                      type="checkbox"
                      id="is_active"
                      checked={formData.is_active}
                      onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                      style={{ width: '18px', height: '18px', cursor: 'pointer', accentColor: '#3b82f6' }}
                    />
                    <label htmlFor="is_active" style={{ fontSize: '0.85rem', color: '#f8fafc', cursor: 'pointer' }}>
                      Active (Visible to customers on browsing screens)
                    </label>
                  </div>
                )}
              </div>

              <div className="modal-footer">
                <button
                  type="button"
                  className="btn btn-outline"
                  onClick={() => setShowModal(false)}
                  disabled={submitting}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={submitting}
                >
                  {submitting ? (
                    <>
                      <RefreshCw size={14} className="spin" /> Saving...
                    </>
                  ) : editingCategory ? (
                    'Save Changes'
                  ) : (
                    'Create Category'
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
