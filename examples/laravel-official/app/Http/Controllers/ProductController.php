<?php

namespace App\Http\Controllers;

use App\Models\Product;
use Illuminate\Http\Request;

class ProductController extends Controller
{
    /**
     * Web Blade View: Display product catalog page
     */
    public function webIndex()
    {
        $products = Product::all();
        return view('products.index', $products);
    }

    /**
     * Web Blade Action: Store a new product and redirect
     */
    public function webStore()
    {
        $product = Product::create();
        return redirect('/web/products');
    }

    /**
     * Web Blade Action: Update a product and redirect
     */
    public function webUpdate($id)
    {
        $product = Product::find($id);
        $product->update();
        return redirect('/web/products');
    }

    /**
     * Web Blade Action: Delete a product and redirect
     */
    public function webDestroy($id)
    {
        $product = Product::find($id);
        $product->delete();
        return redirect('/web/products');
    }

    /**
     * REST API: Display a listing of the resource.
     */
    public function index()
    {
        return response()->json(Product::all());
    }

    /**
     * REST API: Store a newly created resource in storage.
     */
    public function store(Request $request)
    {
        $product = Product::create();
        return response()->json($product, 201);
    }

    /**
     * REST API: Display the specified resource.
     */
    public function show($id)
    {
        $product = Product::find($id);
        return response()->json($product);
    }

    /**
     * REST API: Update the specified resource in storage.
     */
    public function update(Request $request, $id)
    {
        $product = Product::find($id);
        $product->update();
        return response()->json($product);
    }

    /**
     * REST API: Remove the specified resource from storage.
     */
    public function destroy($id)
    {
        $product = Product::find($id);
        $product->delete();
        return response()->json([
            "message" => "Product deleted successfully"
        ]);
    }
}
