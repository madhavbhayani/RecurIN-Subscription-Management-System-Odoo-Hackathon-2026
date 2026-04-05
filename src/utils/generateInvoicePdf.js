import { jsPDF } from 'jspdf'
import autoTable from 'jspdf-autotable'

function drawRecurINLogo(doc, x, y) {
  // Orange rounded rectangle
  doc.setFillColor(255, 107, 0)
  doc.roundedRect(x, y, 14, 14, 3, 3, 'F')

  // White "R" inside the orange box
  doc.setTextColor(255, 255, 255)
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(12)
  doc.text('R', x + 4.8, y + 10)

  // "Recur" in orange
  doc.setTextColor(255, 107, 0)
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(18)
  doc.text('Recur', x + 18, y + 8)

  // "IN" in green
  doc.setTextColor(19, 136, 8)
  doc.text('IN', x + 43, y + 8)

  // "Subscription & Management" tagline
  doc.setTextColor(30, 44, 120)
  doc.setFont('helvetica', 'normal')
  doc.setFontSize(7)
  doc.text('SUBSCRIPTION & MANAGEMENT', x + 18, y + 13)
}

function formatCurrencyUSD(value) {
  const num = Number(value)
  if (!Number.isFinite(num)) return '$0.00'
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num)
}

/**
 * Generate and download a professional invoice PDF.
 * @param {Object} options
 * @param {string} options.subscriptionNumber
 * @param {string} options.customerName
 * @param {string} options.status
 * @param {string} options.date
 * @param {string} [options.recurringPlan]
 * @param {string} [options.paymentMethod]
 * @param {Array} options.products - [{product_name, quantity, unit_price, discount_amount, tax_amount, total_amount}]
 * @param {Object} [options.payment] - {amount_inr, paypal_status, payment_date}
 */
export function generateInvoicePdf(options = {}) {
  const {
    subscriptionNumber = 'N/A',
    customerName = 'Customer',
    status = 'Active',
    date = new Date().toLocaleDateString('en-IN'),
    recurringPlan = null,
    paymentMethod = 'PayPal',
    products = [],
    payment = null,
  } = options

  const doc = new jsPDF('p', 'mm', 'a4')

  // --- Header ---
  drawRecurINLogo(doc, 14, 12)

  // Separator line
  doc.setDrawColor(220, 223, 233)
  doc.line(14, 30, 196, 30)

  // --- Invoice Title ---
  doc.setTextColor(30, 44, 120)
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(16)
  doc.text('INVOICE', 14, 40)

  // --- Invoice Info ---
  doc.setFont('helvetica', 'normal')
  doc.setFontSize(10)
  doc.setTextColor(60, 70, 100)

  let infoY = 48
  doc.text(`Subscription: ${subscriptionNumber}`, 14, infoY)
  infoY += 6
  doc.text(`Customer: ${customerName}`, 14, infoY)
  infoY += 6
  doc.text(`Status: ${status}`, 14, infoY)
  infoY += 6
  doc.text(`Date: ${date}`, 14, infoY)
  if (recurringPlan) {
    infoY += 6
    doc.text(`Recurring Plan: ${recurringPlan}`, 14, infoY)
  }
  infoY += 6
  doc.text(`Payment Method: ${paymentMethod}`, 14, infoY)
  infoY += 10

  // --- Products Table ---
  if (products.length > 0) {
    const tableHeaders = [['Product', 'Qty', 'Unit Price', 'Discount', 'Tax', 'Total']]
    const tableBody = products.map((p) => [
      p.product_name || p.productName || '-',
      String(p.quantity ?? 1),
      formatCurrencyUSD(p.unit_price ?? p.unitPrice ?? 0),
      formatCurrencyUSD(p.discount_amount ?? p.discountAmount ?? 0),
      formatCurrencyUSD(p.tax_amount ?? p.taxAmount ?? 0),
      formatCurrencyUSD(p.total_amount ?? p.totalAmount ?? p.line_total ?? p.lineTotal ?? 0),
    ])

    const grandTotal = products.reduce((sum, p) => {
      return sum + Number(p.total_amount ?? p.totalAmount ?? p.line_total ?? p.lineTotal ?? 0)
    }, 0)

    autoTable(doc, {
      startY: infoY,
      head: tableHeaders,
      body: tableBody,
      theme: 'grid',
      headStyles: {
        fillColor: [30, 44, 120],
        textColor: [255, 255, 255],
        fontStyle: 'bold',
        fontSize: 9,
        halign: 'center',
      },
      bodyStyles: {
        fontSize: 9,
        textColor: [42, 51, 84],
      },
      columnStyles: {
        0: { cellWidth: 62, halign: 'left' },
        1: { cellWidth: 16, halign: 'center' },
        2: { cellWidth: 28, halign: 'right' },
        3: { cellWidth: 28, halign: 'right' },
        4: { cellWidth: 24, halign: 'right' },
        5: { cellWidth: 24, halign: 'right' },
      },
      margin: { left: 14, right: 14 },
    })

    // Grand total row
    const finalY = (doc.lastAutoTable?.finalY || infoY) + 2
    doc.setFont('helvetica', 'bold')
    doc.setFontSize(11)
    doc.setTextColor(30, 44, 120)
    doc.text('Grand Total:', 120, finalY + 6)
    doc.text(formatCurrencyUSD(grandTotal), 182, finalY + 6, { align: 'right' })

    // Payment details
    if (payment) {
      const payY = finalY + 16
      doc.setDrawColor(220, 223, 233)
      doc.line(14, payY - 2, 196, payY - 2)

      doc.setFont('helvetica', 'bold')
      doc.setFontSize(11)
      doc.text('Payment Details', 14, payY + 6)

      doc.setFont('helvetica', 'normal')
      doc.setFontSize(9)
      doc.setTextColor(60, 70, 100)
      doc.text(`Amount Paid: ${formatCurrencyUSD(payment.amount_inr ?? grandTotal)}`, 14, payY + 14)
      doc.text(`PayPal Status: ${payment.paypal_status || 'completed'}`, 14, payY + 20)
      doc.text(`Payment Date: ${payment.payment_date ? new Date(payment.payment_date).toLocaleDateString('en-IN') : date}`, 14, payY + 26)
    }
  }

  // --- Footer ---
  const pageHeight = doc.internal.pageSize.height
  doc.setFont('helvetica', 'normal')
  doc.setFontSize(8)
  doc.setTextColor(130, 140, 160)
  doc.text(
    'Thank you for choosing RecurIN. This invoice is auto-generated upon successful payment.',
    14,
    pageHeight - 16,
  )

  // Download
  doc.save(`Invoice-${subscriptionNumber}.pdf`)
}

export default generateInvoicePdf
