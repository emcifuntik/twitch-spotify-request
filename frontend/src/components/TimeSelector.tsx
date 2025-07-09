import React from 'react'

interface TimeSelectorProps {
  value: number; // Time in seconds
  onChange: (seconds: number) => void;
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
}

const TimeSelector: React.FC<TimeSelectorProps> = ({
  value,
  onChange,
  maxHours = 23,
  maxMinutes = 59,
  maxSeconds = 59,
  minSeconds = 0,
  disabled = false,
  id,
  className = '',
  showHours = true,
  showMinutes = true,
  showSeconds = true
}) => {
  // Convert seconds to hours, minutes, seconds
  const hours = Math.floor(value / 3600)
  const minutes = Math.floor((value % 3600) / 60)
  const seconds = value % 60

  // Calculate max values based on current selection
  const getMaxMinutes = () => {
    if (hours === maxHours) {
      const maxTotalSeconds = maxHours * 3600 + maxMinutes * 60 + maxSeconds
      const currentHourSeconds = hours * 3600
      return Math.floor((maxTotalSeconds - currentHourSeconds) / 60)
    }
    return 59
  }

  const getMaxSeconds = () => {
    if (hours === maxHours && minutes === getMaxMinutes()) {
      const maxTotalSeconds = maxHours * 3600 + maxMinutes * 60 + maxSeconds
      const currentTime = hours * 3600 + minutes * 60
      return maxTotalSeconds - currentTime
    }
    return 59
  }

  const handleTimeChange = (newHours: number, newMinutes: number, newSeconds: number) => {
    const totalSeconds = newHours * 3600 + newMinutes * 60 + newSeconds
    const maxTotalSeconds = maxHours * 3600 + maxMinutes * 60 + maxSeconds
    
    if (totalSeconds >= minSeconds && totalSeconds <= maxTotalSeconds) {
      onChange(totalSeconds)
    }
  }

  const handleHoursChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newHours = parseInt(e.target.value)
    handleTimeChange(newHours, minutes, seconds)
  }

  const handleMinutesChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newMinutes = parseInt(e.target.value)
    handleTimeChange(hours, newMinutes, seconds)
  }

  const handleSecondsChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newSeconds = parseInt(e.target.value)
    handleTimeChange(hours, minutes, newSeconds)
  }

  return (
    <div className={`time-selector ${className}`} id={id}>
      {showHours && (
        <div className="time-part">
          <select
            value={hours}
            onChange={handleHoursChange}
            disabled={disabled}
            className="time-select"
            aria-label="Hours"
          >
            {Array.from({ length: maxHours + 1 }, (_, i) => (
              <option key={i} value={i}>
                {i.toString().padStart(2, '0')}
              </option>
            ))}
          </select>
          <span className="time-label">h</span>
        </div>
      )}

      {showMinutes && (
        <div className="time-part">
          <select
            value={minutes}
            onChange={handleMinutesChange}
            disabled={disabled}
            className="time-select"
            aria-label="Minutes"
          >
            {Array.from({ length: Math.min(getMaxMinutes(), 59) + 1 }, (_, i) => (
              <option key={i} value={i}>
                {i.toString().padStart(2, '0')}
              </option>
            ))}
          </select>
          <span className="time-label">m</span>
        </div>
      )}

      {showSeconds && (
        <div className="time-part">
          <select
            value={seconds}
            onChange={handleSecondsChange}
            disabled={disabled}
            className="time-select"
            aria-label="Seconds"
          >
            {Array.from({ length: Math.min(getMaxSeconds(), 59) + 1 }, (_, i) => (
              <option key={i} value={i}>
                {i.toString().padStart(2, '0')}
              </option>
            ))}
          </select>
          <span className="time-label">s</span>
        </div>
      )}
    </div>
  )
}

export default TimeSelector
