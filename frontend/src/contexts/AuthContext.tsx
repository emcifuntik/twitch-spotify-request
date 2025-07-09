import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { UserProfile } from '../types'
import axios from 'axios'

interface AuthContextType {
  user: UserProfile | null;
  loading: boolean;
  login: () => void;
  logout: () => Promise<void>;
  refreshToken: () => Promise<boolean>;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState<boolean>(true)

  const isAuthenticated = !!user

  useEffect(() => {
    checkAuthStatus()
  }, [])

  const checkAuthStatus = async (): Promise<void> => {
    try {
      setLoading(true)
      const response = await axios.get('/api/user/current')
      
      if (response.data?.success) {
        setUser(response.data.data)
      } else {
        setUser(null)
      }
    } catch (error: any) {
      console.log('Not authenticated:', error.response?.status)
      setUser(null)
    } finally {
      setLoading(false)
    }
  }

  const login = (): void => {
    // Redirect to login endpoint
    window.location.href = '/login'
  }

  const logout = async (): Promise<void> => {
    try {
      await axios.post('/logout')
    } catch (error) {
      console.error('Logout error:', error)
    } finally {
      setUser(null)
      window.location.href = '/'
    }
  }

  const refreshToken = async (): Promise<boolean> => {
    try {
      const response = await axios.post('/refresh')
      if (response.data?.success) {
        // Re-check auth status to get updated user info
        await checkAuthStatus()
        return true
      }
    } catch (error) {
      console.error('Token refresh failed:', error)
      setUser(null)
    }
    return false
  }

  // Configure axios interceptors for automatic token refresh
  useEffect(() => {
    const responseInterceptor = axios.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config

        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true

          const refreshed = await refreshToken()
          if (refreshed) {
            return axios(originalRequest)
          } else {
            // Redirect to login if refresh fails
            window.location.href = '/login'
          }
        }

        return Promise.reject(error)
      }
    )

    return () => {
      axios.interceptors.response.eject(responseInterceptor)
    }
  }, [])

  const value: AuthContextType = {
    user,
    loading,
    login,
    logout,
    refreshToken,
    isAuthenticated,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

export default AuthContext
