import React from 'react'

const OverviewSkeleton: React.FC = () => {
  return (
    <div className="w-full min-h-screen p-6 space-y-6 animate-pulse">
      {/* Header Skeleton */}
      <div className="h-48 bg-gradient-to-r from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-800 rounded-lg"></div>
      
      {/* Stats Cards Skeleton */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-32 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
        ))}
      </div>
      
      {/* Chart Skeleton */}
      <div className="h-64 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
      
      {/* Activity Feed Skeleton */}
      <div className="h-48 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
    </div>
  )
}

export default OverviewSkeleton

