import React from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { Button } from './ui'

const Navbar: React.FC = () => {
  const location = useLocation()
  const { user, login, logout } = useAuth()

  const isActive = (path: string): boolean => location.pathname === path

  const handleLogin = (): void => {
    login()
  }

  const handleLogout = (): void => {
    logout()
  }

  return (
    <nav className="navbar">
      <div className="container">
        <div className="flex justify-between align-center">
          <div className="flex align-center gap-2">
            <Link to="/" className="nav-brand">
              <img src="/static/favicon.png" alt="CatJamMusic" className="nav-logo" />
              CatJamMusic
            </Link>
          </div>
          
          <div className="flex align-center gap-1">
            <Link 
              to="/" 
              className={`nav-link ${isActive('/') ? 'active' : ''}`}
            >
              Home
            </Link>
            <Link 
              to="/dashboard" 
              className={`nav-link ${isActive('/dashboard') ? 'active' : ''}`}
            >
              Dashboard
            </Link>
            
            {user ? (
              <div className="flex align-center gap-2">
                <span className="text-secondary">Welcome, {user.name}</span>
                <Button 
                  onClick={handleLogout} 
                  variant="secondary"
                  size="small"
                >
                  Logout
                </Button>
              </div>
            ) : (
              <Button 
                onClick={handleLogin} 
                variant="primary"
              >
                <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                  <path d="M8 0L0 8l8 8 8-8-8-8zM7 11H5V9h2v2zm0-4H5V5h2v2z"/>
                </svg>
                Login with Twitch
              </Button>
            )}
          </div>
        </div>
      </div>
    </nav>
  )
}

export default Navbar
