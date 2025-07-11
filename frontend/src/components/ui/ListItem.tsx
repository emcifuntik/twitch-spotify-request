import React from 'react'

export interface ListItemProps {
  children: React.ReactNode
  onClick?: () => void
  className?: string
  variant?: 'default' | 'interactive'
  leftContent?: React.ReactNode
  rightContent?: React.ReactNode
  left?: React.ReactNode
  right?: React.ReactNode
  interactive?: boolean
  clickable?: boolean
}

const ListItem: React.FC<ListItemProps> = ({
  children,
  onClick,
  className = '',
  variant = 'default',
  leftContent,
  rightContent,
  left,
  right,
  interactive = false,
  clickable = false
}) => {
  const baseClass = 'twitch-list-item'
  const variantClass = variant === 'interactive' || interactive ? 'twitch-list-item--interactive' : ''
  const clickableClass = clickable ? 'twitch-list-item--clickable' : ''
  
  const combinedClass = `${baseClass} ${variantClass} ${clickableClass} ${className}`.trim()
  
  const leftElement = left || leftContent
  const rightElement = right || rightContent

  const ItemComponent = onClick ? 'button' : 'div'

  return (
    <ItemComponent
      className={combinedClass}
      onClick={onClick}
    >
      {leftElement && <div className="twitch-list-item-left">{leftElement}</div>}
      <div className="twitch-list-item-content">{children}</div>
      {rightElement && <div className="twitch-list-item-right">{rightElement}</div>}
    </ItemComponent>
  )
}

export default ListItem
