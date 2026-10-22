<?php

use Illuminate\Support\Facades\Route;
use App\Http\Controllers\ProductController;
use App\Http\Controllers\PosController;

Route::get('/', function () {
    return redirect('/pos');
});

// POS Cashier Terminal & Transactions
Route::get('/pos', [PosController::class, 'index']);
Route::post('/api/pos/checkout', [PosController::class, 'checkout']);
Route::get('/api/pos/orders', [PosController::class, 'orders']);

// Product Management (Blade UI & Actions)
Route::get('/web/products', [ProductController::class, 'webIndex']);
Route::post('/web/products', [ProductController::class, 'webStore']);
Route::put('/web/products/{id}', [ProductController::class, 'webUpdate']);
Route::delete('/web/products/{id}', [ProductController::class, 'webDestroy']);

