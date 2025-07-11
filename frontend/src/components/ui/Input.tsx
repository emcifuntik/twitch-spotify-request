import React from 'react'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  helperText?: string
  variant?: 'default' | 'search'
  leftIcon?: React.ReactNode
  rightIcon?: React.ReactNode
}

const Input: React.FC<InputProps> = ({
  label,
  error,
  helperText,
  variant = 'default',
  leftIcon,
  rightIcon,
  className = '',
  ...props
}) => {
  const baseClass = 'twitch-input'
  const variantClass = `twitch-input--${variant}`
  const errorClass = error ? 'twitch-input--error' : ''
  const classes = [baseClass, variantClass, errorClass, className].filter(Boolean).join(' ')

  return (
    <div className="twitch-input-wrapper">
      {label && (
        <label className="twitch-input-label">
          {label}
        </label>
      )}
      <div className="twitch-input-container">
        {leftIcon && <div className="twitch-input-icon twitch-input-icon--left">{leftIcon}</div>}
        <input
          {...props}
          className={classes}
        />
        {rightIcon && <div className="twitch-input-icon twitch-input-icon--right">{rightIcon}</div>}
      </div>
      {error && (
        <div className="twitch-input-error">
          {error}
        </div>
      )}
      {helperText && !error && (
        <div className="twitch-input-helper">
          {helperText}
        </div>
      )}
    </div>
  )
}

export default Input
