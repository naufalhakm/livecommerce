import React from 'react';
import { useNavigate } from 'react-router-dom';
import { User, Video, Check, Tv } from 'lucide-react';

const Home = () => {
  const navigate = useNavigate();

  return (
    <div className="min-h-screen bg-gray-900 flex items-center justify-center p-4">
      <div className="max-w-4xl w-full">
        <div className="text-center mb-12">
          <div className="flex items-center justify-center gap-3 mb-4">
            <div className="w-12 h-12 bg-red-500 rounded-lg flex items-center justify-center">
              <Tv className="w-6 h-6 text-white" />
            </div>
            <h1 className="text-white text-4xl font-bold">LiveShop AI</h1>
          </div>
          <p className="text-gray-400 text-lg">Real-time product recognition for live streaming commerce</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Viewer Card */}
          <div 
            onClick={() => navigate('/streams')}
            className="bg-gray-800 rounded-2xl p-8 border-2 border-gray-700 hover:border-red-500 cursor-pointer transition-all hover:scale-105"
          >
            <div className="text-center">
              <div className="w-20 h-20 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                <User className="w-10 h-10 text-red-500" />
              </div>
              <h2 className="text-white text-2xl font-bold mb-2">Join as Viewer</h2>
              <p className="text-gray-400 mb-6">Browse active live streams and join shopping sessions</p>
              <div className="space-y-2 text-left text-sm text-gray-400">
                <div className="flex items-center gap-2">
                  <Check className="w-4 h-4 text-green-500" />
                  <span>Watch live product showcases</span>
                </div>
                <div className="flex items-center gap-2">
                  <Check className="w-4 h-4 text-green-500" />
                  <span>Real-time chat with sellers</span>
                </div>
                <div className="flex items-center gap-2">
                  <Check className="w-4 h-4 text-green-500" />
                  <span>Add products to cart instantly</span>
                </div>
              </div>
            </div>
          </div>

          {/* Seller Card */}
          <div 
            onClick={() => navigate('/seller')}
            className="bg-gray-800 rounded-2xl p-8 border-2 border-gray-700 hover:border-red-500 cursor-pointer transition-all hover:scale-105"
          >
            <div className="text-center">
              <div className="w-20 h-20 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                <Video className="w-10 h-10 text-red-500" />
              </div>
              <h2 className="text-white text-2xl font-bold mb-2">Start as Seller</h2>
              <p className="text-gray-400 mb-6">Go live and showcase your products with AI assistance</p>
              <div className="space-y-2 text-left text-sm text-gray-400">
                <div className="flex items-center gap-2">
                  <Check className="w-4 h-4 text-green-500" />
                  <span>AI-powered product detection</span>
                </div>
                <div className="flex items-center gap-2">
                  <Check className="w-4 h-4 text-green-500" />
                  <span>Real-time sales analytics</span>
                </div>
                <div className="flex items-center gap-2">
                  <Check className="w-4 h-4 text-green-500" />
                  <span>Manage products during stream</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="text-center mt-8">
          <button 
            onClick={() => navigate('/admin')}
            className="text-gray-400 hover:text-white text-sm"
          >
            Admin Dashboard →
          </button>
        </div>
      </div>
    </div>
  );
};

export default Home;