import React from 'react'
import { Paintbrush, Type, LayoutGrid, Box } from 'lucide-react'
import { PageTheme } from '@/types/overview'

interface StyleEditorProps {
  theme: PageTheme
  onThemeChange: (theme: Partial<PageTheme>) => void
}

const StyleEditor: React.FC<StyleEditorProps> = ({ theme, onThemeChange }) => {
  return (
    <div className="space-y-6">
      {/* 颜色配置 */}
      <div>
        <div className="flex items-center gap-2 mb-3">
          <Paintbrush className="w-4 h-4" />
          <h4 className="text-sm font-semibold">颜色配置</h4>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">主色</label>
            <div className="flex gap-2">
              <input
                type="color"
                value={theme.primaryColor}
                onChange={(e) => onThemeChange({ primaryColor: e.target.value })}
                className="w-10 h-10 rounded border cursor-pointer"
              />
              <input
                type="text"
                value={theme.primaryColor}
                onChange={(e) => onThemeChange({ primaryColor: e.target.value })}
                className="flex-1 px-2 py-1 text-xs border rounded"
              />
            </div>
          </div>
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">背景色</label>
            <div className="flex gap-2">
              <input
                type="color"
                value={theme.backgroundColor}
                onChange={(e) => onThemeChange({ backgroundColor: e.target.value })}
                className="w-10 h-10 rounded border cursor-pointer"
              />
              <input
                type="text"
                value={theme.backgroundColor}
                onChange={(e) => onThemeChange({ backgroundColor: e.target.value })}
                className="flex-1 px-2 py-1 text-xs border rounded"
              />
            </div>
          </div>
          {theme.secondaryColor && (
            <div>
              <label className="text-xs text-muted-foreground mb-1 block">次要色</label>
              <div className="flex gap-2">
                <input
                  type="color"
                  value={theme.secondaryColor}
                  onChange={(e) => onThemeChange({ secondaryColor: e.target.value })}
                  className="w-10 h-10 rounded border cursor-pointer"
                />
                <input
                  type="text"
                  value={theme.secondaryColor}
                  onChange={(e) => onThemeChange({ secondaryColor: e.target.value })}
                  className="flex-1 px-2 py-1 text-xs border rounded"
                />
              </div>
            </div>
          )}
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">文字色</label>
            <div className="flex gap-2">
              <input
                type="color"
                value={theme.textColor}
                onChange={(e) => onThemeChange({ textColor: e.target.value })}
                className="w-10 h-10 rounded border cursor-pointer"
              />
              <input
                type="text"
                value={theme.textColor}
                onChange={(e) => onThemeChange({ textColor: e.target.value })}
                className="flex-1 px-2 py-1 text-xs border rounded"
              />
            </div>
          </div>
        </div>
      </div>

      {/* 字体配置 */}
      <div>
        <div className="flex items-center gap-2 mb-3">
          <Type className="w-4 h-4" />
          <h4 className="text-sm font-semibold">字体配置</h4>
        </div>
        <div className="space-y-2">
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">字体族</label>
            <select
              value={theme.fontFamily || 'system-ui'}
              onChange={(e) => onThemeChange({ fontFamily: e.target.value })}
              className="w-full px-2 py-1 text-xs border rounded"
            >
              <option value="system-ui">系统默认</option>
              <option value="Inter, sans-serif">Inter</option>
              <option value="Roboto, sans-serif">Roboto</option>
              <option value="'PingFang SC', sans-serif">PingFang SC</option>
              <option value="'Microsoft YaHei', sans-serif">Microsoft YaHei</option>
            </select>
          </div>
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">字体大小</label>
            <input
              type="text"
              value={theme.fontSize || '16px'}
              onChange={(e) => onThemeChange({ fontSize: e.target.value })}
              className="w-full px-2 py-1 text-xs border rounded"
              placeholder="16px"
            />
          </div>
        </div>
      </div>

      {/* 间距配置 */}
      <div>
        <div className="flex items-center gap-2 mb-3">
          <LayoutGrid className="w-4 h-4" />
          <h4 className="text-sm font-semibold">间距配置</h4>
        </div>
        <div className="grid grid-cols-3 gap-2">
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">小</label>
            <input
              type="number"
              value={theme.spacing?.small || 8}
              onChange={(e) => onThemeChange({
                spacing: { 
                  small: parseInt(e.target.value) || 8,
                  medium: theme.spacing?.medium || 16,
                  large: theme.spacing?.large || 24
                }
              })}
              className="w-full px-2 py-1 text-xs border rounded"
            />
          </div>
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">中</label>
            <input
              type="number"
              value={theme.spacing?.medium || 16}
              onChange={(e) => onThemeChange({
                spacing: { 
                  small: theme.spacing?.small || 8,
                  medium: parseInt(e.target.value) || 16,
                  large: theme.spacing?.large || 24
                }
              })}
              className="w-full px-2 py-1 text-xs border rounded"
            />
          </div>
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">大</label>
            <input
              type="number"
              value={theme.spacing?.large || 24}
              onChange={(e) => onThemeChange({
                spacing: { 
                  small: theme.spacing?.small || 8,
                  medium: theme.spacing?.medium || 16,
                  large: parseInt(e.target.value) || 24
                }
              })}
              className="w-full px-2 py-1 text-xs border rounded"
            />
          </div>
        </div>
      </div>

      {/* 卡片样式 */}
      <div>
        <div className="flex items-center gap-2 mb-3">
          <Box className="w-4 h-4" />
          <h4 className="text-sm font-semibold">卡片样式</h4>
        </div>
        <select
          value={theme.cardStyle}
          onChange={(e) => onThemeChange({ cardStyle: e.target.value as any })}
          className="w-full px-2 py-1 text-xs border rounded"
        >
          <option value="default">默认</option>
          <option value="minimal">极简</option>
          <option value="bordered">边框</option>
          <option value="shadow">阴影</option>
          <option value="glass">玻璃</option>
          <option value="gradient">渐变</option>
        </select>
        <div className="mt-2">
          <label className="text-xs text-muted-foreground mb-1 block">圆角</label>
          <input
            type="text"
            value={theme.borderRadius || '8px'}
            onChange={(e) => onThemeChange({ borderRadius: e.target.value })}
            className="w-full px-2 py-1 text-xs border rounded"
            placeholder="8px"
          />
        </div>
      </div>
    </div>
  )
}

export default StyleEditor

