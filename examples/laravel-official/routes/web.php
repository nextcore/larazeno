<?php

use Illuminate\Support\Facades\Route;
use App\Http\Controllers\ProductController;

Route::get('/', function () {
    return view('welcome');
});

Route::get('/web/products', [ProductController::class, 'webIndex']);
Route::post('/web/products', [ProductController::class, 'webStore']);
Route::put('/web/products/{id}', [ProductController::class, 'webUpdate']);
Route::delete('/web/products/{id}', [ProductController::class, 'webDestroy']);
