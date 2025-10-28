import React, { useRef, useEffect } from 'react';

const VideoPlayer = ({ stream, isLocal = false, className = '' }) => {
  const videoRef = useRef(null);

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);

  return (
    <video
      ref={videoRef}
      autoPlay
      playsInline
      muted={isLocal}
      className={`w-full h-full object-cover ${className}`}
    />
  );
};

export default VideoPlayer;