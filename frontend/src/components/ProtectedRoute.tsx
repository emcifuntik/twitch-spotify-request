import React, { ReactNode } from 'react'
import { useAuth } from '../contexts/AuthContext'
import LoadingSpinner from './LoadingSpinner'

interface ProtectedRouteProps {
  children: ReactNode;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
  const { isAuthenticated, loading, login } = useAuth()

  if (loading) {
    return <LoadingSpinner message="Checking authentication..." />
  }

  if (!isAuthenticated) {
    return (
      <div className="auth-required">
        <div className="auth-required-content">
          <h2>Authentication Required</h2>
          <p>You need to login to access this page.</p>
          <button onClick={login} className="btn btn-primary">
            Login with Twitch
          </button>
        </div>
      </div>
    )
  }

  return <>{children}</>
}

export default ProtectedRoute
