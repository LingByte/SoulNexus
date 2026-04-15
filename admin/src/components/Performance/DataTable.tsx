import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'

interface TableData {
  id: number
  name: string
  value: number
  status: 'active' | 'inactive'
}

const DataTable = () => {
  const [data, setData] = useState<TableData[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // 模拟数据加载
    const timer = setTimeout(() => {
      const newData: TableData[] = Array.from({ length: 10 }, (_, i) => ({
        id: i + 1,
        name: `项目 ${i + 1}`,
        value: Math.floor(Math.random() * 1000),
        status: Math.random() > 0.5 ? 'active' : 'inactive'
      }))
      setData(newData)
      setLoading(false)
    }, 800)

    return () => clearTimeout(timer)
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-32">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    )
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="w-full"
    >
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b">
              <th className="text-left p-2">ID</th>
              <th className="text-left p-2">名称</th>
              <th className="text-left p-2">值</th>
              <th className="text-left p-2">状态</th>
            </tr>
          </thead>
          <tbody>
            {data.map((item, index) => (
              <motion.tr
                key={item.id}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.05 }}
                className="border-b hover:bg-gray-50"
              >
                <td className="p-2">{item.id}</td>
                <td className="p-2">{item.name}</td>
                <td className="p-2">{item.value}</td>
                <td className="p-2">
                  <span className={`px-2 py-1 rounded-full text-xs ${
                    item.status === 'active' 
                      ? 'bg-green-100 text-green-800' 
                      : 'bg-gray-100 text-gray-800'
                  }`}>
                    {item.status === 'active' ? '活跃' : '非活跃'}
                  </span>
                </td>
              </motion.tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="text-center text-xs text-gray-600 mt-2">
        模拟数据表格 (懒加载)
      </div>
    </motion.div>
  )
}

export default DataTable
