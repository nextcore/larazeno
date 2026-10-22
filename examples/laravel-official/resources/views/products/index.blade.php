<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Products Manager - LaraZeno</title>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-body: #0b0f19;
            --surface: #111827;
            --surface-border: #1f2937;
            --primary: #6366f1;
            --primary-hover: #4f46e5;
            --text-main: #f9fafb;
            --text-muted: #9ca3af;
            --danger: #ef4444;
            --danger-hover: #dc2626;
            --warning: #f59e0b;
            --warning-hover: #d97706;
            --success: #10b981;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Plus Jakarta Sans', sans-serif;
        }

        body {
            background-color: var(--bg-body);
            color: var(--text-main);
            min-height: 100vh;
            padding: 2.5rem 1rem;
            display: flex;
            justify-content: center;
        }

        .container {
            width: 100%;
            max-width: 900px;
        }

        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
            border-bottom: 1px solid var(--surface-border);
            padding-bottom: 1.5rem;
        }

        .header-title h1 {
            font-size: 1.875rem;
            font-weight: 700;
            background: linear-gradient(135deg, #a5b4fc 0%, #6366f1 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .header-title p {
            color: var(--text-muted);
            font-size: 0.875rem;
            margin-top: 0.25rem;
        }

        .badge {
            background: rgba(99, 102, 241, 0.1);
            color: #818cf8;
            border: 1px solid rgba(99, 102, 241, 0.2);
            padding: 0.35rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 600;
        }

        .card {
            background: var(--surface);
            border: 1px solid var(--surface-border);
            border-radius: 1rem;
            padding: 1.75rem;
            box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.4);
            margin-bottom: 2rem;
        }

        .form-grid {
            display: grid;
            grid-template-columns: 2fr 1fr auto auto;
            gap: 1rem;
            align-items: flex-end;
        }

        @media (max-width: 640px) {
            .form-grid {
                grid-template-columns: 1fr;
            }
        }

        .form-group {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        label {
            font-size: 0.8125rem;
            font-weight: 500;
            color: var(--text-muted);
        }

        input {
            background: #1e293b;
            border: 1px solid #334155;
            color: #fff;
            padding: 0.75rem 1rem;
            border-radius: 0.5rem;
            font-size: 0.875rem;
            transition: all 0.2s ease;
        }

        input:focus {
            outline: none;
            border-color: var(--primary);
            box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);
        }

        .btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            padding: 0.75rem 1.25rem;
            border-radius: 0.5rem;
            font-size: 0.875rem;
            font-weight: 600;
            cursor: pointer;
            border: none;
            transition: all 0.2s;
            text-decoration: none;
        }

        .btn-primary {
            background: var(--primary);
            color: white;
        }

        .btn-primary:hover {
            background: var(--primary-hover);
            transform: translateY(-1px);
        }

        .btn-warning {
            background: rgba(245, 158, 11, 0.1);
            color: #fbbf24;
            border: 1px solid rgba(245, 158, 11, 0.25);
            padding: 0.4rem 0.8rem;
            font-size: 0.75rem;
            margin-right: 0.35rem;
        }

        .btn-warning:hover {
            background: var(--warning);
            color: white;
        }

        .btn-danger {
            background: rgba(239, 68, 68, 0.1);
            color: #f87171;
            border: 1px solid rgba(239, 68, 68, 0.2);
            padding: 0.4rem 0.8rem;
            font-size: 0.75rem;
        }

        .btn-danger:hover {
            background: var(--danger);
            color: white;
        }

        .btn-cancel {
            background: #374151;
            color: #d1d5db;
        }

        .btn-cancel:hover {
            background: #4b5563;
        }

        .table-responsive {
            overflow-x: auto;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            text-align: left;
        }

        th {
            padding: 0.875rem 1rem;
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--text-muted);
            border-bottom: 1px solid var(--surface-border);
        }

        td {
            padding: 1rem;
            font-size: 0.875rem;
            border-bottom: 1px solid rgba(255, 255, 255, 0.04);
            vertical-align: middle;
        }

        tr:hover td {
            background: rgba(255, 255, 255, 0.015);
        }

        .price-tag {
            font-weight: 600;
            color: #34d399;
        }

        /* Modal Styles */
        .modal-overlay {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.7);
            backdrop-filter: blur(4px);
            z-index: 50;
            justify-content: center;
            align-items: center;
            padding: 1rem;
        }

        .modal-content {
            background: var(--surface);
            border: 1px solid var(--surface-border);
            border-radius: 1rem;
            width: 100%;
            max-width: 500px;
            padding: 2rem;
            box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5);
        }

        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1.5rem;
        }

        .modal-header h3 {
            font-size: 1.25rem;
            font-weight: 600;
        }

        .close-btn {
            background: transparent;
            border: none;
            color: var(--text-muted);
            font-size: 1.5rem;
            cursor: pointer;
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Header -->
        <header class="header">
            <div class="header-title">
                <h1>Products Manager</h1>
                <p>Pure Go AST Laravel Execution Engine</p>
            </div>
            <div>
                <span class="badge">Laravel 13 Blade View</span>
            </div>
        </header>

        <!-- Form Tambah Produk -->
        <div class="card" id="formCard">
            <h2 id="formTitle" style="font-size: 1.125rem; font-weight: 600; margin-bottom: 1.25rem;">Add New Product</h2>
            <form id="productForm" action="/web/products" method="POST" class="form-grid">
                @csrf
                <input type="hidden" id="methodField" name="_method" value="" disabled>
                <div class="form-group">
                    <label for="name">Product Name</label>
                    <input type="text" id="name" name="name" placeholder="e.g. Mechanical Keyboard RGB" required>
                </div>
                <div class="form-group">
                    <label for="price">Price (IDR)</label>
                    <input type="number" id="price" name="price" placeholder="1500000" required>
                </div>
                <button type="submit" id="submitBtn" class="btn btn-primary">+ Save Product</button>
                <button type="button" id="cancelEditBtn" class="btn btn-cancel" style="display: none;" onclick="cancelEdit()">Cancel</button>
            </form>
        </div>

        <!-- Tabel Daftar Produk -->
        <div class="card" style="padding: 0; overflow: hidden;">
            <div style="padding: 1.25rem 1.5rem; border-bottom: 1px solid var(--surface-border);">
                <h2 style="font-size: 1.125rem; font-weight: 600;">Product Catalog</h2>
            </div>
            <div class="table-responsive">
                <table>
                    <thead>
                        <tr>
                            <th style="width: 80px;">ID</th>
                            <th>Product Name</th>
                            <th>Price</th>
                            <th style="width: 160px; text-align: right;">Action</th>
                        </tr>
                    </thead>
                    <tbody>
                        @foreach($products as $product)
                        <tr>
                            <td style="color: var(--text-muted);">#{{ $product['id'] }}</td>
                            <td style="font-weight: 500;" id="prod-name-{{ $product['id'] }}">{{ $product['name'] }}</td>
                            <td class="price-tag" id="prod-price-{{ $product['id'] }}" data-raw-price="{{ $product['price'] }}">Rp {{ $product['price'] }}</td>
                            <td style="text-align: right;">
                                <button type="button" class="btn btn-warning" onclick="editProduct('{{ $product['id'] }}', '{{ $product['name'] }}', '{{ $product['price'] }}')">Edit</button>
                                <form action="/web/products/{{ $product['id'] }}" method="POST" style="display: inline-block;">
                                    @csrf
                                    @method('DELETE')
                                    <input type="hidden" name="_method" value="DELETE">
                                    <button type="submit" class="btn btn-danger" onclick="return confirm('Hapus produk ini?')">Delete</button>
                                </form>
                            </td>
                        </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <!-- Edit Modal Dialog (Opsional / Quick Switch) -->
    <div id="editModal" class="modal-overlay">
        <div class="modal-content">
            <div class="modal-header">
                <h3>Edit Product</h3>
                <button class="close-btn" onclick="closeModal()">&times;</button>
            </div>
            <form id="modalForm" method="POST" style="display: flex; flex-direction: column; gap: 1.25rem;">
                @csrf
                <input type="hidden" name="_method" value="PUT">
                <div class="form-group">
                    <label for="modalName">Product Name</label>
                    <input type="text" id="modalName" name="name" required>
                </div>
                <div class="form-group">
                    <label for="modalPrice">Price (IDR)</label>
                    <input type="number" id="modalPrice" name="price" required>
                </div>
                <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
                    <button type="button" class="btn btn-cancel" onclick="closeModal()">Cancel</button>
                    <button type="submit" class="btn btn-primary">Update Changes</button>
                </div>
            </form>
        </div>
    </div>

    <script>
        function editProduct(id, name, price) {
            // Isi data ke modal dialog
            const form = document.getElementById('modalForm');
            form.action = '/web/products/' + id;
            document.getElementById('modalName').value = name;
            document.getElementById('modalPrice').value = price;
            document.getElementById('editModal').style.display = 'flex';
        }

        function closeModal() {
            document.getElementById('editModal').style.display = 'none';
        }

        // Close modal on escape key
        window.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                closeModal();
            }
        });

        // Close on clicking overlay
        document.getElementById('editModal').addEventListener('click', function(e) {
            if (e.target === this) {
                closeModal();
            }
        });
    </script>
</body>
</html>
