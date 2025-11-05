import React, { useState, useEffect, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import websocketService from '../services/websocket';
import webrtcService from '../services/webrtc';
import sfuService from '../services/sfu';
import webrtcDirectService from '../services/webrtc_direct';
import { Tv, Search, ShoppingCart, Bell, User, Volume2, VolumeX, Maximize, Minimize, Heart, ThumbsUp, Flame, PartyPopper, Send } from 'lucide-react';

const LiveStreamViewer = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [sellerId, setSellerId] = useState(searchParams.get('seller') || '');
  const [hasJoined, setHasJoined] = useState(!!searchParams.get('seller'));
  const [messages, setMessages] = useState([]);
  const [newMessage, setNewMessage] = useState('');
  const [pinnedProduct, setPinnedProduct] = useState(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [viewerCount, setViewerCount] = useState(0);
  const [username] = useState(`Viewer${Math.floor(Math.random() * 1000)}`);
  const [isMuted, setIsMuted] = useState(true);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [reactions, setReactions] = useState([]);
  const [streamEnded, setStreamEnded] = useState(false);
  const videoRef = useRef(null);
  const videoContainerRef = useRef(null);
  const initialized = useRef(false);
  const [connectionStatus, setConnectionStatus] = useState('disconnected');
  const [retryCount, setRetryCount] = useState(0);
  const maxRetries = 3;

  // Auto-join if seller parameter is provided in URL, otherwise redirect to streams list
  useEffect(() => {
    const sellerParam = searchParams.get('seller');
    if (sellerParam && !hasJoined) {
      setSellerId(sellerParam);
      setHasJoined(true);
    } else if (!sellerParam && !hasJoined) {
      // Redirect to streams list if no seller parameter
      navigate('/streams');
    }
  }, [searchParams, hasJoined, navigate]);

  useEffect(() => {
    if (hasJoined && sellerId && !initialized.current) {
      initialized.current = true;
      
      setupConnectionCallbacks();
      initializeStreaming();
    }

    // Cleanup on page unload
    const handleBeforeUnload = () => {
      cleanupConnection();
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    window.addEventListener('unload', handleBeforeUnload);

    return () => {
      cleanupConnection();
      window.removeEventListener('beforeunload', handleBeforeUnload);
      window.removeEventListener('unload', handleBeforeUnload);
    };
  }, [hasJoined, sellerId]);

  // Setup chat and reaction handlers
  useEffect(() => {
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

    const handleSellerOffline = (message) => {
      setStreamEnded(true);
      setIsPlaying(false);
    };

    websocketService.on('chat', handleChat);
    websocketService.on('reaction', handleReaction);
    websocketService.on('seller_offline', handleSellerOffline);

    return () => {
      websocketService.off('chat', handleChat);
      websocketService.off('reaction', handleReaction);
      websocketService.off('seller_offline', handleSellerOffline);
    };
  }, []);


  const setupConnectionCallbacks = () => {
    // Enhanced WebSocket connection management
    websocketService.setConnectionCallbacks({
      onConnected: () => {
        setConnectionStatus('connected');
        setRetryCount(0);
      },
      onDisconnected: (event) => {
        setConnectionStatus('disconnected');
        
        if (retryCount < maxRetries) {
          setRetryCount(prev => prev + 1);
          setTimeout(() => {
            if (hasJoined && sellerId) {
              initializeStreaming();
            }
          }, 2000 * retryCount);
        }
      },
      onError: (error) => {
        setConnectionStatus('error');
      }
    });
  };

  const initializeStreaming = async () => {
    try {
      setConnectionStatus('connecting');
      
      // Setup WebRTC callbacks
      setupWebRTCCallbacks();
      
      // Setup WebSocket event handlers BEFORE connecting
      setupWebSocketHandlers();
      
      // Connect to seller's room
      const viewerClientId = `viewer-${Date.now()}`;
      websocketService.connect(viewerClientId, `seller-${sellerId}`);
      
    } catch (error) {
      setConnectionStatus('error');
    }
  };

  const setupWebRTCCallbacks = () => {
    webrtcService.onRemoteStream = (stream) => {
      if (!stream) {
        return;
      }
      
      handleStreamSetup(stream);
    };
    
    webrtcService.onError = (error) => {
      setConnectionStatus('error');
    };
  };

  const setupWebSocketHandlers = () => {
    // Setup WebRTC signaling listeners first
    webrtcService.setupSignalingListeners();
    
    // Setup WebSocket event handlers
    websocketService.on('connected', () => {
      setConnectionStatus('connected');
      setRetryCount(0);
    });
    
    // Wait for join confirmation, then initiate WebRTC
    websocketService.on('joined', async (message) => {
      try {
        
        // Add delay to ensure WebSocket is fully ready
        setTimeout(async () => {
          try {
            
            // Join broadcast by creating offer to seller
            const peer = await webrtcService.joinBroadcast(`seller-${sellerId}`);
          } catch (error) {
            setConnectionStatus('error');
          }
        }, 100);
      } catch (error) {
        setConnectionStatus('error');
      }
    });
  };

  const handleStreamSetup = (stream) => {
    if (!stream || !stream.active) {
      return;
    }
    
    // Verify video track is enabled
    const videoTrack = stream.getVideoTracks()[0];
    if (videoTrack) {
      // Ensure video track is enabled
      videoTrack.enabled = true;
    }

    if (videoRef.current) {
      videoRef.current.srcObject = stream;
      videoRef.current.muted = true; // Audio muted for viewer
      videoRef.current.playsInline = true;
      
      videoRef.current.onloadedmetadata = () => {
        
        // Only set playing if we have actual video content
        if (videoRef.current.videoWidth > 0 && videoRef.current.videoHeight > 0) {
          setIsPlaying(true);
          setConnectionStatus('connected');
        } else {
        }
        
        videoRef.current.play().catch(e => {
        });
      };
      
      videoRef.current.onerror = (e) => {
        setConnectionStatus('error');
      };
    }
  };

  const cleanupConnection = () => {
    if (initialized.current) {
      initialized.current = false;
      
      try {
        webrtcService.destroy();
        webrtcDirectService.disconnect();
        sfuService.disconnect();
        websocketService.disconnect();
      } catch (error) {
      }
      
      if (videoRef.current) {
        videoRef.current.srcObject = null;
      }
      
      setIsPlaying(false);
      setConnectionStatus('disconnected');
    }
  };


  const joinStream = (sellerIdToJoin) => {
    if (!sellerIdToJoin) {
      alert('Please enter a seller ID');
      return;
    }

    setHasJoined(true);
    navigate(`/viewer?seller=${sellerIdToJoin}`, { replace: true });
  };

  // Setup additional WebSocket handlers for product and user events
  useEffect(() => {
    if (!hasJoined) return;

    websocketService.on('product_pinned', async (message) => {

      // Only update pin if similarity is high enough (80% or higher)
      if (message.data.similarity_score < 0.8) {
        return;
      }
      
      try {
        // Fetch full product details from API
        const response = await fetch(`${import.meta.env.VITE_API_URL}/api/products/${message.data.product_id}`);
        if (response.ok) {
          const productData = await response.json();
          setPinnedProduct({
            ...productData,
            similarity_score: message.data.similarity_score
          });
        } else {
        }
      } catch (error) {
      }
    });

    websocketService.on('pin_product', (message) => {
      if (message.data.product) {
        setPinnedProduct(message.data.product);
      }
    });

    websocketService.on('user_joined', (message) => {
      setViewerCount(prev => prev + 1);
    });

    websocketService.on('user_left', (message) => {
      setViewerCount(prev => Math.max(0, prev - 1));
    });

    return () => {
      websocketService.off('product_pinned');
      websocketService.off('pin_product');
      websocketService.off('user_joined');
      websocketService.off('user_left');
    };
  }, [hasJoined]);

  const sendMessage = () => {
    if (newMessage.trim()) {
      websocketService.sendChat(newMessage, username);
      setNewMessage('');
    }
  };

  const sendReaction = (emoji) => {
    websocketService.send({
      type: 'reaction',
      data: { emoji }
    });
  };

  const toggleMute = () => {
    if (videoRef.current) {
      videoRef.current.muted = !videoRef.current.muted;
      setIsMuted(videoRef.current.muted);
    }
  };

  const toggleFullscreen = async () => {
    if (!videoContainerRef.current) return;

    try {
      if (!document.fullscreenElement) {
        await videoContainerRef.current.requestFullscreen();
        setIsFullscreen(true);
      } else {
        await document.exitFullscreen();
        setIsFullscreen(false);
      }
    } catch (error) {
    }
  };

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => document.removeEventListener('fullscreenchange', handleFullscreenChange);
  }, []);

  if (!hasJoined) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center p-4">
        <div className="max-w-md w-full bg-gray-800 rounded-2xl p-8 border border-gray-700">
          <div className="text-center mb-6">
            <div className="w-16 h-16 bg-red-500 rounded-full flex items-center justify-center mx-auto mb-4">
              <Tv className="w-8 h-8 text-white" />
            </div>
            <h2 className="text-white text-2xl font-bold mb-2">Join Livestream</h2>
            <p className="text-gray-400">Enter the seller ID to watch their live stream</p>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-gray-400 text-sm font-medium mb-2">Seller ID</label>
              <input
                type="text"
                value={sellerId}
                onChange={(e) => setSellerId(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && joinStream(sellerId)}
                placeholder="Enter seller ID (e.g., 1)"
                className="w-full bg-gray-700 border border-gray-600 rounded-lg text-white px-4 py-3 focus:ring-2 focus:ring-red-500 focus:border-transparent outline-none"
              />
            </div>

            <button
              onClick={() => joinStream(sellerId)}
              className="w-full bg-red-500 text-white font-bold py-3 rounded-lg hover:bg-red-600 transition-colors"
            >
              Join Stream
            </button>

            <button
              onClick={() => navigate('/streams')}
              className="w-full bg-gray-700 text-white font-medium py-3 rounded-lg hover:bg-gray-600 transition-colors"
            >
              Back to Streams
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col min-h-screen bg-gray-900">
      <header className="flex items-center justify-between px-6 py-3 bg-gray-900 border-b border-gray-800">
        <div className="flex items-center gap-8">
          <div className="flex items-center gap-3">
            <div className="w-6 h-6 bg-red-500 rounded"></div>
            <h2 className="text-white text-lg font-bold">LiveShop AI</h2>
          </div>
          <div className="hidden md:flex items-center bg-gray-800 rounded-lg px-4 py-2">
            <Search className="w-4 h-4 text-gray-400 mr-2" />
            <input
              type="text"
              placeholder="Search"
              className="bg-transparent text-white border-none outline-none w-40"
            />
          </div>
        </div>
        <div className="flex items-center gap-4">
          <button 
            onClick={() => navigate('/streams')}
            className="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors text-sm font-medium"
          >
            Browse Streams
          </button>
          <button className="p-2 bg-gray-800 rounded-lg text-white hover:bg-gray-700">
            <ShoppingCart className="w-5 h-5" />
          </button>
          <button className="p-2 bg-gray-800 rounded-lg text-white hover:bg-gray-700">
            <Bell className="w-5 h-5" />
          </button>
          <div className="w-10 h-10 bg-gray-600 rounded-full flex items-center justify-center">
            <User className="w-5 h-5 text-gray-400" />
          </div>
        </div>
      </header>

      <main className="flex-1 grid grid-cols-1 lg:grid-cols-[1fr_360px] p-6 gap-6">
        <div className="flex flex-col gap-4">
          <div ref={videoContainerRef} className="relative bg-black rounded-xl overflow-hidden aspect-video">
            <video
              ref={videoRef}
              autoPlay
              playsInline
              muted
              className="w-full h-full object-cover"
            />
            {!isPlaying && (
              <div className="absolute inset-0 w-full h-full bg-gray-800 flex items-center justify-center">
                <div className="text-center">
                  {streamEnded ? (
                    <>
                      <div className="w-16 h-16 bg-red-500/20 rounded-full flex items-center justify-center text-white mx-auto mb-4">
                        <Tv className="w-8 h-8 text-red-500" />
                      </div>
                      <p className="text-red-400 text-lg font-semibold mb-2">Stream Ended</p>
                      <p className="text-gray-400">Seller {sellerId} has ended the livestream</p>
                      <button
                        onClick={() => navigate('/streams')}
                        className="mt-4 px-6 py-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600 transition-colors"
                      >
                        Browse Other Streams
                      </button>
                    </>
                  ) : (
                    <>
                      <div className="w-16 h-16 bg-black/40 rounded-full flex items-center justify-center text-white mx-auto mb-4 animate-pulse">
                        <div className="w-8 h-8 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                      </div>
                      <p className="text-gray-400 animate-pulse">Connecting to Seller {sellerId}...</p>
                      <div className="flex justify-center mt-3">
                        <div className="flex space-x-1">
                          <div className="w-2 h-2 bg-red-500 rounded-full animate-bounce"></div>
                          <div className="w-2 h-2 bg-red-500 rounded-full animate-bounce" style={{animationDelay: '0.1s'}}></div>
                          <div className="w-2 h-2 bg-red-500 rounded-full animate-bounce" style={{animationDelay: '0.2s'}}></div>
                        </div>
                      </div>
                    </>
                  )}
                </div>
              </div>
            )}
            
            <div className="absolute inset-0 bg-gradient-to-t from-black/50 to-transparent pointer-events-none"></div>

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

            <div className="absolute bottom-4 left-4 right-4">
              <div className="flex items-center justify-between">
                <span className="bg-red-500 text-white text-xs font-bold px-2 py-1 rounded">LIVE</span>
                <div className="flex gap-2">
                  <button 
                    onClick={toggleMute}
                    className="text-white hover:text-red-500 transition-colors"
                    title={isMuted ? 'Unmute' : 'Mute'}
                  >
                    {isMuted ? <VolumeX className="w-5 h-5" /> : <Volume2 className="w-5 h-5" />}
                  </button>
                  <button 
                    onClick={toggleFullscreen}
                    className="text-white hover:text-red-500 transition-colors"
                    title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
                  >
                    {isFullscreen ? <Minimize className="w-5 h-5" /> : <Maximize className="w-5 h-5" />}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-between p-4 bg-gray-800 rounded-xl">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 bg-gray-600 rounded-full"></div>
              <div>
                <p className="text-white font-bold">Seller {sellerId}</p>
                <p className="text-gray-400 text-sm flex items-center gap-1">
                  <User className="w-4 h-4" /> {viewerCount} viewers
                </p>
              </div>
            </div>
          </div>

          {pinnedProduct && (
            <div className="bg-gray-800 rounded-xl p-4">
              <div className="flex items-center gap-4">
                {pinnedProduct.images && pinnedProduct.images.length > 0 ? (
                  <img src={pinnedProduct.images[0].image_url} alt={pinnedProduct.name || pinnedProduct.product_name} className="w-24 h-24 bg-gray-700 rounded-lg object-cover" />
                ) : pinnedProduct.image_url ? (
                  <img src={pinnedProduct.image_url} alt={pinnedProduct.name || pinnedProduct.product_name} className="w-24 h-24 bg-gray-700 rounded-lg object-cover" />
                ) : (
                  <div className="w-24 h-24 bg-gray-700 rounded-lg"></div>
                )}
                <div className="flex-1">
                  <p className="text-white text-lg font-bold">{pinnedProduct.name || pinnedProduct.product_name}</p>
                  <p className="text-red-500 text-xl font-bold">${pinnedProduct.price}</p>
                  {pinnedProduct.similarity_score && (
                    <p className="text-green-400 text-sm">
                      ✨ {Math.round(pinnedProduct.similarity_score * 100)}% Match
                    </p>
                  )}
                </div>
                <button className="bg-red-500 text-white px-6 py-3 rounded-lg font-bold hover:bg-red-600 flex items-center gap-2 transition-colors">
                  <ShoppingCart className="w-5 h-5" /> Add to Cart
                </button>
              </div>
            </div>
          )}
        </div>

        <aside className="flex flex-col bg-gray-800 rounded-xl h-fit lg:h-[600px]">
          <h2 className="text-white text-lg font-bold p-4 border-b border-gray-700">Live Chat</h2>
          
          <div className="flex-1 p-4 space-y-4 overflow-y-auto min-h-[300px] lg:min-h-0">
            <div className="p-3 rounded-lg bg-red-900/30">
              <p className="font-bold text-red-500">Seller {sellerId}</p>
              <p className="text-white text-sm">Welcome everyone! Check out our amazing products today!</p>
            </div>
            
            {messages.map((msg, i) => (
              <div key={i} className="flex gap-2 text-sm">
                <div className="w-6 h-6 bg-gray-600 rounded-full flex-shrink-0"></div>
                <p>
                  <span className="font-bold text-gray-400">{msg.username}:</span>
                  <span className="text-white ml-1">{msg.message}</span>
                </p>
              </div>
            ))}

            {messages.length === 0 && (
              <div className="text-center text-gray-500 py-8">
                <p>No messages yet</p>
                <p className="text-sm mt-2">Be the first to say something!</p>
              </div>
            )}
          </div>

          <div className="p-4 border-t border-gray-700 space-y-3">
            <div className="flex justify-around">
              <button onClick={() => sendReaction('❤️')} className="hover:scale-110 transition">
                <Heart className="w-6 h-6 text-red-500" />
              </button>
              <button onClick={() => sendReaction('👍')} className="hover:scale-110 transition">
                <ThumbsUp className="w-6 h-6 text-blue-500" />
              </button>
              <button onClick={() => sendReaction('🔥')} className="hover:scale-110 transition">
                <Flame className="w-6 h-6 text-orange-500" />
              </button>
              <button onClick={() => sendReaction('🎉')} className="hover:scale-110 transition">
                <PartyPopper className="w-6 h-6 text-yellow-500" />
              </button>
            </div>
            <div className="flex gap-2">
              <input
                type="text"
                value={newMessage}
                onChange={(e) => setNewMessage(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
                placeholder="Say something nice..."
                className="flex-1 bg-gray-700 border-none rounded-lg text-white px-4 py-2 text-sm placeholder:text-gray-500 focus:ring-2 focus:ring-red-500 outline-none"
              />
              <button
                onClick={sendMessage}
                className="w-10 h-10 bg-gray-700 rounded-lg text-white hover:bg-gray-600 flex items-center justify-center transition-colors"
              >
                <Send className="w-5 h-5" />
              </button>
            </div>
          </div>
        </aside>
      </main>
    </div>
  );
};

export default LiveStreamViewer;