import React from 'react';

const ProductOverlay = ({ predictions = [], onProductClick }) => {
  if (!predictions.length) return null;

  return (
    <div className="absolute inset-0 pointer-events-none">
      {predictions.map((prediction, index) => {
        const [x1, y1, x2, y2] = prediction.bbox;
        const width = x2 - x1;
        const height = y2 - y1;

        return (
          <div
            key={index}
            className="absolute border-2 border-red-500 pointer-events-auto cursor-pointer"
            style={{
              left: `${(x1 / 1280) * 100}%`,
              top: `${(y1 / 720) * 100}%`,
              width: `${(width / 1280) * 100}%`,
              height: `${(height / 720) * 100}%`,
            }}
            onClick={() => onProductClick?.(prediction)}
          >
            <div className="absolute -top-8 left-0 bg-red-500 text-white px-2 py-1 text-xs rounded">
              {prediction.product_name}
              <span className="ml-1 opacity-75">
                ({Math.round(prediction.similarity_score * 100)}%)
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default ProductOverlay;