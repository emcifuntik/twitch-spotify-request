import React from 'react'
import { LoadingSpinnerProps } from '../types'

const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({ message = 'Loading...' }) => {
  return (
    <div className="loading">
      <div className="spinner"></div>
      {message}
    </div>
  )
}

export default LoadingSpinner
