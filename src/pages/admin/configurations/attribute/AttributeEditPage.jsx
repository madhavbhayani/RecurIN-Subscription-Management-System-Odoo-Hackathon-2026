import { useEffect, useState } from 'react'
import { FiPlus, FiTrash2 } from 'react-icons/fi'
import { Link, useParams } from 'react-router-dom'
import ToastMessage from '../../../../components/common/ToastMessage'
import { getAttributeById, updateAttribute } from '../../../../services/attributeApi'

function createValueRow() {
  return {
    attributeValue: '',
    defaultExtraPrice: '',
  }
}

function mapApiValuesToRows(values) {
  if (!Array.isArray(values) || values.length === 0) {
    return [createValueRow()]
  }

  return values.map((value) => ({
    attributeValue: String(value?.attribute_value ?? ''),
    defaultExtraPrice: String(value?.default_extra_price ?? ''),
  }))
}

function AttributeEditPage() {
  const { attributeId = '' } = useParams()

  const [attributeName, setAttributeName] = useState('')
  const [valueRows, setValueRows] = useState([createValueRow()])
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    let isMounted = true

    const loadAttribute = async () => {
      setIsLoading(true)
      try {
        const response = await getAttributeById(attributeId)
        if (!isMounted) {
          return
        }

        const attribute = response?.attribute
        setAttributeName(String(attribute?.attribute_name ?? ''))
        setValueRows(mapApiValuesToRows(attribute?.values))
      } catch (error) {
        if (!isMounted) {
          return
        }

        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    loadAttribute()

    return () => {
      isMounted = false
    }
  }, [attributeId])

  const handleRowChange = (index, field, value) => {
    setValueRows((previousRows) => previousRows.map((row, rowIndex) => {
      if (rowIndex !== index) {
        return row
      }

      return {
        ...row,
        [field]: value,
      }
    }))
  }

  const handleAddRow = () => {
    setValueRows((previousRows) => [...previousRows, createValueRow()])
  }

  const handleRemoveRow = (index) => {
    setValueRows((previousRows) => {
      if (previousRows.length === 1) {
        return previousRows
      }

      return previousRows.filter((_, rowIndex) => rowIndex !== index)
    })
  }

  const handleSubmit = async (event) => {
    event.preventDefault()

    setToastMessage('')

    const normalizedAttributeName = attributeName.trim()
    if (!normalizedAttributeName) {
      setToastVariant('error')
      setToastMessage('Attribute name is required.')
      return
    }

    if (valueRows.length === 0) {
      setToastVariant('error')
      setToastMessage('Please add at least one attribute value.')
      return
    }

    const normalizedValues = []
    for (const row of valueRows) {
      const normalizedValue = row.attributeValue.trim()
      if (!normalizedValue) {
        setToastVariant('error')
        setToastMessage('Attribute value is required for all rows.')
        return
      }

      const normalizedPriceText = String(row.defaultExtraPrice ?? '').trim()
      const normalizedPrice = normalizedPriceText === '' ? 0 : Number(normalizedPriceText)
      if (!Number.isFinite(normalizedPrice) || normalizedPrice < 0) {
        setToastVariant('error')
        setToastMessage('Default extra price must be a non-negative number.')
        return
      }

      normalizedValues.push({
        attribute_value: normalizedValue,
        default_extra_price: Number(normalizedPrice.toFixed(2)),
      })
    }

    setIsSubmitting(true)
    try {
      const response = await updateAttribute(attributeId, {
        attribute_name: normalizedAttributeName,
        values: normalizedValues,
      })

      setToastVariant('success')
      setToastMessage(response?.message ?? 'Attribute updated successfully.')

      const updatedAttribute = response?.attribute
      if (updatedAttribute) {
        setAttributeName(String(updatedAttribute.attribute_name ?? normalizedAttributeName))
        setValueRows(mapApiValuesToRows(updatedAttribute.values))
      }
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-8 sm:p-10">
      <ToastMessage
        message={toastMessage}
        variant={toastVariant}
        onClose={() => setToastMessage('')}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">
          Edit Attribute
        </h1>
        <Link
          to="/admin/configurations/attribute"
          className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
        >
          Back to Attributes
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Update the attribute name, values, and default extra pricing.
      </p>

      {isLoading ? (
        <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
          Loading attribute details...
        </div>
      ) : (
        <form className="mt-6 space-y-7" onSubmit={handleSubmit} noValidate>
          <div className="space-y-2">
            <label htmlFor="attribute-name" className="block text-sm font-semibold text-[var(--navy)]">
              Attribute Name
            </label>
            <input
              id="attribute-name"
              name="attribute-name"
              type="text"
              value={attributeName}
              onChange={(event) => setAttributeName(event.target.value)}
              placeholder="Enter attribute name"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.14)]">
            <div className="flex items-center justify-between gap-3 bg-[rgba(0,0,128,0.04)] px-5 py-3.5">
              <h2 className="text-sm font-semibold text-[var(--navy)]">Attribute Values and Default Extra Pricing</h2>
              <button
                type="button"
                onClick={handleAddRow}
                className="inline-flex h-10 w-10 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.06)]"
                aria-label="Add row"
                title="Add row"
              >
                <FiPlus className="h-4.5 w-4.5" />
              </button>
            </div>

            <div className="overflow-x-auto">
              <table className="min-w-full border-separate border-spacing-0 text-sm">
                <thead>
                  <tr>
                    <th className="border-b border-[color:rgba(0,0,128,0.12)] px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
                      Attribute Value
                    </th>
                    <th className="border-b border-[color:rgba(0,0,128,0.12)] px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
                      Default Extra Price
                    </th>
                    <th className="border-b border-[color:rgba(0,0,128,0.12)] px-4 py-3 text-right text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
                      Action
                    </th>
                  </tr>
                </thead>

                <tbody>
                  {valueRows.map((row, index) => (
                    <tr key={`attribute-edit-value-row-${index}`}>
                      <td className="border-b border-[color:rgba(0,0,128,0.08)] px-4 py-3">
                        <input
                          type="text"
                          value={row.attributeValue}
                          onChange={(event) => handleRowChange(index, 'attributeValue', event.target.value)}
                          placeholder="e.g. Premium"
                          className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-3 py-2 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
                        />
                      </td>
                      <td className="border-b border-[color:rgba(0,0,128,0.08)] px-4 py-3">
                        <input
                          type="number"
                          inputMode="decimal"
                          min="0"
                          step="0.01"
                          value={row.defaultExtraPrice}
                          onChange={(event) => handleRowChange(index, 'defaultExtraPrice', event.target.value)}
                          placeholder="0.00"
                          className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-3 py-2 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
                        />
                      </td>
                      <td className="border-b border-[color:rgba(0,0,128,0.08)] px-4 py-3 text-right">
                        <button
                          type="button"
                          onClick={() => handleRemoveRow(index)}
                          disabled={valueRows.length === 1}
                          className="inline-flex h-9 w-9 items-center justify-center rounded-lg text-[#c1292e] transition-colors duration-200 hover:bg-[#c1292e]/10 disabled:cursor-not-allowed disabled:opacity-40"
                          aria-label="Delete row"
                          title="Delete row"
                        >
                          <FiTrash2 className="h-4 w-4" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-end gap-3">
            <Link
              to="/admin/configurations/attribute"
              className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70"
            >
              {isSubmitting ? 'Updating...' : 'Update Attribute'}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}

export default AttributeEditPage
