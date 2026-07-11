import { Star } from 'lucide-react'

interface MarketStarRatingProps {
  rating: number
  ratingCount?: number
  userRating?: number
  onRate?: (score: number) => void
  disabled?: boolean
}

export default function MarketStarRating({
  rating,
  ratingCount = 0,
  userRating,
  onRate,
  disabled,
}: MarketStarRatingProps) {
  const display = ratingCount > 0 ? rating : 0

  return (
    <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
      {[1, 2, 3, 4, 5].map((star) => {
        const filled = (userRating ?? display) >= star - 0.25
        return (
          <button
            key={star}
            type="button"
            disabled={disabled || !onRate}
            onClick={() => onRate?.(star)}
            className={`p-0.5 ${onRate && !disabled ? 'hover:scale-110 transition-transform cursor-pointer' : 'cursor-default'}`}
            title={onRate ? `评分 ${star}` : undefined}
          >
            <Star
              className={`w-3.5 h-3.5 ${filled ? 'fill-amber-400 text-amber-400' : 'text-gray-400'}`}
            />
          </button>
        )
      })}
      {ratingCount > 0 && (
        <span className="text-[10px] text-gray-400 ml-1">
          {display.toFixed(1)} ({ratingCount})
        </span>
      )}
    </div>
  )
}
