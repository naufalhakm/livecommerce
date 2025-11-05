import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Bell, User, Tv } from 'lucide-react';

const LiveStreamList = () => {
  const [liveStreams, setLiveStreams] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [activeFilter, setActiveFilter] = useState('Trending');
  const navigate = useNavigate();

  useEffect(() => {
    fetchLiveStreams();
    const interval = setInterval(fetchLiveStreams, 10000);
    return () => clearInterval(interval);
  }, []);

  const fetchLiveStreams = async () => {
    try {
      const response = await fetch(`${import.meta.env.VITE_API_URL}/api/livestreams/active`);
      const data = await response.json();
      
      if (data.success) {
        setLiveStreams(data.data || []);
      } else {
        setError(data.message || 'Failed to fetch live streams');
      }
    } catch (err) {
      setError('Network error: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  const joinStream = (sellerId, viewerCount) => {
    if (viewerCount >= 1) {
      alert('Stream is full! This stream already has a viewer. Please try another stream.');
      return;
    }
    navigate(`/viewer?seller=${sellerId}`);
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-red-500"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="bg-red-900/20 border border-red-500 text-red-400 px-6 py-4 rounded-lg">
          <strong className="font-bold">Error!</strong>
          <span className="block mt-2">{error}</span>
          <button 
            onClick={fetchLiveStreams}
            className="mt-4 bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-900">
      {/* Header */}
      <header className="flex items-center justify-between px-6 py-4 border-b border-gray-800">
        <div className="flex items-center gap-8">
          <div className="flex items-center gap-3">
            <div className="w-6 h-6 bg-red-500 rounded"></div>
            <h1 className="text-white text-xl font-bold">Live Commerce AI</h1>
          </div>
          <nav className="hidden md:flex items-center gap-6">
            <button onClick={() => navigate('/')} className="text-gray-400 hover:text-white">Home</button>
            <button className="text-white font-medium">Explore</button>
          </nav>
          <div className="hidden md:flex items-center bg-gray-800 rounded-lg px-4 py-2">
            <Search className="w-4 h-4 text-gray-400 mr-2" />
            <input
              type="text"
              placeholder="Search streams..."
              className="bg-transparent text-white border-none outline-none w-48"
            />
          </div>
        </div>
        <div className="flex items-center gap-4">
          <button className="p-2 text-gray-400 hover:text-white">
            <Bell className="w-5 h-5" />
          </button>
          <div className="w-8 h-8 bg-gray-600 rounded-full flex items-center justify-center">
            <User className="w-4 h-4 text-gray-400" />
          </div>
        </div>
      </header>

      <div className="px-6 py-6">
        {/* Featured Stream */}
        {liveStreams.length > 0 && (
          <div className="relative mb-8 h-80 bg-gradient-to-r from-blue-600 to-purple-600 rounded-xl overflow-hidden">
            <div className="absolute inset-0 bg-black/30"></div>
            <div className="absolute top-4 left-4">
              <span className="bg-red-500 text-white text-xs font-bold px-2 py-1 rounded">LIVE</span>
            </div>
            <div className="absolute bottom-6 left-6">
              <h2 className="text-white text-3xl font-bold mb-2">Featured: {liveStreams[0].title}</h2>
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-gray-600 rounded-full"></div>
                <span className="text-white font-medium">{liveStreams[0].seller_name}</span>
                <span className="text-gray-300">• {liveStreams[0].viewer_count}k viewers</span>
              </div>
            </div>
            <button 
              onClick={() => joinStream(liveStreams[0].seller_id, liveStreams[0].viewer_count)}
              className={`absolute bottom-6 right-6 px-6 py-2 rounded-lg font-semibold transition-colors ${
                liveStreams[0].viewer_count >= 1 
                  ? 'bg-gray-500 text-gray-300 cursor-not-allowed' 
                  : 'bg-white text-black hover:bg-gray-100'
              }`}
              disabled={liveStreams[0].viewer_count >= 1}
            >
              {liveStreams[0].viewer_count >= 1 ? 'Stream Full' : 'Watch Now'}
            </button>
          </div>
        )}

        {/* Filter Tabs */}
        <div className="flex items-center gap-4 mb-6">
          {['Trending', 'Newest', 'Most Viewers'].map((filter) => (
            <button
              key={filter}
              onClick={() => setActiveFilter(filter)}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                activeFilter === filter
                  ? 'bg-red-500 text-white'
                  : 'text-gray-400 hover:text-white'
              }`}
            >
              {filter}
            </button>
          ))}
        </div>

        {/* Trending Now */}
        <div className="mb-8">
          <h3 className="text-white text-xl font-bold mb-4">Trending Now</h3>
          {liveStreams.length === 0 ? (
            <div className="text-center py-12">
              <Tv className="w-16 h-16 text-gray-600 mx-auto mb-4" />
              <h2 className="text-xl font-semibold text-gray-400 mb-2">No Live Streams</h2>
              <p className="text-gray-500">There are no active live streams at the moment.</p>
              <button 
                onClick={fetchLiveStreams}
                className="mt-4 bg-red-500 text-white px-6 py-2 rounded-lg hover:bg-red-600 transition-colors"
              >
                Refresh
              </button>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {liveStreams.slice(1).map((stream) => (
                <div key={stream.id} className={`group ${stream.viewer_count >= 1 ? 'cursor-not-allowed opacity-60' : 'cursor-pointer'}`} onClick={() => joinStream(stream.seller_id, stream.viewer_count)}>
                  <div className="relative aspect-video bg-gray-800 rounded-lg overflow-hidden mb-3">
                    <div className="absolute inset-0 bg-gradient-to-br from-red-500/20 to-purple-500/20 flex items-center justify-center">
                      <div className="text-4xl">🎥</div>
                    </div>
                    <div className="absolute top-2 left-2 bg-red-500 text-white px-2 py-1 rounded text-xs font-bold">
                      LIVE
                    </div>
                    <div className={`absolute bottom-2 right-2 px-2 py-1 rounded text-xs ${
                      stream.viewer_count >= 1 
                        ? 'bg-red-600 text-white' 
                        : 'bg-black/60 text-white'
                    }`}>
                      👥 {stream.viewer_count} {stream.viewer_count >= 1 ? '(Full)' : ''}
                    </div>
                  </div>
                  <div className="flex items-start gap-3">
                    <div className="w-8 h-8 bg-gray-600 rounded-full flex-shrink-0"></div>
                    <div className="flex-1 min-w-0">
                      <h4 className={`font-medium text-sm line-clamp-2 transition-colors ${
                        stream.viewer_count >= 1 
                          ? 'text-gray-400' 
                          : 'text-white group-hover:text-red-400'
                      }`}>
                        {stream.title} {stream.viewer_count >= 1 ? '(Full)' : ''}
                      </h4>
                      <p className="text-gray-400 text-xs mt-1">{stream.seller_name}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default LiveStreamList;