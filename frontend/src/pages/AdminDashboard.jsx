import React, { useState, useEffect } from 'react';
import { productAPI } from '../services/api';
import { LayoutDashboard, Package, Tv, FileText, TrendingUp, Settings, LogOut, Plus, Search, ChevronDown, Target, Edit, Trash2, Box } from 'lucide-react';

const AdminDashboard = () => {
  const [products, setProducts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [trainingStatus, setTrainingStatus] = useState({});
  const [isTraining, setIsTraining] = useState(false);

  useEffect(() => {
    loadProducts();
  }, []);

  const loadProducts = async () => {
    try {
      const response = await productAPI.getAll();
      setProducts(response.data);
    } catch (error) {
      console.error('Error loading products:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteProduct = async (productId) => {
    if (window.confirm('Are you sure you want to delete this product?')) {
      try {
        await productAPI.delete(productId);
        setProducts(products.filter(p => p.id !== productId));
      } catch (error) {
        console.error('Error deleting product:', error);
        alert('Failed to delete product');
      }
    }
  };

  const handleTrainModel = async (productId, sellerId) => {
    try {
      await productAPI.train(productId);
      alert(`Model training started for seller ${sellerId}`);
    } catch (error) {
      console.error('Error training model:', error);
      alert('Failed to start training');
    }
  };

  const handleTrainAllProducts = async () => {
    if (!window.confirm('Train ML model for all products? This will take a few minutes.')) {
      return;
    }
    
    setIsTraining(true);
    
    try {
      // Get unique seller IDs from products
      const sellerIds = [...new Set(products.map(p => p.seller_id))];
      
      for (const sellerId of sellerIds) {
        const response = await fetch(`http://localhost:7001/train?seller_id=seller_${sellerId}`, {
          method: 'POST'
        });
        
        if (response.ok) {
          console.log(`Training started for seller ${sellerId}`);
        }
      }
      
      // Start polling for progress
      pollTrainingProgress(sellerIds);
      
    } catch (error) {
      console.error('Error training models:', error);
      alert('Failed to start training');
      setIsTraining(false);
    }
  };

  const pollTrainingProgress = (sellerIds) => {
    const interval = setInterval(async () => {
      try {
        const statusPromises = sellerIds.map(async (sellerId) => {
          const response = await fetch(`http://localhost:7001/training-status/seller_${sellerId}`);
          const status = await response.json();
          return { sellerId, status };
        });
        
        const statuses = await Promise.all(statusPromises);
        const statusMap = {};
        
        statuses.forEach(({ sellerId, status }) => {
          statusMap[sellerId] = status;
        });
        
        setTrainingStatus(statusMap);
        
        // Check if all training is complete
        const allComplete = statuses.every(({ status }) => 
          status.status === 'completed' || status.status === 'error'
        );
        
        if (allComplete) {
          clearInterval(interval);
          setIsTraining(false);
          alert('Training completed for all sellers!');
        }
      } catch (error) {
        console.error('Error polling training status:', error);
      }
    }, 2000); // Poll every 2 seconds
  };

  const filteredProducts = products?.filter(p => 
    p.name.toLowerCase().includes(searchQuery.toLowerCase())
  ) || [];

  if (loading) {
    return <div className="flex justify-center items-center h-screen dark:bg-gray-900 dark:text-white">Loading...</div>;
  }

  return (
    <div className="flex min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Sidebar */}
      <aside className="w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700">
        <div className="flex flex-col h-full p-4">
          <div className="flex flex-col gap-4">
            {/* Profile */}
            <div className="flex items-center gap-3 p-2">
              <div className="w-10 h-10 bg-red-500 rounded-full flex items-center justify-center text-white font-bold">
                S
              </div>
              <div>
                <h1 className="text-gray-900 dark:text-white text-base font-semibold">The Seller Shop</h1>
                <p className="text-gray-500 dark:text-gray-400 text-sm">seller@email.com</p>
              </div>
            </div>

            {/* Navigation */}
            <nav className="flex flex-col gap-1 mt-4">
              <a href="/" className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700">
                <LayoutDashboard className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                <p className="text-gray-700 dark:text-white text-sm font-medium">Dashboard</p>
              </a>
              <a href="/admin" className="flex items-center gap-3 px-3 py-2 rounded-lg bg-red-50 dark:bg-gray-700">
                <Package className="w-5 h-5 text-red-500 dark:text-white" />
                <p className="text-red-500 dark:text-white text-sm font-semibold">Product Management</p>
              </a>
              <a href="/seller" className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700">
                <Tv className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                <p className="text-gray-700 dark:text-white text-sm font-medium">Live Streams</p>
              </a>
              <a href="#" className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700">
                <FileText className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                <p className="text-gray-700 dark:text-white text-sm font-medium">Orders</p>
              </a>
              <a href="#" className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700">
                <TrendingUp className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                <p className="text-gray-700 dark:text-white text-sm font-medium">Analytics</p>
              </a>
            </nav>
          </div>

          <div className="mt-auto flex flex-col gap-2">
            <a href="/seller" className="flex items-center justify-center gap-2 h-10 px-4 bg-red-500 text-white text-sm font-bold rounded-lg hover:bg-red-600">
              <Tv className="w-4 h-4" />
              <span>Start a Livestream</span>
            </a>
            <div className="border-t border-gray-200 dark:border-gray-700 my-2"></div>
            <a href="#" className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700">
              <Settings className="w-5 h-5 text-gray-600 dark:text-gray-400" />
              <p className="text-gray-700 dark:text-white text-sm font-medium">Settings</p>
            </a>
            <a href="/" className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700">
              <LogOut className="w-5 h-5 text-gray-600 dark:text-gray-400" />
              <p className="text-gray-700 dark:text-white text-sm font-medium">Logout</p>
            </a>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 p-8">
        {/* Header */}
        <div className="flex justify-between items-start mb-6">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">Product Management</h1>
            <p className="text-gray-500 dark:text-gray-400">View, add, edit, and remove products for your live commerce streams.</p>
          </div>
          <div className="flex gap-3">
            <button 
              onClick={handleTrainAllProducts}
              disabled={isTraining}
              className={`flex items-center gap-2 h-10 px-4 text-white text-sm font-bold rounded-lg ${
                isTraining ? 'bg-gray-500 cursor-not-allowed' : 'bg-blue-500 hover:bg-blue-600'
              }`}
            >
              <Target className="w-4 h-4" />
              <span>{isTraining ? 'Training...' : 'Train All Products'}</span>
            </button>
            <button 
              onClick={() => window.location.href = '/admin/products/create'}
              className="flex items-center gap-2 h-10 px-4 bg-red-500 text-white text-sm font-bold rounded-lg hover:bg-red-600"
            >
              <Plus className="w-4 h-4" />
              <span>Add New Product</span>
            </button>
          </div>
        </div>

        {/* Training Progress */}
        {isTraining && Object.keys(trainingStatus).length > 0 && (
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 mb-6">
            <h3 className="text-blue-800 dark:text-blue-200 font-semibold mb-3">Training Progress</h3>
            <div className="space-y-2">
              {Object.entries(trainingStatus).map(([sellerId, status]) => (
                <div key={sellerId} className="flex items-center gap-3">
                  <span className="text-sm font-medium text-blue-700 dark:text-blue-300">Seller {sellerId}:</span>
                  <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                    <div 
                      className={`h-2 rounded-full transition-all duration-300 ${
                        status.status === 'completed' ? 'bg-green-500' : 
                        status.status === 'error' ? 'bg-red-500' : 'bg-blue-500'
                      }`}
                      style={{ width: `${status.progress || 0}%` }}
                    ></div>
                  </div>
                  <span className="text-xs text-gray-600 dark:text-gray-400">{status.message}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Search & Filters */}
        <div className="flex gap-4 mb-6">
          <div className="flex-1">
            <div className="relative">
              <input
                type="text"
                placeholder="Search by product name, SKU..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full h-12 pl-12 pr-4 bg-gray-100 dark:bg-gray-800 border-0 rounded-lg text-gray-900 dark:text-white placeholder:text-gray-500 dark:placeholder:text-gray-400 focus:ring-2 focus:ring-red-500"
              />
              <Search className="absolute left-4 top-3.5 w-5 h-5 text-gray-400" />
            </div>
          </div>
          <button className="flex items-center gap-2 h-12 px-4 bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-white rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700">
            <span className="text-sm font-medium">Filter by Category</span>
            <ChevronDown className="w-4 h-4" />
          </button>
          <button className="flex items-center gap-2 h-12 px-4 bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-white rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700">
            <span className="text-sm font-medium">Filter by Status</span>
            <ChevronDown className="w-4 h-4" />
          </button>
        </div>

        {/* Table */}
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50 dark:bg-gray-900">
              <tr>
                <th className="px-6 py-4 text-left w-12">
                  <input type="checkbox" className="w-5 h-5 rounded border-gray-300 dark:border-gray-600" />
                </th>
                <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase">Product</th>
                <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase">Images</th>
                <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase">Price</th>
                <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase">Seller ID</th>
                <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {filteredProducts.map((product) => (
                <tr key={product.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                  <td className="px-6 py-4">
                    <input type="checkbox" className="w-5 h-5 rounded border-gray-300 dark:border-gray-600" />
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-4">
                      {product.images && product.images.length > 0 ? (
                        <img src={product.images[0].image_url} alt={product.name} className="w-10 h-10 rounded-md object-cover" />
                      ) : (
                        <div className="w-10 h-10 bg-gray-200 dark:bg-gray-700 rounded-md flex items-center justify-center">
                          <Box className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                        </div>
                      )}
                      <div>
                        <div className="font-medium text-gray-900 dark:text-white">{product.name}</div>
                        <div className="text-sm text-gray-500 dark:text-gray-400">{product.description}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-1">
                      <span className="text-sm font-medium text-gray-900 dark:text-white">
                        {product.images ? product.images.length : 0}
                      </span>
                      <span className="text-xs text-gray-500 dark:text-gray-400">images</span>
                    </div>
                  </td>
                  <td className="px-6 py-4 text-sm font-medium text-gray-900 dark:text-white">${product.price}</td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">{product.seller_id}</td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => handleTrainModel(product.id, product.seller_id)}
                        className="p-2 rounded-md hover:bg-blue-100 dark:hover:bg-blue-900/30 text-blue-600 dark:text-blue-400"
                        title="Train Model"
                      >
                        <Target className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => window.location.href = `/admin/products/edit/${product.id}`}
                        className="p-2 rounded-md hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400"
                        title="Edit"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleDeleteProduct(product.id)}
                        className="p-2 rounded-md hover:bg-red-100 dark:hover:bg-red-900/30 text-red-500 dark:text-red-400"
                        title="Delete"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {filteredProducts.length === 0 && (
                <tr>
                  <td colSpan="7" className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
                    No products found. Add your first product to get started.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between mt-6">
          <div className="text-sm text-gray-500 dark:text-gray-400">
            Showing <span className="font-semibold text-gray-700 dark:text-white">{filteredProducts.length}</span> results
          </div>
          <div className="flex items-center gap-2">
            <button className="flex items-center justify-center h-9 w-9 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white hover:bg-gray-100 dark:hover:bg-gray-700">
              ‹
            </button>
            <button className="flex items-center justify-center h-9 w-9 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white hover:bg-gray-100 dark:hover:bg-gray-700">
              ›
            </button>
          </div>
        </div>
      </main>
    </div>
  );
};

export default AdminDashboard;