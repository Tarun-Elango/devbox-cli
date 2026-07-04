export default function DocOutline({ items }) {
  if (!items?.length) return null

  return (
    <nav className="page-outline" aria-label="On this page">
      <span className="page-outline-label">On this page:</span>{' '}
      <ul>
        {items.map(({ id, label }) => (
          <li key={id}>
            <a href={`#${id}`}>{label}</a>
          </li>
        ))}
      </ul>
    </nav>
  )
}
