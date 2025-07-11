import React from 'react'

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'success' | 'danger' | 'ghost'
  size?: 'small' | 'medium' | 'large'
  loading?: boolean
  children: React.ReactNode
}

const Button: React.FC<ButtonProps> = ({
  variant = 'primary',
  size = 'medium',
  loading = false,
  children,
  className = '',
  disabled,
  ...props
}) => {
  const baseClass = 'btn'
  const variantClass = `btn-${variant}`
  const sizeClass = size !== 'medium' ? `btn-${size}` : ''
  const classes = [baseClass, variantClass, sizeClass, className].filter(Boolean).join(' ')

  return (
    <button
      {...props}
      className={classes}
      disabled={disabled || loading}
    >
      {loading ? (
        <span className="flex align-center gap-2">
          <div className="spinner-small"></div>
          {children}
        </span>
      ) : (
        children
      )}
    </button>
  )
}

export default Button
