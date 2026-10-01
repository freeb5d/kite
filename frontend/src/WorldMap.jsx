import { useMemo } from 'react'
import map from './assets/worldmap.json'

// Country outlines from Natural Earth (public domain), Mercator-projected
// and simplified into integer rings: { w, h, c: { "DE": [[x,y,x,y,...], ...] } }.

const ringPath = (r) => {
  let d = `M${r[0]} ${r[1]}`
  for (let i = 2; i < r.length; i += 2) d += `L${r[i]} ${r[i + 1]}`
  return d + 'Z'
}

const paths = Object.fromEntries(Object.entries(map.c).map(([code, rings]) => [code, rings.map(ringPath).join('')]))

// Bounding box of a country's largest ring, so overseas territories (e.g.
// France's) don't pull the view away from the mainland.
function mainBox(code) {
  const rings = map.c[code]
  if (!rings) return null
  let best = null
  for (const r of rings) {
    let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity
    for (let i = 0; i < r.length; i += 2) {
      x0 = Math.min(x0, r[i]); x1 = Math.max(x1, r[i])
      y0 = Math.min(y0, r[i + 1]); y1 = Math.max(y1, r[i + 1])
    }
    const area = (x1 - x0) * (y1 - y0)
    if (!best || area > best.area) best = { x0, y0, x1, y1, area }
  }
  return best
}

const ASPECT = 2.2

/** Map zoomed onto `country`, which is highlighted; `children` overlay it. */
export default function WorldMap({ country, children }) {
  const code = (country || '').toUpperCase()
  const viewBox = useMemo(() => {
    const b = mainBox(code)
    if (!b) return `0 0 ${map.w} ${map.w / ASPECT}`
    const cw = b.x1 - b.x0, ch = b.y1 - b.y0
    const w = Math.min(map.w, Math.max(cw * 3.2, ch * 3.2 * ASPECT, 240))
    const h = w / ASPECT
    // Keep the country left of centre: the info card sits on the right.
    const cx = (b.x0 + b.x1) / 2 + w * 0.16
    const cy = (b.y0 + b.y1) / 2
    const x = Math.max(0, Math.min(map.w - w, cx - w / 2))
    const y = Math.max(0, Math.min(map.h - h, cy - h / 2))
    return `${x} ${y} ${w} ${h}`
  }, [code])
  const stroke = useMemo(() => Number(viewBox.split(' ')[2]) / 700, [viewBox])

  return (
    <div className="relative w-full overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--map-sea)]" style={{ aspectRatio: ASPECT }}>
      <svg viewBox={viewBox} className="absolute inset-0 w-full h-full" preserveAspectRatio="xMidYMid slice" aria-hidden="true">
        {Object.entries(paths).map(([c, d]) =>
          c === code ? null : <path key={c} d={d} fill="var(--map-land)" stroke="var(--map-border)" strokeWidth={stroke} />,
        )}
        {paths[code] && (
          <path d={paths[code]} fill="var(--accent)" stroke="var(--map-highlight-border)" strokeWidth={stroke * 2.5} />
        )}
      </svg>
      {children}
    </div>
  )
}
