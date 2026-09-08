# ShopMe - Mobile-First Frontend (Smart Dukan & Merchant OS)

Yeh **ShopMe** Go backend ke sath seamlessly judne wala ek clean, high-performance aur 100% mobile-first React frontend hai.

---

## 🚀 Shuru Kaise Karein (Quick Start)

### 1. Backend Run Karein (Terminal 1)
```bash
# Project root directory me
go run cmd/api/main.go
# Backend chalega: http://localhost:8080
```

### 2. Frontend Run Karein (Terminal 2)
```bash
cd frontend
npm run dev
# Frontend chalega: http://localhost:5173
```

---

## 📁 File & Folder Structure (Kis File Ka Kya Kaam Hai)

```text
frontend/
├── src/
│   ├── api/                     # 🌐 Go Backend se baat karne wali files (1-to-1 API matching)
│   │   ├── client.js            # Axios client with JWT interceptor & baseURL (http://localhost:8080)
│   │   ├── auth.api.js          # /auth/login, /auth/register, password recovery
│   │   ├── shop.api.js          # /shops/me, /shops (Merchant profile & QR code)
│   │   ├── product.api.js       # /products, /products/scan/{code} (Barcode scan)
│   │   ├── pos.api.js           # /shops/me/pos/sale (Counter billing & daily summary)
│   │   ├── khata.api.js         # /shops/me/khata (Customer udhar & jama passbook)
│   │   ├── expense.api.js       # /shops/me/expenses (Daily store expenses)
│   │   ├── inventory.api.js     # /shops/me/inventory/adjust (Stock alerts & reorder PDF)
│   │   ├── analytics.api.js     # /shops/me/analytics/profit (Net pocket profit)
│   │   └── reservation.api.js   # /reservations (Click & Collect hold items)
│   │
│   ├── context/                 # 🧠 App Ka State Management
│   │   ├── AuthContext.jsx      # Login status, token persistence, user & shop state
│   │   └── POSContext.jsx       # Counter billing cart (+ / - quantity, totals, discount)
│   │
│   ├── components/layout/       # 📱 Mobile UI Shell
│   │   ├── AppLayout.jsx        # Mobile container wrapper
│   │   ├── AppHeader.jsx        # Sticky top header (shop status Online/Offline)
│   │   └── BottomNav.jsx        # Mobile bottom tab bar (Home, POS, Khata, Kharche, Stock)
│   │
│   ├── pages/                   # 📱 Mobile Screens
│   │   ├── auth/
│   │   │   ├── LoginScreen.jsx     # Dukaandar & Grahak login
│   │   │   └── RegisterScreen.jsx  # Naya account banana
│   │   │
│   │   ├── merchant/               # 🏪 Dukaandar Merchant OS
│   │   │   ├── DashboardScreen.jsx # Main home (Stats, Status switch, Quick actions, QR)
│   │   │   ├── POSScreen.jsx       # Fast counter billing terminal + digital PDF receipt
│   │   │   ├── KhataScreen.jsx     # Customer Udhar passbook + payment settlement
│   │   │   ├── ExpenseScreen.jsx   # Dukan ke roz ke kharche (Chai, bijli, rent)
│   │   │   ├── InventoryScreen.jsx # Stock adjust, low-stock warnings, reorder PDF
│   │   │   └── AnalyticsScreen.jsx # Real Net Pocket Profit & deadstock matrix
│   │   │
│   │   └── customer/               # 🛒 Grahak Marketplace & Storefront
│   │       ├── ExploreShopsScreen.jsx # Local shops khojna
│   │       ├── StorefrontScreen.jsx   # Shop profile & item hold/reservation
│   │       └── ReservationsScreen.jsx # Customer pickup OTP/codes list
│   │
│   ├── styles/                  # 🎨 Modern Clean Design System (Vanilla CSS)
│   │   ├── variables.css        # Colors, shadows, border-radii
│   │   ├── mobile.css           # Mobile shell layout & bottom nav styles
│   │   └── components.css       # Buttons, cards, modals, form inputs
│   │
│   ├── App.jsx                  # Main application routes & protected route guards
│   ├── main.jsx                 # React root entry
│   └── index.css                # Global stylesheet imports
```

---

## 🎯 Important Features Implemented

1. **Clean Code & Hinglish Comments**: Har file ke top par clear hint likhi hai ki backend ke kaunse controller se yeh file judi hai.
2. **Mobile-First App Experience**: Screen par bottom tabs, touch buttons, swipeable cards aur bottom sheets milti hain bilkul Android/iOS app jaisi.
3. **Backend-Matched Payloads**: DTOs aur JSON field names (jaise `customer_phone`, `discount_amount`, `payment_method`, `cost_price`, `stock_quantity`) Go backend ke sath 100% matched hain.
4. **Fast POS Terminal**: Barcode input box me barcode enter dabate hi item cart me chala jata hai aur bill bante hi PDF receipt generate ho jati hai.
5. **Customer Khata Book**: Market me total baki udhar ka hisab, customer-wise history aur jama ki entry.
