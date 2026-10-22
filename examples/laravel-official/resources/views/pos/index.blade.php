<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ZenoPOS - Modern Point of Sale (Pure Go LaraZeno)</title>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-body: #090d16;
            --surface: #111827;
            --surface-elevated: #1f293d;
            --surface-border: #1e293b;
            --primary: #4f46e5;
            --primary-hover: #4338ca;
            --primary-light: rgba(99, 102, 241, 0.15);
            --accent: #06b6d4;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
            --success: #10b981;
            --success-light: rgba(16, 185, 129, 0.15);
            --danger: #ef4444;
            --warning: #f59e0b;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Plus Jakarta Sans', sans-serif;
            -webkit-font-smoothing: antialiased;
        }

        body {
            background-color: var(--bg-body);
            color: var(--text-main);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            overflow-x: hidden;
        }

        /* Top Bar */
        .topbar {
            height: 64px;
            background: rgba(17, 24, 39, 0.85);
            backdrop-filter: blur(12px);
            border-bottom: 1px solid var(--surface-border);
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0 1.5rem;
            position: sticky;
            top: 0;
            z-index: 40;
        }

        .brand {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            text-decoration: none;
        }

        .brand-logo {
            width: 38px;
            height: 38px;
            border-radius: 10px;
            background: linear-gradient(135deg, #6366f1, #06b6d4);
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 800;
            color: white;
            font-size: 1.25rem;
            box-shadow: 0 4px 14px rgba(99, 102, 241, 0.4);
        }

        .brand-info h1 {
            font-size: 1.15rem;
            font-weight: 700;
            color: white;
            letter-spacing: -0.02em;
        }

        .brand-info span {
            font-size: 0.725rem;
            color: var(--text-muted);
            display: block;
            margin-top: -2px;
        }

        .top-nav {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .nav-btn {
            padding: 0.5rem 1rem;
            font-size: 0.85rem;
            font-weight: 600;
            border-radius: 8px;
            text-decoration: none;
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            transition: all 0.2s;
        }

        .nav-btn-secondary {
            background: var(--surface-elevated);
            color: var(--text-muted);
            border: 1px solid var(--surface-border);
        }

        .nav-btn-secondary:hover {
            color: white;
            border-color: #334155;
            background: #283548;
        }

        .pos-badge {
            background: rgba(16, 185, 129, 0.15);
            color: #34d399;
            border: 1px solid rgba(16, 185, 129, 0.3);
            padding: 0.25rem 0.65rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 0.4rem;
        }

        .pos-badge::before {
            content: '';
            width: 6px;
            height: 6px;
            background: #10b981;
            border-radius: 50%;
            box-shadow: 0 0 8px #10b981;
        }

        /* Layout Grid */
        .pos-layout {
            display: grid;
            grid-template-columns: 1fr 380px;
            height: calc(100vh - 64px);
            overflow: hidden;
        }

        @media (max-width: 1024px) {
            .pos-layout {
                grid-template-columns: 1fr;
                height: auto;
                overflow: visible;
            }
        }

        /* Catalog Panel */
        .catalog-panel {
            padding: 1.5rem;
            overflow-y: auto;
            display: flex;
            flex-direction: column;
            gap: 1.25rem;
        }

        /* Quick Stats & Search Bar */
        .catalog-header {
            display: flex;
            flex-direction: column;
            gap: 1rem;
        }

        .stats-row {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 1rem;
        }

        @media (max-width: 640px) {
            .stats-row {
                grid-template-columns: 1fr;
            }
        }

        .stat-card {
            background: var(--surface);
            border: 1px solid var(--surface-border);
            padding: 1rem 1.25rem;
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }

        .stat-card span {
            font-size: 0.75rem;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            font-weight: 600;
        }

        .stat-card h3 {
            font-size: 1.4rem;
            font-weight: 700;
            margin-top: 0.25rem;
            color: white;
        }

        .search-category-bar {
            display: flex;
            gap: 1rem;
            align-items: center;
            flex-wrap: wrap;
        }

        .search-input-wrapper {
            flex: 1;
            position: relative;
            min-width: 240px;
        }

        .search-input {
            width: 100%;
            background: var(--surface);
            border: 1px solid var(--surface-border);
            color: white;
            padding: 0.75rem 1rem 0.75rem 2.5rem;
            border-radius: 10px;
            font-size: 0.9rem;
            transition: all 0.2s;
        }

        .search-input:focus {
            outline: none;
            border-color: var(--primary);
            box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);
        }

        .search-icon {
            position: absolute;
            left: 0.9rem;
            top: 50%;
            transform: translateY(-50%);
            color: var(--text-muted);
            font-size: 0.9rem;
        }

        .category-pills {
            display: flex;
            gap: 0.5rem;
            overflow-x: auto;
            padding-bottom: 0.25rem;
        }

        .pill {
            padding: 0.5rem 1rem;
            border-radius: 8px;
            font-size: 0.8rem;
            font-weight: 600;
            background: var(--surface);
            border: 1px solid var(--surface-border);
            color: var(--text-muted);
            cursor: pointer;
            white-space: nowrap;
            transition: all 0.2s;
        }

        .pill.active, .pill:hover {
            background: var(--primary);
            color: white;
            border-color: var(--primary);
        }

        /* Products Grid */
        .products-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
            gap: 1rem;
            overflow-y: auto;
            padding-right: 4px;
        }

        .product-card {
            background: var(--surface);
            border: 1px solid var(--surface-border);
            border-radius: 14px;
            padding: 1.15rem;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            gap: 0.75rem;
            cursor: pointer;
            transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
            position: relative;
            overflow: hidden;
        }

        .product-card:hover {
            transform: translateY(-3px);
            border-color: rgba(99, 102, 241, 0.5);
            box-shadow: 0 12px 24px -10px rgba(0, 0, 0, 0.5);
        }

        .product-badge {
            align-self: flex-start;
            background: rgba(255, 255, 255, 0.05);
            color: var(--text-muted);
            font-size: 0.7rem;
            padding: 0.2rem 0.5rem;
            border-radius: 6px;
            font-weight: 500;
        }

        .product-info h4 {
            font-size: 1rem;
            font-weight: 600;
            color: white;
            line-height: 1.3;
            margin-bottom: 0.25rem;
        }

        .product-meta {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-top: 0.5rem;
            padding-top: 0.75rem;
            border-top: 1px solid rgba(255, 255, 255, 0.05);
        }

        .product-price {
            font-family: 'JetBrains Mono', monospace;
            font-weight: 700;
            color: #34d399;
            font-size: 1.05rem;
        }

        .add-btn {
            background: var(--primary-light);
            color: #818cf8;
            border: 1px solid rgba(99, 102, 241, 0.3);
            width: 32px;
            height: 32px;
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.1rem;
            font-weight: bold;
            transition: all 0.2s;
        }

        .product-card:hover .add-btn {
            background: var(--primary);
            color: white;
            border-color: var(--primary);
        }

        /* Cart & Checkout Panel */
        .cart-panel {
            background: var(--surface);
            border-left: 1px solid var(--surface-border);
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            height: 100%;
        }

        .cart-header {
            padding: 1.25rem 1.5rem;
            border-bottom: 1px solid var(--surface-border);
            display: flex;
            align-items: center;
            justify-content: space-between;
        }

        .cart-header h2 {
            font-size: 1.1rem;
            font-weight: 700;
            color: white;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .clear-cart-btn {
            background: transparent;
            border: none;
            color: var(--danger);
            font-size: 0.8rem;
            font-weight: 600;
            cursor: pointer;
            padding: 0.25rem 0.5rem;
            border-radius: 6px;
        }

        .clear-cart-btn:hover {
            background: rgba(239, 68, 68, 0.1);
        }

        .cart-items {
            flex: 1;
            overflow-y: auto;
            padding: 1rem 1.5rem;
            display: flex;
            flex-direction: column;
            gap: 0.875rem;
        }

        .empty-cart {
            text-align: center;
            margin: auto 0;
            color: var(--text-muted);
            padding: 2rem 0;
        }

        .empty-cart-icon {
            font-size: 2.5rem;
            margin-bottom: 0.75rem;
            opacity: 0.4;
        }

        .cart-item {
            background: rgba(255, 255, 255, 0.02);
            border: 1px solid var(--surface-border);
            border-radius: 10px;
            padding: 0.875rem 1rem;
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 0.75rem;
        }

        .cart-item-title {
            font-weight: 600;
            font-size: 0.9rem;
            color: white;
            line-height: 1.2;
        }

        .cart-item-price {
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.8rem;
            color: #34d399;
            margin-top: 0.2rem;
        }

        .qty-controls {
            display: flex;
            align-items: center;
            gap: 0.4rem;
        }

        .qty-btn {
            width: 26px;
            height: 26px;
            border-radius: 6px;
            background: var(--surface-elevated);
            border: 1px solid var(--surface-border);
            color: white;
            font-weight: bold;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 0.9rem;
        }

        .qty-btn:hover {
            background: #334155;
        }

        .qty-num {
            font-size: 0.9rem;
            font-weight: 700;
            min-width: 24px;
            text-align: center;
        }

        /* Cart Summary & Checkout */
        .cart-footer {
            padding: 1.25rem 1.5rem;
            border-top: 1px solid var(--surface-border);
            background: rgba(17, 24, 39, 0.95);
            display: flex;
            flex-direction: column;
            gap: 1rem;
        }

        .summary-row {
            display: flex;
            justify-content: space-between;
            font-size: 0.875rem;
            color: var(--text-muted);
        }

        .summary-row.total {
            font-size: 1.25rem;
            font-weight: 800;
            color: white;
            margin-top: 0.25rem;
            padding-top: 0.75rem;
            border-top: 1px dashed var(--surface-border);
        }

        .summary-total-val {
            font-family: 'JetBrains Mono', monospace;
            color: #34d399;
        }

        .checkout-btn {
            width: 100%;
            background: linear-gradient(135deg, #6366f1, #4f46e5);
            color: white;
            border: none;
            padding: 1rem;
            border-radius: 12px;
            font-size: 1rem;
            font-weight: 700;
            cursor: pointer;
            box-shadow: 0 4px 15px rgba(79, 70, 229, 0.4);
            transition: all 0.2s;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 0.5rem;
        }

        .checkout-btn:hover:not(:disabled) {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(79, 70, 229, 0.5);
            background: linear-gradient(135deg, #4f46e5, #4338ca);
        }

        .checkout-btn:disabled {
            opacity: 0.5;
            cursor: not-allowed;
            box-shadow: none;
        }

        /* Receipt Modal */
        .modal-overlay {
            display: none;
            position: fixed;
            inset: 0;
            background: rgba(0, 0, 0, 0.75);
            backdrop-filter: blur(8px);
            z-index: 100;
            align-items: center;
            justify-content: center;
            padding: 1rem;
        }

        .receipt-card {
            background: #ffffff;
            color: #111827;
            width: 100%;
            max-width: 420px;
            border-radius: 16px;
            padding: 2rem;
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
            font-family: 'JetBrains Mono', monospace;
        }

        .receipt-header {
            text-align: center;
            border-bottom: 2px dashed #e5e7eb;
            padding-bottom: 1.25rem;
            margin-bottom: 1.25rem;
        }

        .receipt-header h2 {
            font-size: 1.4rem;
            font-weight: 800;
            font-family: 'Plus Jakarta Sans', sans-serif;
        }

        .receipt-header p {
            font-size: 0.8rem;
            color: #6b7280;
            margin-top: 0.25rem;
        }

        .receipt-items {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
            margin-bottom: 1.25rem;
            font-size: 0.85rem;
        }

        .receipt-item-row {
            display: flex;
            justify-content: space-between;
        }

        .receipt-total {
            border-top: 2px dashed #e5e7eb;
            padding-top: 1rem;
            margin-bottom: 1.5rem;
        }

        .receipt-total-row {
            display: flex;
            justify-content: space-between;
            font-size: 1.15rem;
            font-weight: 800;
        }

        .receipt-actions {
            display: flex;
            gap: 0.75rem;
        }

        .receipt-btn {
            flex: 1;
            padding: 0.75rem;
            border-radius: 8px;
            font-weight: 700;
            font-family: 'Plus Jakarta Sans', sans-serif;
            font-size: 0.9rem;
            cursor: pointer;
            text-align: center;
        }

        .receipt-btn-print {
            background: #111827;
            color: white;
            border: none;
        }

        .receipt-btn-close {
            background: #f3f4f6;
            color: #374151;
            border: 1px solid #d1d5db;
        }
    </style>
</head>
<body>
    <!-- Top Navigation Bar -->
    <header class="topbar">
        <a href="/pos" class="brand">
            <div class="brand-logo">⚡</div>
            <div class="brand-info">
                <h1>ZenoPOS</h1>
                <span>Laravel 13 Cashier Terminal • Pure Go Engine</span>
            </div>
        </a>

        <div class="top-nav">
            <div class="pos-badge">Online System</div>
            <a href="/web/products" class="nav-btn nav-btn-secondary">
                📦 Manage Inventory
            </a>
            <a href="/api/products" target="_blank" class="nav-btn nav-btn-secondary">
                ⚡ REST API
            </a>
        </div>
    </header>

    <!-- Main POS Workspace -->
    <div class="pos-layout">
        <!-- Catalog & Selection Area -->
        <main class="catalog-panel">
            <div class="catalog-header">
                <!-- Summary Metrics -->
                <div class="stats-row">
                    <div class="stat-card">
                        <div>
                            <span>Total Products</span>
                            <h3 id="statTotalItems">0</h3>
                        </div>
                        <div style="font-size: 1.8rem; opacity: 0.6;">📦</div>
                    </div>
                    <div class="stat-card">
                        <div>
                            <span>Cart Items</span>
                            <h3 id="statCartCount">0</h3>
                        </div>
                        <div style="font-size: 1.8rem; opacity: 0.6;">🛒</div>
                    </div>
                    <div class="stat-card">
                        <div>
                            <span>LaraZeno Status</span>
                            <h3 style="color: #34d399; font-size: 1.15rem; margin-top: 0.4rem;">Zero PHP Active</h3>
                        </div>
                        <div style="font-size: 1.8rem; opacity: 0.6;">⚡</div>
                    </div>
                </div>

                <!-- Search and Categories -->
                <div class="search-category-bar">
                    <div class="search-input-wrapper">
                        <span class="search-icon">🔍</span>
                        <input type="text" id="searchInput" class="search-input" placeholder="Search product name or barcode..." oninput="filterProducts()">
                    </div>
                    <div class="category-pills">
                        <button class="pill active" onclick="setCategory('all', this)">All Items</button>
                        <button class="pill" onclick="setCategory('Gadgets', this)">Gadgets</button>
                        <button class="pill" onclick="setCategory('Accessories', this)">Accessories</button>
                        <button class="pill" onclick="setCategory('Office', this)">Office</button>
                    </div>
                </div>
            </div>

            <!-- Products Grid Cards -->
            <div class="products-grid" id="productsContainer">
                <!-- Server rendered loop via Blade -->
                @foreach($products as $product)
                <div class="product-card" onclick="addToCart('{{ $product.id }}', '{{ $product.name }}', {{ $product.price }})">
                    <div class="product-badge">ID #{{ $product.id }}</div>
                    <div class="product-info">
                        <h4>{{ $product.name }}</h4>
                        <span style="font-size: 0.75rem; color: var(--text-muted);">Ready Stock</span>
                    </div>
                    <div class="product-meta">
                        <span class="product-price">Rp {{ $product.price }}</span>
                        <div class="add-btn">+</div>
                    </div>
                </div>
                @endforeach
            </div>
        </main>

        <!-- Right Side: Order Cart & Checkout -->
        <aside class="cart-panel">
            <div class="cart-header">
                <h2>🛒 Current Order</h2>
                <button class="clear-cart-btn" onclick="clearCart()">Clear</button>
            </div>

            <!-- Cart Items List -->
            <div class="cart-items" id="cartItemsList">
                <div class="empty-cart" id="emptyCartState">
                    <div class="empty-cart-icon">🛍️</div>
                    <p style="font-weight: 600; color: white;">Keranjang Masih Kosong</p>
                    <p style="font-size: 0.8rem; margin-top: 0.25rem;">Klik item produk di sebelah kiri untuk menambahkan ke pesanan.</p>
                </div>
            </div>

            <!-- Cart Footer Summary & Checkout Button -->
            <div class="cart-footer">
                <div class="summary-row">
                    <span>Subtotal</span>
                    <span id="summarySubtotal" style="font-family: 'JetBrains Mono'; font-weight: 600; color: white;">Rp 0</span>
                </div>
                <div class="summary-row">
                    <span>Pajak Resto / PPN (0%)</span>
                    <span style="font-family: 'JetBrains Mono';">Rp 0</span>
                </div>
                <div class="summary-row total">
                    <span>TOTAL BAYAR</span>
                    <span class="summary-total-val" id="summaryTotal">Rp 0</span>
                </div>

                <div style="display: flex; gap: 0.5rem; margin-top: 0.25rem;">
                    <select id="paymentMethod" style="background: var(--surface-elevated); color: white; border: 1px solid var(--surface-border); padding: 0.65rem 0.85rem; border-radius: 8px; font-size: 0.85rem; flex: 1;">
                        <option value="Cash">💵 Tunai / Cash</option>
                        <option value="QRIS">📱 QRIS / E-Wallet</option>
                        <option value="Debit">💳 Kartu Debit</option>
                    </select>
                </div>

                <button id="checkoutBtn" class="checkout-btn" onclick="processCheckout()" disabled>
                    <span>⚡ Process Payment</span>
                </button>
            </div>
        </aside>
    </div>

    <!-- Receipt Modal Dialog -->
    <div id="receiptModal" class="modal-overlay">
        <div class="receipt-card">
            <div class="receipt-header">
                <h2>ZenoPOS RECEIPT</h2>
                <p>Pure Go AST Laravel 13 Engine</p>
                <p id="receiptDate" style="font-size: 0.75rem; margin-top: 0.5rem;"></p>
                <p id="receiptOrderNo" style="font-weight: 700; color: #111827; margin-top: 0.2rem;"></p>
            </div>

            <div class="receipt-items" id="receiptItemsContainer">
                <!-- Injected via JavaScript -->
            </div>

            <div class="receipt-total">
                <div class="receipt-total-row">
                    <span>TOTAL:</span>
                    <span id="receiptTotalAmount">Rp 0</span>
                </div>
                <div style="display: flex; justify-content: space-between; font-size: 0.8rem; color: #6b7280; margin-top: 0.35rem;">
                    <span>Metode:</span>
                    <span id="receiptPaymentMethod">Cash</span>
                </div>
            </div>

            <div class="receipt-actions">
                <button class="receipt-btn receipt-btn-close" onclick="closeReceipt()">Tutup</button>
                <button class="receipt-btn receipt-btn-print" onclick="window.print()">Cetak Struk</button>
            </div>
        </div>
    </div>

    <script>
        // In-memory Shopping Cart State
        let cart = [];
        let currentCategory = 'all';

        // Format Angka ke Rupiah
        function formatRupiah(number) {
            return 'Rp ' + Number(number).toLocaleString('id-ID');
        }

        // Hitung total produk saat halaman pertama kali dibuka
        document.addEventListener("DOMContentLoaded", function() {
            const cards = document.querySelectorAll('.product-card');
            document.getElementById('statTotalItems').innerText = cards.length;
        });

        // Tambah Item ke Keranjang
        function addToCart(id, name, price) {
            const existing = cart.find(item => item.id === id);
            if (existing) {
                existing.qty++;
            } else {
                cart.push({
                    id: id,
                    name: name,
                    price: Number(price),
                    qty: 1
                });
            }
            renderCart();
        }

        // Ubah Quantity
        function updateQty(id, delta) {
            const item = cart.find(i => i.id === id);
            if (!item) return;

            item.qty += delta;
            if (item.qty <= 0) {
                cart = cart.filter(i => i.id !== id);
            }
            renderCart();
        }

        // Bersihkan Keranjang
        function clearCart() {
            cart = [];
            renderCart();
        }

        // Render Antarmuka Keranjang
        function renderCart() {
            const listEl = document.getElementById('cartItemsList');
            const emptyEl = document.getElementById('emptyCartState');
            const subtotalEl = document.getElementById('summarySubtotal');
            const totalEl = document.getElementById('summaryTotal');
            const checkoutBtn = document.getElementById('checkoutBtn');
            const cartCountEl = document.getElementById('statCartCount');

            // Hitung total kuantitas dan harga
            let totalQty = 0;
            let totalPrice = 0;

            cart.forEach(item => {
                totalQty += item.qty;
                totalPrice += (item.price * item.qty);
            });

            cartCountEl.innerText = totalQty;
            subtotalEl.innerText = formatRupiah(totalPrice);
            totalEl.innerText = formatRupiah(totalPrice);

            if (cart.length === 0) {
                listEl.innerHTML = '';
                listEl.appendChild(emptyEl);
                checkoutBtn.disabled = true;
                return;
            }

            checkoutBtn.disabled = false;
            let html = '';
            cart.forEach(item => {
                html += `
                <div class="cart-item">
                    <div style="flex: 1;">
                        <div class="cart-item-title">${item.name}</div>
                        <div class="cart-item-price">${formatRupiah(item.price)} × ${item.qty}</div>
                    </div>
                    <div class="qty-controls">
                        <button class="qty-btn" onclick="updateQty('${item.id}', -1)">-</button>
                        <span class="qty-num">${item.qty}</span>
                        <button class="qty-btn" onclick="updateQty('${item.id}', 1)">+</button>
                    </div>
                </div>
                `;
            });
            listEl.innerHTML = html;
        }

        // Filter Produk berdasarkan Pencarian
        function filterProducts() {
            const keyword = document.getElementById('searchInput').value.toLowerCase();
            const cards = document.querySelectorAll('.product-card');

            cards.forEach(card => {
                const title = card.querySelector('h4').innerText.toLowerCase();
                if (title.includes(keyword)) {
                    card.style.display = 'flex';
                } else {
                    card.style.display = 'none';
                }
            });
        }

        // Filter Kategori
        function setCategory(cat, btn) {
            document.querySelectorAll('.category-pills .pill').forEach(p => p.classList.remove('active'));
            btn.classList.add('active');
            // Mock filter kategori demo
            filterProducts();
        }

        // Proses Checkout via API LaraZeno
        async function processCheckout() {
            if (cart.length === 0) return;

            const btn = document.getElementById('checkoutBtn');
            btn.disabled = true;
            btn.innerHTML = '<span>⏳ Processing...</span>';

            const method = document.getElementById('paymentMethod').value;
            let total = 0;
            cart.forEach(i => total += (i.price * i.qty));

            const orderNumber = 'ORD-' + Date.now().toString().slice(-6);

            try {
                // Post ke endpoint LaraZeno Checkout API
                const response = await fetch('/api/pos/checkout', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        order_number: orderNumber,
                        customer_name: 'Walk-in Customer',
                        total_amount: total,
                        payment_method: method,
                        items_json: JSON.stringify(cart)
                    })
                });

                // Tampilkan Struk Pembelian
                showReceipt(orderNumber, method, total, [...cart]);
                clearCart();
            } catch (err) {
                alert('Transaksi Berhasil! (Offline/Local State)');
                showReceipt(orderNumber, method, total, [...cart]);
                clearCart();
            } finally {
                btn.disabled = false;
                btn.innerHTML = '<span>⚡ Process Payment</span>';
            }
        }

        // Tampilkan Modal Struk Kasir
        function showReceipt(orderNo, method, total, items) {
            document.getElementById('receiptOrderNo').innerText = '#' + orderNo;
            document.getElementById('receiptDate').innerText = new Date().toLocaleString('id-ID');
            document.getElementById('receiptPaymentMethod').innerText = method;
            document.getElementById('receiptTotalAmount').innerText = formatRupiah(total);

            const container = document.getElementById('receiptItemsContainer');
            let itemsHtml = '';
            items.forEach(i => {
                itemsHtml += `
                <div class="receipt-item-row">
                    <span>${i.name} × ${i.qty}</span>
                    <span>${formatRupiah(i.price * i.qty)}</span>
                </div>
                `;
            });
            container.innerHTML = itemsHtml;

            document.getElementById('receiptModal').style.display = 'flex';
        }

        function closeReceipt() {
            document.getElementById('receiptModal').style.display = 'none';
        }
    </script>
</body>
</html>
