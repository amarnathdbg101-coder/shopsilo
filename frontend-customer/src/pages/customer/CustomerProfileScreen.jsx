/**
 * Dedicated Customer Profile Screen
 * 
 * Hinglish Hint:
 * Grahak (Customer) ka personal account page:
 * - Profile details (Naam, Phone, Email, Photo upload)
 * - Meri Pickups & Holds ka quick link
 * - Direct Logout button
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  User,
  Phone,
  Mail,
  ShoppingBag,
  Store,
  LogOut,
  Camera,
  ChevronRight,
  Shield,
  HelpCircle,
} from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { uploadApi } from '../../api/upload.api';
import { AppLayout } from '../../components/layout/AppLayout';
import { getImageUrl } from '../../utils/imageUrl';

export const CustomerProfileScreen = () => {
  const navigate = useNavigate();
  const { user, logout, updateUser } = useAuth();
  const [avatarUploading, setAvatarUploading] = useState(false);

  const handleAvatarChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (file.size > 2 * 1024 * 1024) {
      alert('Photo ka size 2MB se kam hona chahiye');
      return;
    }

    try {
      setAvatarUploading(true);
      const data = await uploadApi.uploadUserAvatar(file);
      updateUser({ avatar_url: data.avatar_url });
      alert('Profile photo safaltapoorvak update ho gayi!');
    } catch (err) {
      console.error('Avatar upload error:', err);
      alert('Avatar upload nahi ho saka: ' + (err.message || 'Error'));
    } finally {
      setAvatarUploading(false);
    }
  };

  const handleLogout = () => {
    if (window.confirm('Kya aap sach me logout karna chahte hain?')) {
      logout();
      navigate('/login');
    }
  };

  return (
    <AppLayout title="Mera Account">
      <div className="profile-container" style={{ padding: '1rem', maxWidth: '600px', margin: '0 auto' }}>
        
        {/* User Card */}
        <div className="card profile-user-card" style={{ padding: '1.5rem', textAlign: 'center', marginBottom: '1.25rem', background: '#1e293b', borderRadius: '16px' }}>
          <div style={{ position: 'relative', width: '90px', height: '90px', margin: '0 auto 1rem auto' }}>
            <div style={{
              width: '90px',
              height: '90px',
              borderRadius: '50%',
              backgroundColor: '#3b82f6',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'white',
              fontSize: '2rem',
              fontWeight: 700,
              overflow: 'hidden',
              boxShadow: '0 4px 14px rgba(59, 130, 246, 0.4)'
            }}>
              {user?.avatar_url ? (
                <img 
                  src={getImageUrl(user.avatar_url)} 
                  alt={user?.name || 'Customer'} 
                  style={{ width: '100%', height: '100%', objectFit: 'cover' }} 
                  onError={(e) => {
                    e.currentTarget.style.display = 'none';
                  }}
                />
              ) : (
                user?.name ? user.name[0].toUpperCase() : 'G'
              )}
            </div>

            <label 
              htmlFor="customer-avatar-upload"
              style={{
                position: 'absolute',
                bottom: '0',
                right: '0',
                background: '#2563eb',
                color: 'white',
                padding: '6px',
                borderRadius: '50%',
                cursor: 'pointer',
                boxShadow: '0 2px 6px rgba(0,0,0,0.4)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center'
              }}
              title="Profile photo badlein"
            >
              <Camera size={16} />
            </label>
            <input 
              id="customer-avatar-upload" 
              type="file" 
              accept="image/*" 
              onChange={handleAvatarChange} 
              style={{ display: 'none' }} 
              disabled={avatarUploading}
            />
          </div>

          {avatarUploading && (
            <p style={{ fontSize: '0.8rem', color: '#60a5fa', marginBottom: '0.5rem' }}>Photo upload ho rahi hai...</p>
          )}

          <h2 style={{ fontSize: '1.25rem', fontWeight: 600, color: '#f8fafc', margin: '0 0 0.25rem 0' }}>
            {user?.name || 'Grahak'}
          </h2>
          <span style={{ 
            display: 'inline-block', 
            background: 'rgba(59, 130, 246, 0.15)', 
            color: '#60a5fa', 
            fontSize: '0.75rem', 
            padding: '2px 10px', 
            borderRadius: '12px',
            fontWeight: 500,
            marginBottom: '1rem'
          }}>
            Grahak (Buyer)
          </span>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', textAlign: 'left', background: 'rgba(15, 23, 42, 0.6)', padding: '0.85rem', borderRadius: '12px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem', color: '#cbd5e1', fontSize: '0.85rem' }}>
              <Phone size={16} style={{ color: '#3b82f6' }} />
              <span>{user?.phone || 'Phone number joda nahi'}</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem', color: '#cbd5e1', fontSize: '0.85rem' }}>
              <Mail size={16} style={{ color: '#3b82f6' }} />
              <span>{user?.email || 'Email missing'}</span>
            </div>
          </div>
        </div>

        {/* Action Shortcuts */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem', marginBottom: '1.5rem' }}>
          
          <div 
            onClick={() => navigate('/reservations')}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '1rem',
              background: '#1e293b',
              borderRadius: '14px',
              cursor: 'pointer',
              border: '1px solid rgba(255, 255, 255, 0.05)'
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
              <div style={{ background: 'rgba(16, 185, 129, 0.15)', color: '#34d399', padding: '8px', borderRadius: '10px' }}>
                <ShoppingBag size={20} />
              </div>
              <div>
                <h4 style={{ margin: 0, fontSize: '0.95rem', color: '#f8fafc', fontWeight: 600 }}>Meri Bookings & Pickups</h4>
                <p style={{ margin: 0, fontSize: '0.8rem', color: '#94a3b8' }}>Dukaan par pickup ke liye hold kiye gaye items</p>
              </div>
            </div>
            <ChevronRight size={18} style={{ color: '#64748b' }} />
          </div>

          <div 
            onClick={() => navigate('/')}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '1rem',
              background: '#1e293b',
              borderRadius: '14px',
              cursor: 'pointer',
              border: '1px solid rgba(255, 255, 255, 0.05)'
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
              <div style={{ background: 'rgba(59, 130, 246, 0.15)', color: '#60a5fa', padding: '8px', borderRadius: '10px' }}>
                <Store size={20} />
              </div>
              <div>
                <h4 style={{ margin: 0, fontSize: '0.95rem', color: '#f8fafc', fontWeight: 600 }}>Aas-paas ki Dukaanein</h4>
                <p style={{ margin: 0, fontSize: '0.8rem', color: '#94a3b8' }}>Apne ilaqe ki certified dukaano se khareedari karein</p>
              </div>
            </div>
            <ChevronRight size={18} style={{ color: '#64748b' }} />
          </div>

        </div>

        {/* Logout */}
        <button
          onClick={handleLogout}
          style={{
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '0.5rem',
            padding: '0.9rem',
            background: 'rgba(239, 68, 68, 0.1)',
            color: '#f87171',
            border: '1px solid rgba(239, 68, 68, 0.2)',
            borderRadius: '12px',
            cursor: 'pointer',
            fontWeight: 600,
            fontSize: '0.95rem',
            transition: 'all 0.2s'
          }}
        >
          <LogOut size={18} />
          <span>Account Se Logout Karein</span>
        </button>

      </div>
    </AppLayout>
  );
};
