import React from 'react'
import Modal from '@/components/UI/Modal'

interface HelpModalProps {
  isOpen: boolean
  onClose: () => void
}

/**
 * 帮助模态框组件
 * 显示工作流编辑器的操作说明
 */
export const HelpModal: React.FC<HelpModalProps> = ({ isOpen, onClose }) => {
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="操作说明"
      size="lg"
    >
      <div className="space-y-6">
        {/* 连接操作说明 */}
        <div>
          <h4 className="text-sm font-semibold text-gray-900 dark:text-white mb-3">
            连接操作
          </h4>
          <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 bg-blue-500 rounded-full"></div>
              <span>蓝色点：输出连接点</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 bg-green-500 rounded-full"></div>
              <span>绿色点：输入连接点</span>
            </div>
            <ul className="list-disc list-inside space-y-1 ml-5">
              <li>点击蓝色点开始连接</li>
              <li>拖拽到绿色点完成连接</li>
              <li>点击连接线选中连接</li>
              <li>右键连接线打开菜单</li>
              <li>双击连接线直接删除连接</li>
              <li>按Delete键删除选中连接</li>
            </ul>
          </div>
        </div>
        
        {/* 画布操作说明 */}
        <div>
          <h4 className="text-sm font-semibold text-gray-900 dark:text-white mb-3">
            画布操作
          </h4>
          <ul className="list-disc list-inside space-y-1 text-sm text-gray-600 dark:text-gray-400 ml-5">
            <li>拖拽空白区域移动画布视角</li>
            <li>拖拽节点调整位置（无限画布）</li>
            <li>使用 +/- 按钮缩放画布</li>
            <li>使用鼠标滚轮缩放画布</li>
            <li>点击"重置"恢复默认视图</li>
            <li>点击"居中"显示所有节点</li>
            <li>点击节点选中，显示配置按钮</li>
            <li>点击设置按钮配置节点参数</li>
          </ul>
        </div>

        {/* 快捷键说明 */}
        <div>
          <h4 className="text-sm font-semibold text-gray-900 dark:text-white mb-3">
            快捷键
          </h4>
          <ul className="list-disc list-inside space-y-1 text-sm text-gray-600 dark:text-gray-400 ml-5">
            <li><kbd className="px-2 py-1 bg-gray-200 dark:bg-gray-700 rounded text-xs">Delete</kbd> - 删除选中的节点或连接</li>
            <li><kbd className="px-2 py-1 bg-gray-200 dark:bg-gray-700 rounded text-xs">Esc</kbd> - 取消选择</li>
            <li><kbd className="px-2 py-1 bg-gray-200 dark:bg-gray-700 rounded text-xs">Ctrl+S</kbd> - 保存工作流</li>
          </ul>
        </div>

        {/* 节点类型说明 */}
        <div>
          <h4 className="text-sm font-semibold text-gray-900 dark:text-white mb-3">
            节点类型
          </h4>
          <div className="grid grid-cols-2 gap-3 text-sm text-gray-600 dark:text-gray-400">
            <div>
              <strong className="text-gray-900 dark:text-white">开始节点</strong>
              <p className="text-xs">工作流的入口点</p>
            </div>
            <div>
              <strong className="text-gray-900 dark:text-white">结束节点</strong>
              <p className="text-xs">工作流的出口点</p>
            </div>
            <div>
              <strong className="text-gray-900 dark:text-white">任务节点</strong>
              <p className="text-xs">执行具体操作</p>
            </div>
            <div>
              <strong className="text-gray-900 dark:text-white">条件分支</strong>
              <p className="text-xs">根据条件分支</p>
            </div>
            <div>
              <strong className="text-gray-900 dark:text-white">事件节点</strong>
              <p className="text-xs">发布或监听事件</p>
            </div>
            <div>
              <strong className="text-gray-900 dark:text-white">脚本节点</strong>
              <p className="text-xs">执行自定义脚本</p>
            </div>
          </div>
        </div>
      </div>
    </Modal>
  )
}
