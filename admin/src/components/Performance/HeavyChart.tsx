import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'

const HeavyChart = () => {
  const [data, setData] = useState<number[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // 模拟数据加载
    const timer = setTimeout(() => {
      const newData = Array.from({ length: 20 }, () => Math.floor(Math.random() * 100))
      setData(newData)
      setLoading(false)
    }, 1000)

    return () => clearTimeout(timer)
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-32">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    )
  }

  const maxValue = Math.max(...data)
  const minValue = Math.min(...data)

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      className="w-full h-32 bg-gradient-to-r from-blue-50 to-purple-50 rounded-lg p-4"
    >
      <div className="flex items-end justify-between h-full space-x-1">
        {data.map((value, index) => {
          const height = ((value - minValue) / (maxValue - minValue)) * 100
          return (
            <motion.div
              key={index}
              initial={{ height: 0 }}
              animate={{ height: `${height}%` }}
              transition={{ delay: index * 0.05, duration: 0.5 }}
              className="bg-gradient-to-t from-blue-500 to-purple-500 rounded-t flex-1"
            />
          )
        })}
      </div>
      <div className="text-center text-xs text-gray-600 mt-2">
        模拟图表数据 (懒加载)
      </div>
    </motion.div>
  )
}

export default HeavyChart
