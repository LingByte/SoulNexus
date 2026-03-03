import React from 'react';
import { motion } from 'framer-motion';
import { Trash2, Settings, Play, Pause, AlertCircle, CheckCircle } from 'lucide-react';
import Button from '@/components/UI/Button';
import type { MCPServer } from '@/types/mcp';

interface MCPCardProps {
  server: MCPServer;
  onEdit?: (server: MCPServer) => void;
  onDelete?: (server: MCPServer) => void;
  onToggle?: (server: MCPServer) => void;
  isMarketplace?: boolean;
  onInstall?: (server: MCPServer) => void;
}

const MCPCard: React.FC<MCPCardProps> = ({
  server,
  onEdit,
  onDelete,
  onToggle,
  isMarketplace = false,
  onInstall,
}) => {
  const getStatusIcon = () => {
    switch (server.status) {
      case 'running':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'error':
        return <AlertCircle className="w-4 h-4 text-red-500" />;
      default:
        return <AlertCircle className="w-4 h-4 text-gray-400" />;
    }
  };

  const getStatusText = () => {
    switch (server.status) {
      case 'running':
        return '运行中';
      case 'stopped':
        return '已停止';
      case 'error':
        return '错误';
      default:
        return '未知';
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      className="bg-white dark:bg-neutral-800 rounded-lg border border-gray-200 dark:border-neutral-700 p-4 hover:border-purple-400 transition-colors"
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1">
          <h3 className="font-semibold text-gray-900 dark:text-gray-100 mb-1">
            {server.name}
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
            {server.description}
          </p>
        </div>
        <div className="flex items-center gap-1 ml-2">
          {getStatusIcon()}
        </div>
      </div>

      <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400 mb-3">
        <span>v{server.version}</span>
        <span className="px-2 py-1 rounded bg-gray-100 dark:bg-neutral-700">
          {server.type === 'builtin' ? '内置' : '外部'}
        </span>
      </div>

      {!isMarketplace && (
        <div className="mb-3 text-xs text-gray-600 dark:text-gray-400">
          <div className="flex items-center justify-between">
            <span>状态: {getStatusText()}</span>
            <span>工具数: {server.toolCount}</span>
          </div>
          {server.lastError && (
            <div className="text-red-500 mt-1 truncate">{server.lastError}</div>
          )}
        </div>
      )}

      <div className="flex items-center gap-2">
        {isMarketplace ? (
          <Button
            onClick={() => onInstall?.(server)}
            variant="primary"
            size="sm"
            className="flex-1"
          >
            安装
          </Button>
        ) : (
          <>
            <Button
              onClick={() => onToggle?.(server)}
              variant={server.enabled ? 'outline' : 'ghost'}
              size="sm"
              leftIcon={server.enabled ? <Pause className="w-3 h-3" /> : <Play className="w-3 h-3" />}
            >
              {server.enabled ? '禁用' : '启用'}
            </Button>
            <Button
              onClick={() => onEdit?.(server)}
              variant="ghost"
              size="sm"
              leftIcon={<Settings className="w-3 h-3" />}
            >
              配置
            </Button>
            <Button
              onClick={() => onDelete?.(server)}
              variant="ghost"
              size="sm"
              leftIcon={<Trash2 className="w-3 h-3" />}
              className="text-red-500 hover:text-red-600"
            >
              删除
            </Button>
          </>
        )}
      </div>
    </motion.div>
  );
};

export default MCPCard;
