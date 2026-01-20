import React from 'react'
import { WidgetConfig } from '@/types/overview'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'

interface TableWidgetProps {
  config: WidgetConfig
  data?: {
    columns: string[]
    rows: any[][]
  }
}

const TableWidget: React.FC<TableWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  
  // 安全地获取 columns
  let columns: string[] = []
  if (data?.columns && Array.isArray(data.columns)) {
    columns = data.columns
  } else if (props?.columns && Array.isArray(props.columns)) {
    columns = props.columns
  } else {
    columns = ['列1', '列2', '列3']
  }

  // 安全地获取 rows
  let rows: any[][] = []
  if (data?.rows && Array.isArray(data.rows)) {
    rows = data.rows
  } else if (props?.rows && Array.isArray(props.rows)) {
    rows = props.rows
  } else {
    rows = [
      ['数据1', '数据2', '数据3'],
      ['数据4', '数据5', '数据6'],
    ]
  }

  return (
    <Card className="h-full" style={style}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="h-[calc(100%-60px)] overflow-auto">
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                {columns.map((col, index) => (
                  <th
                    key={index}
                    className="px-4 py-3 text-left font-medium text-muted-foreground"
                  >
                    {col}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, rowIndex) => (
                <tr
                  key={rowIndex}
                  className="border-b hover:bg-muted/30 transition-colors"
                >
                  {row.map((cell, cellIndex) => (
                    <td key={cellIndex} className="px-4 py-3">
                      {cell}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
          {rows.length === 0 && (
            <div className="p-8 text-center text-muted-foreground">
              暂无数据
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

export default TableWidget

