<?php

use Illuminate\Support\Facades\Route;
use App\Http\Controllers\ProductController;

Route::get('/', function () {
    return view('welcome');
});

Route::get('/web/products', [ProductController::class, 'webIndex']);
Route::post('/web/products', [ProductController::class, 'webStore']);
Route::delete('/web/products/{id}', [ProductController::class, 'webDestroy']);
