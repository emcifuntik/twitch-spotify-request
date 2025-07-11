import React from 'react'

export interface SearchableDropdownProps {
  placeholder?: string
  value: string
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  onClear?: () => void
  loading?: boolean
  error?: string
  disabled?: boolean
  showDropdown?: boolean
  children?: React.ReactNode
  className?: string
}

const SearchableDropdown = React.forwardRef<HTMLDivElement, SearchableDropdownProps>(({
  placeholder = 'Search...',
  value,
  onChange,
  onClear,
  loading = false,
  error,
  disabled = false,
  showDropdown = false,
  children,
  className = ''
}, ref) => {
  return (
    <div className={`twitch-searchable-dropdown ${className}`} ref={ref}>
      <div className="twitch-input-container">
        <input
          type="text"
          className="twitch-input twitch-input--search"
          placeholder={placeholder}
          value={value}
          onChange={onChange}
          disabled={disabled}
          autoComplete="off"
        />
        
        {value && onClear && (
          <button
            type="button"
            onClick={onClear}
            className="clear-button"
            disabled={disabled}
          >
            ✕
          </button>
        )}
        
        {loading && (
          <div className="search-loading">
            <div className="spinner-small"></div>
          </div>
        )}
      </div>

      {showDropdown && children && (
        <div className="search-dropdown">
          {children}
        </div>
      )}

      {error && (
        <div className="search-error">
          {error}
        </div>
      )}
    </div>
  )
})

SearchableDropdown.displayName = 'SearchableDropdown'

export default SearchableDropdown
