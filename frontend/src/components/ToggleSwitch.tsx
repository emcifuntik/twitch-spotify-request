import React from 'react'

interface ToggleSwitchProps {
  leftLabel: string
  rightLabel: string
  value: boolean // true for right option, false for left option
  onChange: (value: boolean) => void
  disabled?: boolean
  className?: string
}

const ToggleSwitch: React.FC<ToggleSwitchProps> = ({
  leftLabel,
  rightLabel,
  value,
  onChange,
  disabled = false,
  className = ''
}) => {
  return (
    <div className={`toggle-switch ${className} ${disabled ? 'disabled' : ''}`}>
      <button
        type="button"
        className={`toggle-option ${!value ? 'active' : ''}`}
        onClick={() => !disabled && onChange(false)}
        disabled={disabled}
      >
        {leftLabel}
      </button>
      <button
        type="button"
        className={`toggle-option ${value ? 'active' : ''}`}
        onClick={() => !disabled && onChange(true)}
        disabled={disabled}
      >
        {rightLabel}
      </button>
    </div>
  )
}

export default ToggleSwitch
