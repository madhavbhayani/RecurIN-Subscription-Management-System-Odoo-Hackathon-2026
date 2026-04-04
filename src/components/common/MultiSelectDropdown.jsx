import { useEffect, useMemo, useRef, useState } from 'react'
import { FiCheck, FiChevronDown } from 'react-icons/fi'

function MultiSelectDropdown({
  buttonLabel,
  options,
  selectedValues,
  onChange,
  placeholder,
  emptyMessage,
}) {
  const [isOpen, setIsOpen] = useState(false)
  const rootRef = useRef(null)

  const selectedSet = useMemo(() => new Set(selectedValues), [selectedValues])
  const selectedCount = selectedValues.length

  useEffect(() => {
    const handleDocumentMouseDown = (event) => {
      if (!rootRef.current?.contains(event.target)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleDocumentMouseDown)
    return () => {
      document.removeEventListener('mousedown', handleDocumentMouseDown)
    }
  }, [])

  const toggleOption = (value) => {
    if (selectedSet.has(value)) {
      onChange(selectedValues.filter((selectedValue) => selectedValue !== value))
      return
    }

    onChange([...selectedValues, value])
  }

  return (
    <div className="relative" ref={rootRef}>
      <button
        type="button"
        onClick={() => setIsOpen((previousState) => !previousState)}
        className="inline-flex h-11 w-full items-center justify-between rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 text-sm text-[var(--navy)] outline-none transition-colors duration-200 hover:bg-[rgba(0,0,128,0.03)] focus:border-[var(--orange)]"
        aria-haspopup="listbox"
        aria-expanded={isOpen}
      >
        <span className="truncate text-left">
          {selectedCount > 0 ? `${buttonLabel}: ${selectedCount} selected` : placeholder}
        </span>
        <FiChevronDown className={`h-4 w-4 flex-none transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {isOpen && (
        <div className="absolute left-0 right-0 z-20 mt-2 max-h-60 overflow-y-auto rounded-lg border border-[color:rgba(0,0,128,0.18)] bg-[var(--white)] p-1 shadow-[0_14px_30px_rgba(0,0,0,0.15)]">
          {options.length === 0 ? (
            <div className="px-3 py-2 text-xs text-[color:rgba(0,0,128,0.62)]">{emptyMessage}</div>
          ) : (
            options.map((option) => {
              const isSelected = selectedSet.has(option.value)

              return (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => toggleOption(option.value)}
                  className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors duration-150 ${
                    isSelected
                      ? 'bg-[rgba(0,0,128,0.08)] text-[var(--navy)]'
                      : 'text-[var(--navy)] hover:bg-[rgba(0,0,128,0.05)]'
                  }`}
                >
                  <span className={`inline-flex h-4 w-4 items-center justify-center rounded border ${
                    isSelected
                      ? 'border-[var(--orange)] bg-[var(--orange)] text-white'
                      : 'border-[color:rgba(0,0,128,0.3)] bg-white text-transparent'
                  }`}>
                    <FiCheck className="h-3 w-3" />
                  </span>
                  <span className="truncate">{option.label}</span>
                </button>
              )
            })
          )}
        </div>
      )}
    </div>
  )
}

MultiSelectDropdown.defaultProps = {
  placeholder: 'Select options',
  emptyMessage: 'No options available.',
}

export default MultiSelectDropdown
