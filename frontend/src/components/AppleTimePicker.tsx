import React, { useState, useEffect, useRef } from 'react'

interface AppleTimePickerProps {
  value: number; // Time in seconds
  onChange: (seconds: number) => void;
  onChangeImmediate?: (seconds: number) => void; // For immediate UI updates
  maxHours?: number;
  maxMinutes?: number;
  maxSeconds?: number;
  minSeconds?: number;
  disabled?: boolean;
  id?: string;
  className?: string;
  showHours?: boolean;
  showMinutes?: boolean;
  showSeconds?: boolean;
  debounceMs?: number; // Debounce delay in milliseconds
  enableWheelScroll?: boolean; // Enable scroll wheel interaction
}

const AppleTimePicker: React.FC<AppleTimePickerProps> = ({
  value,
  onChange,
  onChangeImmediate,
  maxHours = 23,
  maxMinutes = 59,
  maxSeconds = 59,
  minSeconds = 0,
  disabled = false,
  id,
  className = '',
  showHours = true,
  showMinutes = true,
  showSeconds = true,
  debounceMs = 3000,
  enableWheelScroll = false
}) => {
  const hours = Math.floor(value / 3600)
  const minutes = Math.floor((value % 3600) / 60)
  const seconds = value % 60

  const hoursRef = useRef<HTMLDivElement>(null)
  const minutesRef = useRef<HTMLDivElement>(null)
  const secondsRef = useRef<HTMLDivElement>(null)

  const [isDragging, setIsDragging] = useState<string | null>(null)
  const [dragStart, setDragStart] = useState({ y: 0, value: 0 })
  const [pendingValue, setPendingValue] = useState<number | null>(null)
  const debounceRef = useRef<number | null>(null)

  const ITEM_HEIGHT = 40 // Height of each time item

  const handleTimeChange = (newHours: number, newMinutes: number, newSeconds: number) => {
    const totalSeconds = newHours * 3600 + newMinutes * 60 + newSeconds
    const maxTotalSeconds = maxHours * 3600 + maxMinutes * 60 + maxSeconds
    
    if (totalSeconds >= minSeconds && totalSeconds <= maxTotalSeconds) {
      // Immediate callback for UI updates (optional)
      if (onChangeImmediate) {
        onChangeImmediate(totalSeconds)
      }
      
      // Set pending value for debounced update
      setPendingValue(totalSeconds)
      
      // Clear existing timeout
      if (debounceRef.current !== null) {
        clearTimeout(debounceRef.current)
      }
      
      // Set new debounced timeout
      debounceRef.current = window.setTimeout(() => {
        onChange(totalSeconds)
        setPendingValue(null)
      }, debounceMs)
    }
  }

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current !== null) {
        clearTimeout(debounceRef.current)
      }
    }
  }, [])

  const handleMouseDown = (type: 'hours' | 'minutes' | 'seconds', e: React.MouseEvent) => {
    if (disabled) return
    
    e.preventDefault()
    setIsDragging(type)
    
    const currentValue = type === 'hours' ? hours : type === 'minutes' ? minutes : seconds
    setDragStart({
      y: e.clientY,
      value: currentValue
    })
  }

  const handleMouseMove = (e: MouseEvent) => {
    if (!isDragging || disabled) return

    const deltaY = dragStart.y - e.clientY
    const deltaValue = Math.round(deltaY / ITEM_HEIGHT)
    
    let maxValue = 59
    if (isDragging === 'hours') {
      maxValue = maxHours
    } else if (isDragging === 'minutes') {
      maxValue = hours === maxHours ? Math.floor((maxHours * 3600 + maxMinutes * 60 + maxSeconds - hours * 3600) / 60) : 59
    } else if (isDragging === 'seconds') {
      maxValue = hours === maxHours && minutes === Math.floor((maxHours * 3600 + maxMinutes * 60 + maxSeconds - hours * 3600) / 60) 
        ? (maxHours * 3600 + maxMinutes * 60 + maxSeconds) - (hours * 3600 + minutes * 60) 
        : 59
    }

    // Handle cyclical values
    const totalValues = maxValue + 1
    let newValue = (dragStart.value + deltaValue) % totalValues
    if (newValue < 0) {
      newValue = totalValues + newValue
    }

    if (isDragging === 'hours') {
      handleTimeChange(newValue, minutes, seconds)
    } else if (isDragging === 'minutes') {
      handleTimeChange(hours, newValue, seconds)
    } else if (isDragging === 'seconds') {
      handleTimeChange(hours, minutes, newValue)
    }
  }

  const handleMouseUp = () => {
    setIsDragging(null)
    
    // Force immediate update when dragging ends
    if (pendingValue !== null) {
      if (debounceRef.current !== null) {
        clearTimeout(debounceRef.current)
      }
      onChange(pendingValue)
      setPendingValue(null)
    }
  }

  const handleTouchStart = (type: 'hours' | 'minutes' | 'seconds', e: React.TouchEvent) => {
    if (disabled) return
    
    e.preventDefault()
    setIsDragging(type)
    
    const currentValue = type === 'hours' ? hours : type === 'minutes' ? minutes : seconds
    setDragStart({
      y: e.touches[0].clientY,
      value: currentValue
    })
  }

  const handleTouchMove = (e: TouchEvent) => {
    if (!isDragging || disabled) return

    const deltaY = dragStart.y - e.touches[0].clientY
    const deltaValue = Math.round(deltaY / ITEM_HEIGHT)
    
    let maxValue = 59
    if (isDragging === 'hours') {
      maxValue = maxHours
    } else if (isDragging === 'minutes') {
      maxValue = hours === maxHours ? Math.floor((maxHours * 3600 + maxMinutes * 60 + maxSeconds - hours * 3600) / 60) : 59
    } else if (isDragging === 'seconds') {
      maxValue = hours === maxHours && minutes === Math.floor((maxHours * 3600 + maxMinutes * 60 + maxSeconds - hours * 3600) / 60) 
        ? (maxHours * 3600 + maxMinutes * 60 + maxSeconds) - (hours * 3600 + minutes * 60) 
        : 59
    }

    // Handle cyclical values
    const totalValues = maxValue + 1
    let newValue = (dragStart.value + deltaValue) % totalValues
    if (newValue < 0) {
      newValue = totalValues + newValue
    }

    if (isDragging === 'hours') {
      handleTimeChange(newValue, minutes, seconds)
    } else if (isDragging === 'minutes') {
      handleTimeChange(hours, newValue, seconds)
    } else if (isDragging === 'seconds') {
      handleTimeChange(hours, minutes, newValue)
    }
  }

  const handleTouchEnd = () => {
    setIsDragging(null)
    
    // Force immediate update when touch ends
    if (pendingValue !== null) {
      if (debounceRef.current !== null) {
        clearTimeout(debounceRef.current)
      }
      onChange(pendingValue)
      setPendingValue(null)
    }
  }

  const handleWheel = (type: 'hours' | 'minutes' | 'seconds', e: React.WheelEvent) => {
    if (disabled || !enableWheelScroll) return
    
    e.preventDefault()
    
    const currentValue = type === 'hours' ? hours : type === 'minutes' ? minutes : seconds
    const delta = e.deltaY > 0 ? -1 : 1
    
    let maxValue = 59
    if (type === 'hours') {
      maxValue = maxHours
    } else if (type === 'minutes') {
      maxValue = hours === maxHours ? Math.floor((maxHours * 3600 + maxMinutes * 60 + maxSeconds - hours * 3600) / 60) : 59
    } else if (type === 'seconds') {
      maxValue = hours === maxHours && minutes === Math.floor((maxHours * 3600 + maxMinutes * 60 + maxSeconds - hours * 3600) / 60) 
        ? (maxHours * 3600 + maxMinutes * 60 + maxSeconds) - (hours * 3600 + minutes * 60) 
        : 59
    }

    // Handle cyclical values
    const totalValues = maxValue + 1
    let newValue = (currentValue + delta) % totalValues
    if (newValue < 0) {
      newValue = totalValues + newValue
    }

    if (type === 'hours') {
      handleTimeChange(newValue, minutes, seconds)
    } else if (type === 'minutes') {
      handleTimeChange(hours, newValue, seconds)
    } else if (type === 'seconds') {
      handleTimeChange(hours, minutes, newValue)
    }
  }

  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)
      document.addEventListener('touchmove', handleTouchMove)
      document.addEventListener('touchend', handleTouchEnd)
      document.body.style.cursor = 'grabbing'
      document.body.style.userSelect = 'none'
      
      return () => {
        document.removeEventListener('mousemove', handleMouseMove)
        document.removeEventListener('mouseup', handleMouseUp)
        document.removeEventListener('touchmove', handleTouchMove)
        document.removeEventListener('touchend', handleTouchEnd)
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
      }
    }
  }, [isDragging, dragStart])

  const renderTimeColumn = (
    type: 'hours' | 'minutes' | 'seconds',
    currentValue: number,
    maxValue: number,
    label: string
  ) => {
    // Create cyclical array with padding for smooth scrolling
    const totalItems = maxValue + 1
    const visibleItems = 5 // Show 2 items above and below current
    const paddingItems = Math.floor(visibleItems / 2) // 2 items above and below
    
    const allItems = []
    
    // Add padding items before (show previous values cyclically)
    for (let i = paddingItems; i > 0; i--) {
      const value = (currentValue - i + totalItems) % totalItems
      allItems.push({
        value,
        display: value.toString().padStart(2, '0'),
        isPadding: true
      })
    }
    
    // Add current item (this will be in the center)
    allItems.push({
      value: currentValue,
      display: currentValue.toString().padStart(2, '0'),
      isPadding: false
    })
    
    // Add padding items after (show next values cyclically)
    for (let i = 1; i <= paddingItems; i++) {
      const value = (currentValue + i) % totalItems
      allItems.push({
        value,
        display: value.toString().padStart(2, '0'),
        isPadding: true
      })
    }
    
    return (
      <div className="apple-time-column">
        <div className="apple-time-label">{label}</div>
        <div 
          className={`apple-time-wheel ${isDragging === type ? 'dragging' : ''} ${disabled ? 'disabled' : ''}`}
          onMouseDown={(e) => handleMouseDown(type, e)}
          onTouchStart={(e) => handleTouchStart(type, e)}
          onWheel={(e) => handleWheel(type, e)}
          ref={type === 'hours' ? hoursRef : type === 'minutes' ? minutesRef : secondsRef}
        >
          <div className="apple-time-selection-indicator" />
          <div 
            className="apple-time-list"
          >
            {allItems.map((item, index) => (
              <div
                key={`${item.value}-${index}`}
                className={`apple-time-item ${item.value === currentValue && !item.isPadding ? 'selected' : ''} ${item.isPadding ? 'padding' : ''}`}
              >
                {item.display}
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className={`apple-time-picker ${className} ${disabled ? 'disabled' : ''} ${pendingValue !== null ? 'pending-update' : ''}`} id={id}>
      {showHours && renderTimeColumn('hours', hours, maxHours, 'hours')}
      {showMinutes && renderTimeColumn('minutes', minutes, 
        hours === maxHours ? Math.floor((maxHours * 3600 + maxMinutes * 60 + maxSeconds - hours * 3600) / 60) : 59, 
        'min')}
      {showSeconds && renderTimeColumn('seconds', seconds,
        hours === maxHours && minutes === Math.floor((maxHours * 3600 + maxMinutes * 60 + maxSeconds - hours * 3600) / 60) 
          ? (maxHours * 3600 + maxMinutes * 60 + maxSeconds) - (hours * 3600 + minutes * 60) 
          : 59,
        'sec')}
    </div>
  )
}

export default AppleTimePicker
