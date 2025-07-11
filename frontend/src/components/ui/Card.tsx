import React from 'react'

export interface CardProps {
  children: React.ReactNode
  className?: string
  variant?: 'default' | 'elevated' | 'outlined' | 'flat'
  padding?: 'none' | 'small' | 'medium' | 'large'
}

const Card: React.FC<CardProps> = ({
  children,
  className = '',
  variant = 'default',
  padding = 'medium'
}) => {
  const baseClass = 'twitch-card'
  const variantClass = `twitch-card--${variant}`
  const paddingClass = padding !== 'medium' ? `twitch-card--padding-${padding}` : ''
  const classes = [baseClass, variantClass, paddingClass, className].filter(Boolean).join(' ')

  return (
    <div className={classes}>
      {children}
    </div>
  )
}

export default Card
