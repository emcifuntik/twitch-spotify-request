import React from 'react'
import { StatusIndicatorProps } from '../types'

const StatusIndicator: React.FC<StatusIndicatorProps> = ({ status, label }) => {
  const getStatusClass = (): string => {
    switch (status) {
      case 'connected':
        return 'status-connected'
      case 'disconnected':
        return 'status-disconnected'
      case 'warning':
        return 'status-warning'
      default:
        return 'status-disconnected'
    }
  }

  const getStatusIcon = (): string => {
    switch (status) {
      case 'connected':
        return '✓'
      case 'disconnected':
        return '✗'
      case 'warning':
        return '⚠'
      default:
        return '?'
    }
  }

  return (
    <span className={`status-indicator ${getStatusClass()}`}>
      {getStatusIcon()} {label}
    </span>
  )
}

export default StatusIndicator
