import React, { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { productAPI, streamAPI, pinAPI } from '../services/api';
import websocketService from '../services/websocket';
import webrtcService from '../services/webrtc';
import { LayoutDashboard, Package, Tv, FileText, TrendingUp, Settings, LogOut, Video, Eye, Plus, Pin, Send, RotateCcw } from 'lucide-react';

const LiveStreamSeller = () => {
  const [sellerId, setSellerId] = useState('');
  const [hasStarted, setHasStarted] = useState(false);
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamEnded, setStreamEnded] = useState(false);
  const [products, setProducts] = useState([]);
  const [pinnedProduct, setPinnedProduct] = useState(null);
  const [detectedProducts, setDetectedProducts] = useState([]);
  const [detectedObjects, setDetectedObjects] = useState([]);
  const [isProcessingFrame, setIsProcessingFrame] = useState(false);
  const [reactions, setReactions] = useState([]);
  const [currentCamera, setCurrentCamera] = useState('user'); // 'user' = front, 'environment' = back
  const [streamSource, setStreamSource] = useState(''); // 'camera' or 'screen'
  const [showSourceModal, setShowSourceModal] = useState(false);
  const [stats, setStats] = useState({
    sales: 0,
    orders: 0,
    addToCart: 0,
    watchTime: '0m 0s',
    viewers: 0
  });
  const [messages, setMessages] = useState([]);
  const [newMessage, setNewMessage] = useState('');
  const videoRef = useRef(null);
  const frameProcessingRef = useRef(null);

  useEffect(() => {
    loadProducts();

    // Cleanup on page unload
    const handleBeforeUnload = () => {
      if (isStreaming) {
        endStream();
      }
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    window.addEventListener('unload', handleBeforeUnload);

    return () => {
      if (isStreaming) {
        endStream();
      }
      window.removeEventListener('beforeunload', handleBeforeUnload);
      window.removeEventListener('unload', handleBeforeUnload);
    };
  }, [isStreaming]);

  const loadProducts = async () => {
    try {
      const response = await productAPI.getBySellerId(sellerId);
      setProducts(response.data);
    } catch (error) {
      // Error loading products
    }
  };

  const initializeStream = () => {
    if (!sellerId) {
      alert('Please enter a seller ID');
      return;
    }
    setHasStarted(true);
    setTimeout(() => loadProducts(), 100);
  };

  const startStream = async () => {
    setShowSourceModal(true);
  };

  const startStreamWithSource = async (source) => {
    setShowSourceModal(false);
    setStreamSource(source);
    
    try {
      // Start livestream in database
      const streamTitle = `${sellerId}'s Live Stream`;
      const streamData = {
        seller_id: sellerId,
        seller_name: `Seller ${sellerId}`,
        title: streamTitle,
        description: 'Live product showcase'
      };
      
      const streamResponse = await fetch(`${import.meta.env.VITE_API_URL}/api/livestreams/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(streamData)
      });
      
      if (!streamResponse.ok) {
        const errorData = await streamResponse.json();
        throw new Error(errorData.message || 'Failed to start livestream');
      }
      

      
      let stream;
      if (source === 'screen') {
        stream = await webrtcService.initializeScreenShare();

        
        // Listen for screen share end
        stream.getVideoTracks()[0].addEventListener('ended', () => {

          alert('Screen sharing ended. Redirecting to dashboard...');
          window.location.href = '/';
        });
      } else {
        stream = await webrtcService.initializeCamera(currentCamera);

      }
      
      // Ensure stream is set in WebRTC service
      webrtcService.localStream = stream;

      
      // Set streaming true first to render video element
      setIsStreaming(true);
      
      // Wait for next tick to ensure video element is rendered
      setTimeout(() => {
        if (videoRef.current) {
          videoRef.current.srcObject = stream;

        } else {

        }
      }, 50);
      
      // Setup WebRTC signaling listeners

      webrtcService.setupSignalingListeners();
      
      // Debug: Log when seller is ready to receive offers

      
      // Connect to WebSocket with seller's room

      websocketService.connect(`seller-${sellerId}`, `seller-${sellerId}`);
      
      // Wait for WebSocket to connect
      websocketService.on('connected', async () => {

        
        // Notify that seller is live
        websocketService.send({
          type: 'seller_live',
          data: { seller_id: sellerId, status: 'live' }
        });
        

        
        // Start frame processing for ML prediction (disabled for debugging)
        startFrameProcessing();

      });
      
      // Listen for auto pin updates
      websocketService.on('product_pinned', (message) => {
        loadPinnedProducts();
      });
      
      websocketService.on('product_unpinned', (message) => {
        setPinnedProduct(null);
      });
    } catch (error) {

      alert('Failed to access camera. Please check permissions.');
    }
  };

  const endStream = async () => {
    try {
      // End livestream in database
      const endResponse = await fetch(`${import.meta.env.VITE_API_URL}/api/livestreams/end/${sellerId}`, {
        method: 'POST'
      });
      
      if (endResponse.ok) {

      }
      
      // Unpin all products for this seller
      await pinAPI.unpinAllProducts(sellerId);

    } catch (error) {

    }
    
    websocketService.send({
      type: 'seller_offline',
      data: { seller_id: sellerId, status: 'offline' }
    });
    
    // Stop frame processing
    if (frameProcessingRef.current) {
      clearInterval(frameProcessingRef.current);
    }
    
    // Stop all tracks
    if (videoRef.current && videoRef.current.srcObject) {
      videoRef.current.srcObject.getTracks().forEach(track => track.stop());
    }
    
    // Cleanup streaming services
    try {
      webrtcService.destroy();
      websocketService.disconnect();
    } catch (error) {

    }
    
    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }
    setIsStreaming(false);
    setStreamEnded(true);
  };

  const startFrameProcessing = () => {
    // Process frame every 5 seconds for CPU optimization
    frameProcessingRef.current = setInterval(async () => {
      if (videoRef.current && !isProcessingFrame) {
        await captureAndProcessFrame();
      }
    }, 5000);
  };
  
  const captureAndProcessFrame = async () => {
    if (!videoRef.current || isProcessingFrame) return;
    
    try {
      setIsProcessingFrame(true);
      
      // Capture frame from video
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      canvas.width = videoRef.current.videoWidth;
      canvas.height = videoRef.current.videoHeight;
      ctx.drawImage(videoRef.current, 0, 0);
      
      // Convert to blob
      canvas.toBlob(async (blob) => {
        if (blob) {
          try {
            const response = await streamAPI.predictFrame(sellerId, blob);
            
            // Update detected products (recognized products)
            if (response.data.predictions?.length > 0) {
              setDetectedProducts(response.data.predictions);
              
              // Auto-pin high confidence products
              const highConfidenceProducts = response.data.predictions.filter(
                p => p.similarity_score >= 0.8
              );
              
              if (highConfidenceProducts.length > 0) {
                const bestProduct = highConfidenceProducts.reduce((prev, current) => 
                  (prev.similarity_score > current.similarity_score) ? prev : current
                );
                
                // Pin the best product
                try {
                  await pinAPI.pinProduct(bestProduct.product_id, parseInt(sellerId), bestProduct.similarity_score);
                  
                  // Send WebSocket message to notify viewers
                  const pinMessage = {
                    type: 'product_pinned',
                    data: {
                      product_id: bestProduct.product_id,
                      product_name: bestProduct.product_name,
                      price: bestProduct.price,
                      similarity_score: bestProduct.similarity_score
                    }
                  };
                  websocketService.send(pinMessage);
                } catch (error) {
                  console.error('❌ Failed to pin product:', error);
                }
              }
            } else {
              setDetectedProducts([]);
            }
            
            // Update detected objects (all YOLO detections)
            if (response.data.detections?.length > 0) {
              setDetectedObjects(response.data.detections);

            } else {
              setDetectedObjects([]);
            }
          } catch (error) {

            setDetectedProducts([]);
            setDetectedObjects([]);
          }
        }
        setIsProcessingFrame(false);
      }, 'image/jpeg', 0.8);
    } catch (error) {

      setIsProcessingFrame(false);
    }
  };
  
  const loadPinnedProducts = async () => {
    try {
      const response = await pinAPI.getPinnedProducts(sellerId);
      if (response.data.length > 0) {
        setPinnedProduct(response.data[0].product);
      }
    } catch (error) {

    }
  };

  const pinProduct = async (product) => {
    try {
      if (pinnedProduct?.id === product.id) {
        // Unpin the product
        await pinAPI.unpinProduct(product.id, sellerId);
        setPinnedProduct(null);
        
        // Send WebSocket message
        websocketService.send({
          type: 'product_unpinned',
          data: { product_id: product.id }
        });
      } else {
        // Pin the product
        await pinAPI.pinProduct(product.id, parseInt(sellerId), 1.0);
        setPinnedProduct(product);
        
        // Send WebSocket message
        websocketService.send({
          type: 'product_pinned',
          data: {
            product_id: product.id,
            product_name: product.name,
            price: product.price,
            similarity_score: 1.0
          }
        });
      }
    } catch (error) {

    }
  };

  useEffect(() => {
    if (isStreaming) {
      websocketService.on('user_joined', (message) => {

        setStats(prev => ({ ...prev, viewers: prev.viewers + 1 }));
        
        // Ensure stream is available for new viewers
        if (videoRef.current && videoRef.current.srcObject) {
          webrtcService.localStream = videoRef.current.srcObject;

        }
      });
      


      websocketService.on('user_left', (message) => {

        setStats(prev => ({ ...prev, viewers: Math.max(0, prev.viewers - 1) }));
      });

      const handleChat = (message) => {

        setMessages(prev => [...prev, message.data]);
      };

      const handleReaction = (message) => {

        const newReaction = {
          id: Date.now() + Math.random(),
          emoji: message.data.emoji,
          x: Math.random() * 80 + 10,
        };
        setReactions(prev => [...prev, newReaction]);
        
        setTimeout(() => {
          setReactions(prev => prev.filter(r => r.id !== newReaction.id));
        }, 3000);
      };

      websocketService.on('chat', handleChat);
      websocketService.on('reaction', handleReaction);
    }
  }, [isStreaming]);

  const sendMessage = () => {
    if (newMessage.trim()) {
      websocketService.sendChat(newMessage, `Seller ${sellerId}`);
      setNewMessage('');
    }
  };

  const switchCamera = async () => {
    if (!isStreaming || streamSource === 'screen') return;
    
    try {
      // Stop current stream
      if (videoRef.current && videoRef.current.srcObject) {
        videoRef.current.srcObject.getTracks().forEach(track => track.stop());
      }
      
      // Switch camera
      const newCamera = currentCamera === 'user' ? 'environment' : 'user';
      setCurrentCamera(newCamera);
      
      // Initialize new camera
      const stream = await webrtcService.initializeCamera(newCamera);
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
      }
      

    } catch (error) {

      alert('Failed to switch camera');
    }
  };

  // Seller ID Input Screen
  if (!hasStarted) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center p-4">
        <div className="max-w-md w-full bg-gray-800 rounded-2xl p-8 border border-gray-700">
          <div className="text-center mb-6">
            <div className="w-16 h-16 bg-red-500 rounded-full flex items-center justify-center mx-auto mb-4">
              <Video className="w-8 h-8 text-white" />
            </div>
            <h2 className="text-white text-2xl font-bold mb-2">Start Livestream</h2>
            <p className="text-gray-400">Enter your seller ID to create a livestream room</p>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-gray-400 text-sm font-medium mb-2">Seller ID</label>
              <input
                type="text"
                value={sellerId}
                onChange={(e) => setSellerId(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && initializeStream()}
                placeholder="Enter your seller ID (e.g., 1)"
                className="w-full bg-gray-700 border border-gray-600 rounded-lg text-white px-4 py-3 focus:ring-2 focus:ring-red-500 focus:border-transparent outline-none"
              />
            </div>

            <button
              onClick={initializeStream}
              className="w-full bg-red-500 text-white font-bold py-3 rounded-lg hover:bg-red-600 transition-colors"
            >
              Create Stream Room
            </button>

            <button
              onClick={() => window.location.href = '/'}
              className="w-full bg-gray-700 text-white font-medium py-3 rounded-lg hover:bg-gray-600 transition-colors"
            >
              Back to Home
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Sidebar */}
      <aside className="w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 p-4">
        <div className="flex flex-col h-full">
          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-3 px-3">
              <div className="w-10 h-10 bg-red-500 rounded-full flex items-center justify-center text-white font-bold">S</div>
              <div>
                <h1 className="text-gray-900 dark:text-white text-base font-semibold">The Seller Shop</h1>
                <p className="text-gray-500 dark:text-gray-400 text-sm">seller@email.com</p>
              </div>
            </div>

            <nav className="flex flex-col gap-1 mt-4">
              <a href="/" className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700">
                <LayoutDashboard className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                <p className="text-gray-700 dark:text-white text-sm font-medium">Dashboard</p>
              </a>
              <a href="/admin" className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700">
                <Package className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                <p className="text-gray-700 dark:text-white text-sm font-medium">Product Management</p>
              </a>
              <a href="/seller" className="flex items-center gap-3 px-3 py-2 rounded-lg bg-red-50 dark:bg-gray-700">
                <Tv className="w-5 h-5 text-red-500 dark:text-white" />
                <p className="text-red-500 dark:text-white text-sm font-semibold">Live Streams</p>
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
      <main className="flex-1 grid grid-cols-12 gap-6 p-6 min-h-0">
        <div className="col-span-12 lg:col-span-9 flex flex-col gap-6">
          {/* Video Preview */}
          <div className="relative w-full aspect-video bg-black rounded-xl overflow-hidden shadow-lg">
            {!isStreaming ? (
              <div className="absolute inset-0 flex items-center justify-center bg-gray-800">
                {streamEnded ? (
                  <div className="text-center">
                    <div className="w-16 h-16 bg-red-500/20 rounded-full flex items-center justify-center text-white mx-auto mb-4">
                      <Video className="w-8 h-8 text-red-500" />
                    </div>
                    <p className="text-red-400 text-lg font-semibold mb-2">Stream Ended</p>
                    <p className="text-gray-400 mb-4">Your livestream has ended successfully</p>
                    <button
                      onClick={() => window.location.href = '/'}
                      className="px-6 py-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600 transition-colors"
                    >
                      Back to Dashboard
                    </button>
                  </div>
                ) : (
                  <button
                    onClick={startStream}
                    className="px-8 py-4 bg-red-500 text-white font-bold rounded-lg hover:bg-red-600 transition-colors"
                  >
                    Start Livestream
                  </button>
                )}
              </div>
            ) : (
              <>
                <div className="relative w-full h-full">
                  <video
                    ref={videoRef}
                    autoPlay
                    playsInline
                    muted
                    className="w-full h-full object-cover"
                  />
                  
                  {/* Object Detection Bounding Boxes */}
                  {detectedObjects.map((obj, index) => {
                    if (!videoRef.current) return null;
                    
                    const videoRect = videoRef.current.getBoundingClientRect();
                    const videoWidth = videoRef.current.videoWidth || 640;
                    const videoHeight = videoRef.current.videoHeight || 480;
                    
                    const scaleX = videoRect.width / videoWidth;
                    const scaleY = videoRect.height / videoHeight;
                    
                    const [x1, y1, x2, y2] = obj.bbox;
                    const left = x1 * scaleX;
                    const top = y1 * scaleY;
                    const width = (x2 - x1) * scaleX;
                    const height = (y2 - y1) * scaleY;
                    
                    return (
                      <div
                        key={`obj-${index}`}
                        className="absolute border-2 border-blue-400 bg-blue-400/10"
                        style={{
                          left: `${left}px`,
                          top: `${top}px`,
                          width: `${width}px`,
                          height: `${height}px`,
                        }}
                      >
                        <div className="absolute -top-6 left-0 bg-blue-500 text-white text-xs px-2 py-1 rounded">
                          {obj.class} ({Math.round(obj.confidence * 100)}%)
                        </div>
                      </div>
                    );
                  })}
                  
                  {/* Product Recognition Bounding Boxes */}
                  {detectedProducts.map((product, index) => {
                    if (!videoRef.current) return null;
                    
                    const videoRect = videoRef.current.getBoundingClientRect();
                    const videoWidth = videoRef.current.videoWidth || 640;
                    const videoHeight = videoRef.current.videoHeight || 480;
                    
                    const scaleX = videoRect.width / videoWidth;
                    const scaleY = videoRect.height / videoHeight;
                    
                    const [x1, y1, x2, y2] = product.bbox;
                    const left = x1 * scaleX;
                    const top = y1 * scaleY;
                    const width = (x2 - x1) * scaleX;
                    const height = (y2 - y1) * scaleY;
                    
                    return (
                      <div
                        key={`product-${index}`}
                        className="absolute border-2 border-green-400 bg-green-400/10"
                        style={{
                          left: `${left}px`,
                          top: `${top}px`,
                          width: `${width}px`,
                          height: `${height}px`,
                        }}
                      >
                        <div className="absolute -top-12 left-0 bg-green-500 text-white text-xs px-2 py-1 rounded max-w-48">
                          <div className="font-semibold">{product.product_name}</div>
                          <div>${product.price} ({Math.round(product.similarity_score * 100)}%)</div>
                        </div>
                      </div>
                    );
                  })}
                </div>
                
                <div className="absolute top-4 left-4 flex items-center gap-2">
                  <span className="relative flex h-3 w-3">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-500 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-3 w-3 bg-red-500"></span>
                  </span>
                  <span className="text-white font-semibold text-sm">LIVE</span>
                </div>

                <div className="absolute top-4 right-4 flex flex-col gap-2">
                  <div className="flex items-center gap-2 bg-black/30 text-white px-3 py-1.5 rounded-lg backdrop-blur-sm">
                    <Eye className="w-4 h-4" />
                    <span className="text-sm font-medium">{stats.viewers}</span>
                  </div>
                  {isProcessingFrame && (
                    <div className="bg-purple-500/80 text-white px-3 py-1.5 rounded-lg backdrop-blur-sm">
                      <span className="text-xs font-medium">AI Processing...</span>
                    </div>
                  )}
                  {detectedObjects.length > 0 && (
                    <div className="bg-blue-500/80 text-white px-3 py-1.5 rounded-lg backdrop-blur-sm">
                      <span className="text-xs font-medium">{detectedObjects.length} Objects</span>
                    </div>
                  )}
                  {detectedProducts.length > 0 && (
                    <div className="bg-green-500/80 text-white px-3 py-1.5 rounded-lg backdrop-blur-sm">
                      <span className="text-xs font-medium">{detectedProducts.length} Products</span>
                    </div>
                  )}
                </div>

                {/* Floating Reactions */}
                <div className="absolute inset-0 pointer-events-none overflow-hidden">
                  <AnimatePresence>
                    {reactions.map((reaction) => (
                      <motion.div
                        key={reaction.id}
                        className="absolute bottom-20 text-4xl"
                        style={{ left: `${reaction.x}%` }}
                        initial={{ opacity: 1, y: 0, scale: 0.8 }}
                        animate={{ opacity: 0, y: -200, scale: 1.2 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 3, ease: "easeOut" }}
                      >
                        {reaction.emoji}
                      </motion.div>
                    ))}
                  </AnimatePresence>
                </div>

                <div className="absolute bottom-4 left-4 right-4 flex justify-between items-end">
                  <div className="text-white">
                    <h2 className="text-lg font-bold">Live Product Showcase</h2>
                    <p className="text-sm opacity-80">Featuring our latest products</p>
                  </div>
                  <div className="flex items-center gap-3">
                    {streamSource === 'camera' && (
                      <button
                        onClick={switchCamera}
                        className="p-2.5 rounded-full bg-black/30 text-white hover:bg-black/50 transition-colors"
                        title="Switch Camera"
                      >
                        <RotateCcw className="w-4 h-4" />
                      </button>
                    )}
                    <button
                      onClick={endStream}
                      className="px-6 py-2.5 rounded-full bg-red-500 text-white font-bold flex items-center gap-2 hover:bg-red-700 transition-colors"
                    >
                      <Video className="w-4 h-4" /> End Stream
                    </button>
                  </div>
                </div>
              </>
            )}
            
            {/* Stream Source Selection Modal */}
            {showSourceModal && (
              <div className="absolute inset-0 bg-black/50 flex items-center justify-center z-50">
                <div className="bg-white dark:bg-gray-800 rounded-xl p-6 max-w-sm w-full mx-4">
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white mb-4 text-center">
                    Choose Stream Source
                  </h3>
                  <div className="space-y-3">
                    <button
                      onClick={() => startStreamWithSource('camera')}
                      className="w-full flex items-center gap-3 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
                    >
                      <Video className="w-6 h-6 text-blue-500" />
                      <div className="text-left">
                        <div className="font-medium text-gray-900 dark:text-white">Camera</div>
                        <div className="text-sm text-gray-500 dark:text-gray-400">Use device camera</div>
                      </div>
                    </button>
                    <button
                      onClick={() => startStreamWithSource('screen')}
                      className="w-full flex items-center gap-3 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
                    >
                      <div className="w-6 h-6 bg-green-500 rounded flex items-center justify-center">
                        <div className="w-4 h-3 bg-white rounded-sm"></div>
                      </div>
                      <div className="text-left">
                        <div className="font-medium text-gray-900 dark:text-white">Screen Share</div>
                        <div className="text-sm text-gray-500 dark:text-gray-400">Share your screen</div>
                      </div>
                    </button>
                  </div>
                  <button
                    onClick={() => setShowSourceModal(false)}
                    className="w-full mt-4 py-2 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Products Below Video */}
          <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl">
            <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
              <h3 className="font-bold text-lg dark:text-white">Products in this Stream</h3>
              <a href="/admin" className="text-red-500 hover:text-red-600 text-sm font-medium flex items-center gap-1">
                <Plus className="w-4 h-4" /> Add Product
              </a>
            </div>
            <div className="p-4 grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
              {products?.length > 0 ? (
                products.slice(0, 8).map((product) => (
                  <div key={product.id} className="relative group">
                    <div className="aspect-square bg-gray-200 dark:bg-gray-700 rounded-lg overflow-hidden mb-2">
                      {product.images && product.images.length > 0 ? (
                        <img src={product.images[0].image_url} alt={product.name} className="w-full h-full object-cover" />
                      ) : product.image_url ? (
                        <img src={product.image_url} alt={product.name} className="w-full h-full object-cover" />
                      ) : (
                        <div className="w-full h-full flex items-center justify-center">
                          <Package className="w-8 h-8 text-gray-400" />
                        </div>
                      )}
                    </div>
                    <p className="font-medium text-sm text-gray-800 dark:text-white truncate">{product.name}</p>
                    <p className="font-bold text-red-500">${product.price}</p>
                    <button
                      onClick={() => pinProduct(product)}
                      className={`absolute top-2 right-2 w-8 h-8 rounded-full flex items-center justify-center transition-all ${
                        pinnedProduct?.id === product.id
                          ? 'bg-red-500 text-white'
                          : 'bg-white/80 dark:bg-gray-800/80 text-gray-600 dark:text-gray-400 opacity-0 group-hover:opacity-100'
                      }`}
                    >
                      <Pin className="w-4 h-4" />
                    </button>
                  </div>
                ))
              ) : (
                <div className="col-span-full text-center py-8 text-gray-500 dark:text-gray-400">
                  <Package className="w-12 h-12 mx-auto mb-2 text-gray-400" />
                  <p>No products available</p>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Chat Sidebar */}
        <div className="col-span-12 lg:col-span-3 flex flex-col bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl h-fit lg:h-[600px]">
          <div className="p-4 border-b border-gray-200 dark:border-gray-700">
            <h3 className="font-bold text-lg dark:text-white">Live Chat</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">{stats.viewers} viewers</p>
          </div>

          <div className="flex-1 p-4 space-y-3 overflow-y-auto min-h-[300px] lg:min-h-0">
            {messages.map((msg, i) => (
              <div key={i} className="flex gap-2">
                <div className="w-8 h-8 bg-gray-300 dark:bg-gray-600 rounded-full flex-shrink-0 flex items-center justify-center">
                  <span className="text-xs font-bold text-gray-600 dark:text-gray-300">{msg.username?.charAt(0)}</span>
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-semibold text-gray-700 dark:text-gray-300">{msg.username}</p>
                  <p className="text-sm text-gray-900 dark:text-white break-words">{msg.message}</p>
                </div>
              </div>
            ))}
            {messages.length === 0 && (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                <p className="text-sm">No messages yet</p>
              </div>
            )}
          </div>

          <div className="p-4 border-t border-gray-200 dark:border-gray-700">
            <div className="flex gap-2">
              <input
                type="text"
                value={newMessage}
                onChange={(e) => setNewMessage(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
                placeholder="Send a message..."
                className="flex-1 bg-gray-100 dark:bg-gray-700 border-none rounded-lg text-gray-900 dark:text-white px-3 py-2 text-sm placeholder:text-gray-500 focus:ring-2 focus:ring-red-500 outline-none"
              />
              <button
                onClick={sendMessage}
                className="w-10 h-10 bg-red-500 rounded-lg text-white hover:bg-red-600 flex items-center justify-center transition-colors"
              >
                <Send className="w-5 h-5" />
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
};

export default LiveStreamSeller;