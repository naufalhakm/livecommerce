import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Home from './pages/Home';
import AdminDashboard from './pages/AdminDashboard';
import LiveStreamViewer from './pages/LiveStreamViewer';
import LiveStreamSeller from './pages/LiveStreamSeller';
import CreateProduct from './pages/CreateProduct';
import EditProduct from './pages/EditProduct';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/viewer" element={<LiveStreamViewer />} />
        <Route path="/seller" element={<LiveStreamSeller />} />
        <Route path="/admin" element={<AdminDashboard />} />
        <Route path="/admin/products/create" element={<CreateProduct />} />
        <Route path="/admin/products/edit/:id" element={<EditProduct />} />
      </Routes>
    </Router>
  );
}

export default App;