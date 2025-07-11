import React from 'react'

export interface AvatarProps {
  src: string
  alt: string
  size?: 'small' | 'medium' | 'large'
  className?: string
}

const Avatar: React.FC<AvatarProps> = ({
  src,
  alt,
  size = 'medium',
  className = ''
}) => {
  const baseClass = 'twitch-avatar'
  const sizeClass = `twitch-avatar--${size}`
  const classes = [baseClass, sizeClass, className].filter(Boolean).join(' ')

  return (
    <img
      src={src}
      alt={alt}
      className={classes}
    />
  )
}

export default Avatar
