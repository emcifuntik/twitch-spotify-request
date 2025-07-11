import React from 'react'

export interface SpinnerProps {
  size?: 'small' | 'medium' | 'large'
  className?: string
}

const Spinner: React.FC<SpinnerProps> = ({
  size = 'medium',
  className = ''
}) => {
  const baseClass = 'twitch-spinner'
  const sizeClass = `twitch-spinner--${size}`
  const classes = [baseClass, sizeClass, className].filter(Boolean).join(' ')

  return (
    <div className={classes}>
      <div className="twitch-spinner-circle"></div>
    </div>
  )
}

export default Spinner
