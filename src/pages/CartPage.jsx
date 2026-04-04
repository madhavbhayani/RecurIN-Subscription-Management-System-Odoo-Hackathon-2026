import { Link } from 'react-router-dom'

const cartItems = [
  {
    name: 'Starter Recurring Plan',
    quantity: 1,
    amount: 'INR 999.00',
  },
  {
    name: 'Premium Support Add-on',
    quantity: 1,
    amount: 'INR 299.00',
  },
]

function CartPage() {
  return (
    <div className="w-full px-4 py-8 sm:px-6 lg:px-8">
      <section className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-6 shadow-[0_8px_24px_rgba(0,0,128,0.08)] sm:p-8">
        <h1 className="text-3xl font-bold text-[var(--navy)] sm:text-4xl">Cart</h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.8)] sm:text-base">
          Review your selected subscription products before checkout.
        </p>

        <div className="mt-6 overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.12)]">
          <div className="grid grid-cols-[1.8fr_0.6fr_0.8fr] gap-3 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
            <span>Item</span>
            <span>Qty</span>
            <span>Amount</span>
          </div>

          <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
            {cartItems.map((item) => (
              <div key={item.name} className="grid grid-cols-[1.8fr_0.6fr_0.8fr] gap-3 px-4 py-3 text-sm text-[var(--navy)]">
                <span className="font-semibold">{item.name}</span>
                <span>{item.quantity}</span>
                <span>{item.amount}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="mt-6 flex flex-wrap gap-3">
          <Link
            to="/shop"
            className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-300 hover:border-[var(--orange)] hover:text-[var(--orange)]"
          >
            Continue Shopping
          </Link>
          <Link
            to="/signup"
            className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
          >
            Proceed to Checkout
          </Link>
        </div>
      </section>
    </div>
  )
}

export default CartPage
