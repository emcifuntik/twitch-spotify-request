import React from 'react'
import { BrowserRouter as Router, Routes, Route, useLocation } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import { ToastProvider } from './contexts/ToastContext'
import Navbar from './components/Navbar'
import Home from './pages/Home'
import Dashboard from './pages/Dashboard'
import Queue from './pages/Queue'
import QueueCompact from './pages/QueueCompact'
import ProtectedRoute from './components/ProtectedRoute'
import './App.css'

const AppContent: React.FC = () => {
  const location = useLocation()
  const isCompactRoute = location.pathname.includes('/queue-compact/')

  return (
    <div className={`App ${isCompactRoute ? 'compact-route' : ''}`}>
      {!isCompactRoute && <Navbar />}
      <main>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route 
            path="/dashboard" 
            element={
              <ProtectedRoute>
                <Dashboard />
              </ProtectedRoute>
            } 
          />
          <Route path="/queue/:streamerId" element={<Queue />} />
          <Route path="/queue-compact/:streamerId" element={<QueueCompact />} />
        </Routes>
      </main>
    </div>
  )
}

const App: React.FC = () => {
  return (
    <AuthProvider>
      <ToastProvider>
        <Router>
          <AppContent />
        </Router>
      </ToastProvider>
    </AuthProvider>
  )
}

export default App
