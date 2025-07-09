import React, { createContext, useContext, useState, ReactNode } from 'react'
import Toast from '../components/Toast'

interface ToastData {
  id: string
  message: string
  type: 'success' | 'error' | 'warning' | 'info'
  duration?: number
}

interface ToastContextType {
  showToast: (message: string, type: 'success' | 'error' | 'warning' | 'info', duration?: number) => void
  showSuccess: (message: string, duration?: number) => void
  showError: (message: string, duration?: number) => void
  showWarning: (message: string, duration?: number) => void
  showInfo: (message: string, duration?: number) => void
}

const ToastContext = createContext<ToastContextType | undefined>(undefined)

export const useToast = () => {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider')
  }
  return context
}

interface ToastProviderProps {
  children: ReactNode
}

export const ToastProvider: React.FC<ToastProviderProps> = ({ children }) => {
  const [toasts, setToasts] = useState<ToastData[]>([])

  const showToast = (message: string, type: 'success' | 'error' | 'warning' | 'info', duration = 3000) => {
    const id = Math.random().toString(36).substr(2, 9)
    const newToast: ToastData = { id, message, type, duration }
    
    setToasts(prev => [...prev, newToast])
  }

  const removeToast = (id: string) => {
    setToasts(prev => prev.filter(toast => toast.id !== id))
  }

  const showSuccess = (message: string, duration = 3000) => showToast(message, 'success', duration)
  const showError = (message: string, duration = 5000) => showToast(message, 'error', duration)
  const showWarning = (message: string, duration = 4000) => showToast(message, 'warning', duration)
  const showInfo = (message: string, duration = 3000) => showToast(message, 'info', duration)

  const value = {
    showToast,
    showSuccess,
    showError,
    showWarning,
    showInfo
  }

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-container">
        {toasts.map(toast => (
          <Toast
            key={toast.id}
            message={toast.message}
            type={toast.type}
            duration={toast.duration}
            onClose={() => removeToast(toast.id)}
          />
        ))}
      </div>
    </ToastContext.Provider>
  )
}
