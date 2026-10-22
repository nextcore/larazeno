<?php

namespace App\Http\Controllers;

use App\Models\Order;
use App\Models\Product;
use Illuminate\Http\Request;

class PosController extends Controller
{
    /**
     * Tampilan Utama POS Cashier Terminal
     */
    public function index()
    {
        $products = Product::all();
        $orders = Order::all();
        return view('pos.index', $products);
    }

    /**
     * Endpoint API untuk memproses transaksi checkout POS
     */
    public function checkout()
    {
        $order = Order::create();
        return response()->json($order);
    }

    /**
     * Endpoint API daftar transaksi penjualan
     */
    public function orders()
    {
        $orders = Order::all();
        return response()->json($orders);
    }
}
