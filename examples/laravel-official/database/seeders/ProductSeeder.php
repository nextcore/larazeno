<?php

namespace Database\Seeders;

use App\Models\Product;
use Illuminate\Database\Seeder;

class ProductSeeder extends Seeder
{
    /**
     * Run the database seeds for ZenoPOS.
     */
    public function run(): void
    {
        Product::create([
            'name' => 'Logitech Wireless Mouse M185',
            'price' => 165000,
            'image' => 'https://images.unsplash.com/photo-1527864550417-7fd91fc51a46?w=400&q=80',
        ]);

        Product::create([
            'name' => 'Mechanical Keyboard RGB TKL',
            'price' => 450000,
            'image' => 'https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=400&q=80',
        ]);

        Product::create([
            'name' => 'USB-C Multifunction Hub 7-in-1',
            'price' => 285000,
            'image' => 'https://images.unsplash.com/photo-1544652478-6653e09f18a2?w=400&q=80',
        ]);

        Product::create([
            'name' => 'Ergonomic Desk Mousepad XL',
            'price' => 85000,
            'image' => 'https://images.unsplash.com/photo-1616401784845-180882ba9ba8?w=400&q=80',
        ]);

        Product::create([
            'name' => 'USB Condenser Microphone RGB',
            'price' => 320000,
            'image' => 'https://images.unsplash.com/photo-1590658268037-6bf12165a8df?w=400&q=80',
        ]);

        Product::create([
            'name' => 'Monitor Light Bar LED Dimmer',
            'price' => 245000,
            'image' => 'https://images.unsplash.com/photo-1507473885765-e6ed057f782c?w=400&q=80',
        ]);
    }
}
