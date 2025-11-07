import React, { useState, useEffect } from 'react';
import { Line, Bar, Doughnut } from 'react-chartjs-2';
import { ArrowLeft, BarChart3, Download, GraduationCap, Activity, Cpu, Zap, Database } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { metricsAPI } from '../services/api';
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
  const navigate = useNavigate();
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
      const response = await metricsAPI.getMetrics();
      setMetricsData(response.data);
    } catch (error) {
      console.error('Error fetching metrics:', error);
    }
  };

  const fetchThesisData = async () => {
    try {
      const response = await metricsAPI.getThesisData();
      setThesisData(response.data);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching thesis data:', error);
      setLoading(false);
    }
  };

  const downloadCSV = (type) => {
    metricsAPI.downloadCSV(type);
  };

  const generateThesisResults = async () => {
    try {
      const response = await metricsAPI.generateThesisResults();
      const result = response.data;
      
      // Show success message with download options
      const files = result.files || [];
      const fileList = files.map(file => `• ${file}`).join('\n');
      
      if (confirm(`🎓 Thesis results generated successfully!\n\nFiles created:\n${fileList}\n\nLocation: ${result.location || 'thesis_results/'}\n\nWould you like to download the files?`)) {
        // Download each file
        files.forEach(filename => {
          setTimeout(() => {
            metricsAPI.downloadThesisFile(filename);
          }, 500); // Small delay between downloads
        });
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
      backgroundColor: ['#EF4444', '#10B981', '#F59E0B', '#6B7280'],
      borderColor: ['#DC2626', '#059669', '#D97706', '#4B5563'],
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
      backgroundColor: '#EF4444',
      borderColor: '#DC2626',
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
      backgroundColor: ['#EF4444', '#10B981', '#F59E0B', '#6B7280'],
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
        {/* Header with back button */}
        <div className="mb-8">
          <div className="flex items-center gap-4 mb-4">
            <button 
              onClick={() => navigate('/')}
              className="flex items-center gap-2 text-gray-400 hover:text-white transition-colors"
            >
              <ArrowLeft className="w-5 h-5" />
              <span>Back to Home</span>
            </button>
          </div>
          <div className="flex items-center gap-3 mb-4">
            <div className="w-12 h-12 bg-red-500 rounded-lg flex items-center justify-center">
              <BarChart3 className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-3xl font-bold">Thesis Metrics Dashboard</h1>
              <p className="text-gray-400">Prototipe Asisten Belanja Siaran Langsung Berbasis YOLO, CLIP dan FAISS</p>
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="flex flex-wrap gap-2 mb-8">
          {[
            { key: 'overview', label: 'Overview', icon: Activity },
            { key: 'yolo', label: 'YOLO', icon: Zap },
            { key: 'clip', label: 'CLIP', icon: Database },
            { key: 'system', label: 'System', icon: Cpu },
            { key: 'thesis', label: 'Thesis', icon: GraduationCap }
          ].map((tab) => {
            const IconComponent = tab.icon;
            return (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors ${
                  activeTab === tab.key
                    ? 'bg-red-500 text-white'
                    : 'bg-gray-800 text-gray-300 hover:bg-gray-700 border border-gray-700'
                }`}
              >
                <IconComponent className="w-4 h-4" />
                {tab.label}
              </button>
            );
          })}
        </div>

        {/* Action Buttons */}
        <div className="mb-8 flex flex-wrap gap-3">
          <button
            onClick={() => downloadCSV('yolo')}
            className="flex items-center gap-2 bg-gray-800 hover:bg-gray-700 border border-gray-700 px-4 py-2 rounded-lg transition-colors"
          >
            <Download className="w-4 h-4" />
            YOLO Data
          </button>
          <button
            onClick={() => downloadCSV('clip')}
            className="flex items-center gap-2 bg-gray-800 hover:bg-gray-700 border border-gray-700 px-4 py-2 rounded-lg transition-colors"
          >
            <Download className="w-4 h-4" />
            CLIP Data
          </button>
          <button
            onClick={() => downloadCSV('system')}
            className="flex items-center gap-2 bg-gray-800 hover:bg-gray-700 border border-gray-700 px-4 py-2 rounded-lg transition-colors"
          >
            <Download className="w-4 h-4" />
            System Data
          </button>
          <button
            onClick={() => downloadCSV('all')}
            className="flex items-center gap-2 bg-gray-800 hover:bg-gray-700 border border-gray-700 px-4 py-2 rounded-lg transition-colors"
          >
            <Download className="w-4 h-4" />
            All Data
          </button>
          <button
            onClick={generateThesisResults}
            className="flex items-center gap-2 bg-red-500 hover:bg-red-600 px-4 py-2 rounded-lg font-semibold transition-colors"
          >
            <GraduationCap className="w-4 h-4" />
            Generate Thesis Results
          </button>
        </div>

        {/* Content based on active tab */}
        {activeTab === 'overview' && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-lg font-semibold mb-2">Total Frames</h3>
              <p className="text-3xl font-bold text-red-400">
                {metricsData?.summary?.total_frames_processed || 0}
              </p>
            </div>
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-lg font-semibold mb-2">Avg YOLO Latency</h3>
              <p className="text-3xl font-bold text-red-400">
                {metricsData?.summary?.avg_yolo_latency_ms?.toFixed(1) || 0}ms
              </p>
            </div>
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-lg font-semibold mb-2">CLIP Top-1 Acc</h3>
              <p className="text-3xl font-bold text-red-400">
                {((metricsData?.summary?.clip_top1_accuracy || 0) * 100).toFixed(1)}%
              </p>
            </div>
            <div className="bg-gray-800 p-6 rounded-lg">
              <h3 className="text-lg font-semibold mb-2">System Latency</h3>
              <p className="text-3xl font-bold text-red-400">
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
                            className="bg-red-500 h-2 rounded-full" 
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