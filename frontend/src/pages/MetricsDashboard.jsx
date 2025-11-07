import React, { useState, useEffect } from 'react';
import { Line, Bar, Doughnut } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  ArcElement,
} from 'chart.js';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  ArcElement
);

const MetricsDashboard = () => {
  const [metricsData, setMetricsData] = useState(null);
  const [thesisData, setThesisData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('overview');

  useEffect(() => {
    fetchMetricsData();
    fetchThesisData();
  }, []);

  const fetchMetricsData = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/metrics/');
      const data = await response.json();
      setMetricsData(data);
    } catch (error) {
      console.error('Error fetching metrics:', error);
    }
  };

  const fetchThesisData = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/metrics/thesis');
      const data = await response.json();
      setThesisData(data);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching thesis data:', error);
      setLoading(false);
    }
  };

  const downloadCSV = (type) => {
    const url = `http://localhost:8080/api/metrics/download?type=${type}`;
    const link = document.createElement('a');
    link.href = url;
    link.download = `${type}_metrics.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const generateThesisResults = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/export/thesis-results', {
        method: 'POST'
      });
      const result = await response.json();
      
      if (response.ok) {
        alert(`🎓 Thesis results generated successfully!\n\nFiles created:\n${result.files.join('\n')}\n\nLocation: ${result.location}`);
      } else {
        alert('Failed to generate thesis results');
      }
    } catch (error) {
      console.error('Error generating thesis results:', error);
      alert('Error generating thesis results');
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="text-white text-xl">Loading metrics data...</div>
      </div>
    );
  }

  const yoloPerformanceChart = {
    labels: ['Precision', 'Recall', 'F1-Score', 'mAP@0.5'],
    datasets: [{
      label: 'YOLO Performance (%)',
      data: thesisData ? [
        thesisData.yolo_performance.precision * 100,
        thesisData.yolo_performance.recall * 100,
        thesisData.yolo_performance.f1_score * 100,
        thesisData.yolo_performance.map_05 * 100
      ] : [],
      backgroundColor: ['#3B82F6', '#10B981', '#F59E0B', '#EF4444'],
      borderColor: ['#2563EB', '#059669', '#D97706', '#DC2626'],
      borderWidth: 2
    }]
  };

  const clipPerformanceChart = {
    labels: ['Top-1', 'Top-3', 'Top-5'],
    datasets: [{
      label: 'CLIP Accuracy (%)',
      data: thesisData ? [
        thesisData.clip_performance.top1_accuracy * 100,
        thesisData.clip_performance.top3_accuracy * 100,
        thesisData.clip_performance.top5_accuracy * 100
      ] : [],
      backgroundColor: '#8B5CF6',
      borderColor: '#7C3AED',
      borderWidth: 2
    }]
  };

  const latencyChart = {
    labels: ['YOLO', 'CLIP', 'FAISS', 'Overhead'],
    datasets: [{
      label: 'Latency (ms)',
      data: thesisData ? [
        thesisData.latency_breakdown.yolo_inference_ms,
        thesisData.latency_breakdown.clip_embedding_ms,
        thesisData.latency_breakdown.faiss_search_ms,
        thesisData.latency_breakdown.overhead_ms
      ] : [],
      backgroundColor: ['#F59E0B', '#10B981', '#3B82F6', '#EF4444'],
    }]
  };

  const scenarioChart = {
    labels: thesisData ? thesisData.scenario_comparison.map(s => s.scenario.replace('S', 'Skenario ')) : [],
    datasets: [
      {
        label: 'YOLO Only (%)',
        data: thesisData ? thesisData.scenario_comparison.map(s => s.yolo_only * 100) : [],
        backgroundColor: '#EF4444',
        borderColor: '#DC2626',
        borderWidth: 2
      },
      {
        label: 'Hybrid System (%)',
        data: thesisData ? thesisData.scenario_comparison.map(s => s.hybrid * 100) : [],
        backgroundColor: '#10B981',
        borderColor: '#059669',
        borderWidth: 2
      }
    ]
  };

  const chartOptions = {
    responsive: true,
    plugins: {
      legend: {
        labels: { color: '#fff' }
      },
      title: {
        display: true,
        color: '#fff'
      }
    },
    scales: {
      x: {
        ticks: { color: '#fff' },
        grid: { color: '#374151' }
      },
      y: {
        ticks: { color: '#fff' },
        grid: { color: '#374151' }
      }
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-6">
      <div className="max-w-7xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold mb-2">Dashboard Metriks Skripsi</h1>
          <p className="text-gray-400">Prototipe Asisten Belanja Siaran Langsung Berbasis YOLO, CLIP dan FAISS</p>
        </div>

        {/* Tab Navigation */}
        <div className="flex space-x-4 mb-8">
          {['overview', 'yolo', 'clip', 'system', 'thesis'].map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 rounded-lg font-medium ${
                activeTab === tab
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        {/* Download Buttons */}
        <div className="mb-8 flex space-x-4">
          <button
            onClick={() => downloadCSV('yolo')}
            className="bg-blue-600 hover:bg-blue-700 px-4 py-2 rounded-lg"
          >
            Download YOLO Data
          </button>
          <button
            onClick={() => downloadCSV('clip')}
            className="bg-purple-600 hover:bg-purple-700 px-4 py-2 rounded-lg"
          >
            Download CLIP Data
          </button>
          <button
            onClick={() => downloadCSV('system')}
            className="bg-green-600 hover:bg-green-700 px-4 py-2 rounded-lg"
          >
            Download System Data
          </button>
          <button
            onClick={() => downloadCSV('all')}
            className="bg-red-600 hover:bg-red-700 px-4 py-2 rounded-lg"
          >
            Download All Data
          </button>
          <button
            onClick={generateThesisResults}
            className="bg-yellow-600 hover:bg-yellow-700 px-4 py-2 rounded-lg font-semibold"
          >
            🎓 Generate Thesis Results
          </button>
        </div>

        {/* Content based on active tab */}
        {activeTab === 'overview' && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-lg font-semibold mb-2">Total Frames</h3>
              <p className="text-3xl font-bold text-blue-400">
                {metricsData?.summary?.total_frames_processed || 0}
              </p>
            </div>
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-lg font-semibold mb-2">Avg YOLO Latency</h3>
              <p className="text-3xl font-bold text-green-400">
                {metricsData?.summary?.avg_yolo_latency_ms?.toFixed(1) || 0}ms
              </p>
            </div>
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-lg font-semibold mb-2">CLIP Top-1 Acc</h3>
              <p className="text-3xl font-bold text-purple-400">
                {((metricsData?.summary?.clip_top1_accuracy || 0) * 100).toFixed(1)}%
              </p>
            </div>
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-lg font-semibold mb-2">System Latency</h3>
              <p className="text-3xl font-bold text-yellow-400">
                {metricsData?.summary?.avg_system_latency_ms?.toFixed(1) || 0}ms
              </p>
            </div>
          </div>
        )}

        {activeTab === 'yolo' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-xl font-semibold mb-4">YOLO Performance Metrics</h3>
              <Bar data={yoloPerformanceChart} options={{...chartOptions, plugins: {...chartOptions.plugins, title: {display: true, text: 'YOLO Detection Performance', color: '#fff'}}}} />
            </div>
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-xl font-semibold mb-4">Category Performance (AP@0.5)</h3>
              {thesisData?.category_performance && (
                <div className="space-y-3">
                  {thesisData.category_performance.map((cat, idx) => (
                    <div key={idx} className="flex justify-between items-center">
                      <span className="text-sm">{cat.category}</span>
                      <div className="flex items-center space-x-2">
                        <div className="w-32 bg-gray-700 rounded-full h-2">
                          <div 
                            className="bg-blue-500 h-2 rounded-full" 
                            style={{width: `${cat.ap_05 * 100}%`}}
                          ></div>
                        </div>
                        <span className="text-sm font-mono">{(cat.ap_05 * 100).toFixed(1)}%</span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {activeTab === 'clip' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-xl font-semibold mb-4">CLIP Top-K Accuracy</h3>
              <Bar data={clipPerformanceChart} options={{...chartOptions, plugins: {...chartOptions.plugins, title: {display: true, text: 'CLIP Recognition Accuracy', color: '#fff'}}}} />
            </div>
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-xl font-semibold mb-4">Similarity Score Distribution</h3>
              <div className="space-y-4">
                <div className="flex justify-between">
                  <span>Average Similarity</span>
                  <span className="font-mono">{thesisData?.clip_performance?.avg_similarity || 0}</span>
                </div>
                <div className="flex justify-between">
                  <span>Top-1 Accuracy</span>
                  <span className="font-mono">{((thesisData?.clip_performance?.top1_accuracy || 0) * 100).toFixed(1)}%</span>
                </div>
                <div className="flex justify-between">
                  <span>Top-3 Accuracy</span>
                  <span className="font-mono">{((thesisData?.clip_performance?.top3_accuracy || 0) * 100).toFixed(1)}%</span>
                </div>
                <div className="flex justify-between">
                  <span>Top-5 Accuracy</span>
                  <span className="font-mono">{((thesisData?.clip_performance?.top5_accuracy || 0) * 100).toFixed(1)}%</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'system' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-xl font-semibold mb-4">Latency Breakdown</h3>
              <Doughnut data={latencyChart} options={{...chartOptions, plugins: {...chartOptions.plugins, title: {display: true, text: 'System Latency Components', color: '#fff'}}}} />
            </div>
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-xl font-semibold mb-4">Performance Summary</h3>
              <div className="space-y-4">
                <div className="flex justify-between">
                  <span>Total Latency</span>
                  <span className="font-mono">{thesisData?.latency_breakdown?.total_ms || 0} ms</span>
                </div>
                <div className="flex justify-between">
                  <span>System FPS</span>
                  <span className="font-mono">{thesisData?.latency_breakdown?.fps || 0}</span>
                </div>
                <div className="flex justify-between">
                  <span>YOLO Inference</span>
                  <span className="font-mono">{thesisData?.latency_breakdown?.yolo_inference_ms || 0} ms</span>
                </div>
                <div className="flex justify-between">
                  <span>CLIP Embedding</span>
                  <span className="font-mono">{thesisData?.latency_breakdown?.clip_embedding_ms || 0} ms</span>
                </div>
                <div className="flex justify-between">
                  <span>FAISS Search</span>
                  <span className="font-mono">{thesisData?.latency_breakdown?.faiss_search_ms || 0} ms</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'thesis' && (
          <div className="space-y-8">
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-xl font-semibold mb-4">Perbandingan Skenario Eksperimental</h3>
              <Bar data={scenarioChart} options={{...chartOptions, plugins: {...chartOptions.plugins, title: {display: true, text: 'YOLO-only vs Hybrid System Performance', color: '#fff'}}}} />
            </div>
            
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
              <div className="bg-gray-800 p-6 rounded-lg">
                <h3 className="text-xl font-semibold mb-4">Tabel 4.1: Metrik Kinerja YOLOv11n</h3>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gray-600">
                        <th className="text-left py-2">Metrik</th>
                        <th className="text-right py-2">Nilai</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr><td className="py-1">mAP@0.5</td><td className="text-right font-mono">{((thesisData?.yolo_performance?.map_05 || 0) * 100).toFixed(1)}%</td></tr>
                      <tr><td className="py-1">mAP@0.5:0.95</td><td className="text-right font-mono">{((thesisData?.yolo_performance?.map_05_095 || 0) * 100).toFixed(1)}%</td></tr>
                      <tr><td className="py-1">Precision</td><td className="text-right font-mono">{((thesisData?.yolo_performance?.precision || 0) * 100).toFixed(1)}%</td></tr>
                      <tr><td className="py-1">Recall</td><td className="text-right font-mono">{((thesisData?.yolo_performance?.recall || 0) * 100).toFixed(1)}%</td></tr>
                    </tbody>
                  </table>
                </div>
              </div>

              <div className="bg-gray-800 p-6 rounded-lg">
                <h3 className="text-xl font-semibold mb-4">Tabel 4.4: Analisis Latency Breakdown</h3>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gray-600">
                        <th className="text-left py-2">Komponen</th>
                        <th className="text-right py-2">Latency (ms)</th>
                        <th className="text-right py-2">Persentase</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr><td className="py-1">YOLO Inference</td><td className="text-right font-mono">{thesisData?.latency_breakdown?.yolo_inference_ms || 0}</td><td className="text-right font-mono">36.6%</td></tr>
                      <tr><td className="py-1">CLIP Embedding</td><td className="text-right font-mono">{thesisData?.latency_breakdown?.clip_embedding_ms || 0}</td><td className="text-right font-mono">48.4%</td></tr>
                      <tr><td className="py-1">FAISS Search</td><td className="text-right font-mono">{thesisData?.latency_breakdown?.faiss_search_ms || 0}</td><td className="text-right font-mono">1.6%</td></tr>
                      <tr><td className="py-1">Overhead</td><td className="text-right font-mono">{thesisData?.latency_breakdown?.overhead_ms || 0}</td><td className="text-right font-mono">13.4%</td></tr>
                      <tr className="border-t border-gray-600 font-semibold"><td className="py-1">Total</td><td className="text-right font-mono">{thesisData?.latency_breakdown?.total_ms || 0}</td><td className="text-right font-mono">100%</td></tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default MetricsDashboard;