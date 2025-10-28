import React from 'react';

function App() {
  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">
          Live Shopping AI
        </h1>
        <p className="text-lg text-gray-600 mb-8">
          System is loading...
        </p>
        <div className="space-y-4">
          <a href="/admin" className="block bg-blue-500 text-white px-6 py-3 rounded-lg hover:bg-blue-600">
            Admin Dashboard
          </a>
          <a href="/viewer" className="block bg-green-500 text-white px-6 py-3 rounded-lg hover:bg-green-600">
            Live Stream Viewer
          </a>
          <a href="/seller" className="block bg-purple-500 text-white px-6 py-3 rounded-lg hover:bg-purple-600">
            Live Stream Seller
          </a>
        </div>
      </div>
    </div>
  );
}

export default App;