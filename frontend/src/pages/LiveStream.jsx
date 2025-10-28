import React, { useState, useEffect, useRef } from 'react';
import VideoPlayer from '../components/VideoPlayer';
import ProductOverlay from '../components/ProductOverlay';
import webrtcService from '../services/webrtc';
import websocketService from '../services/websocket';
import { streamAPI } from '../services/api';

const LiveStream = () => {
  const [isStreaming, setIsStreaming] = useState(false);
  const [localStream, setLocalStream] = useState(null);
  const [remoteStream, setRemoteStream] = useState(null);
  const [predictions, setPredictions] = useState([]);
  const [selectedProduct, setSelectedProduct] = useState(null);
  const [sellerId, setSellerId] = useState('1');
  const intervalRef = useRef(null);

  useEffect(() => {
    // Connect to WebSocket
    websocketService.connect('livestream-client');

    // Listen for product detection results
    websocketService.on('product_detection', (message) => {
      setPredictions(message.data.predictions || []);
    });

    // Setup WebRTC event handlers
    webrtcService.onRemoteStream = (stream) => {
      setRemoteStream(stream);
    };

    webrtcService.onConnect = () => {
      console.log('WebRTC connected');
    };

    webrtcService.onError = (error) => {
      console.error('WebRTC error:', error);
    };

    return () => {
      websocketService.disconnect();
      webrtcService.destroy();
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  const startStreaming = async () => {
    try {
      const stream = await webrtcService.initializeCamera();
      setLocalStream(stream);
      webrtcService.startBroadcast();
      setIsStreaming(true);

      // Start frame processing for ML detection
      intervalRef.current = setInterval(async () => {
        try {
          const frameBlob = await webrtcService.captureFrame();
          if (frameBlob) {
            await streamAPI.processFrame(sellerId, frameBlob);
          }
        } catch (error) {
          console.error('Error processing frame:', error);
        }
      }, 2000); // Process frame every 2 seconds

    } catch (error) {
      console.error('Error starting stream:', error);
      alert('Failed to start streaming. Please check camera permissions.');
    }
  };

  const stopStreaming = () => {
    webrtcService.destroy();
    setLocalStream(null);
    setRemoteStream(null);
    setIsStreaming(false);
    setPredictions([]);
    
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  };

  const joinStream = async () => {
    try {
      webrtcService.joinBroadcast();
    } catch (error) {
      console.error('Error joining stream:', error);
      alert('Failed to join stream');
    }
  };

  const handleProductClick = (prediction) => {
    setSelectedProduct(prediction);
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">Live Stream</h1>
      
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Video Stream */}
        <div className="lg:col-span-2">
          <div className="bg-black rounded-lg overflow-hidden relative aspect-video">
            {localStream && (
              <>
                <VideoPlayer stream={localStream} isLocal={true} />
                <ProductOverlay 
                  predictions={predictions} 
                  onProductClick={handleProductClick}
                />
              </>
            )}
            
            {remoteStream && !localStream && (
              <>
                <VideoPlayer stream={remoteStream} />
                <ProductOverlay 
                  predictions={predictions} 
                  onProductClick={handleProductClick}
                />
              </>
            )}
            
            {!localStream && !remoteStream && (
              <div className="flex items-center justify-center h-full text-white">
                <div className="text-center">
                  <p className="text-xl mb-4">No stream active</p>
                  <div className="space-x-4">
                    <button
                      onClick={startStreaming}
                      className="bg-red-600 text-white px-6 py-2 rounded hover:bg-red-700"
                    >
                      Start Streaming
                    </button>
                    <button
                      onClick={joinStream}
                      className="bg-blue-600 text-white px-6 py-2 rounded hover:bg-blue-700"
                    >
                      Join Stream
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
          
          {/* Stream Controls */}
          <div className="mt-4 flex justify-between items-center">
            <div className="flex items-center space-x-4">
              <label className="text-sm font-medium">Seller ID:</label>
              <input
                type="text"
                value={sellerId}
                onChange={(e) => setSellerId(e.target.value)}
                className="border border-gray-300 rounded px-2 py-1 w-20"
                disabled={isStreaming}
              />
            </div>
            
            {isStreaming && (
              <button
                onClick={stopStreaming}
                className="bg-gray-600 text-white px-4 py-2 rounded hover:bg-gray-700"
              >
                Stop Streaming
              </button>
            )}
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Detection Status */}
          <div className="bg-white p-4 rounded-lg shadow">
            <h3 className="text-lg font-semibold mb-3">Detection Status</h3>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span>Stream Active:</span>
                <span className={isStreaming ? 'text-green-600' : 'text-red-600'}>
                  {isStreaming ? 'Yes' : 'No'}
                </span>
              </div>
              <div className="flex justify-between">
                <span>Products Detected:</span>
                <span className="font-medium">{predictions.length}</span>
              </div>
            </div>
          </div>

          {/* Detected Products */}
          <div className="bg-white p-4 rounded-lg shadow">
            <h3 className="text-lg font-semibold mb-3">Detected Products</h3>
            <div className="space-y-3">
              {predictions.map((prediction, index) => (
                <div
                  key={index}
                  className="border border-gray-200 rounded p-3 cursor-pointer hover:bg-gray-50"
                  onClick={() => handleProductClick(prediction)}
                >
                  <div className="font-medium">{prediction.product_name}</div>
                  <div className="text-sm text-gray-600">
                    Confidence: {Math.round(prediction.confidence * 100)}%
                  </div>
                  <div className="text-sm text-gray-600">
                    Similarity: {Math.round(prediction.similarity_score * 100)}%
                  </div>
                </div>
              ))}
              
              {predictions.length === 0 && (
                <p className="text-gray-500 text-sm">No products detected</p>
              )}
            </div>
          </div>

          {/* Selected Product Details */}
          {selectedProduct && (
            <div className="bg-white p-4 rounded-lg shadow">
              <h3 className="text-lg font-semibold mb-3">Selected Product</h3>
              <div className="space-y-2">
                <div className="font-medium">{selectedProduct.product_name}</div>
                <div className="text-sm text-gray-600">
                  Detection Confidence: {Math.round(selectedProduct.confidence * 100)}%
                </div>
                <div className="text-sm text-gray-600">
                  Similarity Score: {Math.round(selectedProduct.similarity_score * 100)}%
                </div>
                <button className="w-full bg-blue-600 text-white py-2 px-4 rounded hover:bg-blue-700 mt-3">
                  Add to Cart
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default LiveStream;