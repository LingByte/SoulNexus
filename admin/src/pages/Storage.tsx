import { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import {
  Folder,
  File,
  Upload,
  Trash2,
  Copy,
  Move,
  RefreshCw,
  Plus,
  ExternalLink,
  Search,
  ChevronRight,
  FolderOpen,
  Eye,
  Grid3x3,
  List,
  Image as ImageIcon,
} from 'lucide-react'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Badge from '@/components/UI/Badge'
import EmptyState from '@/components/UI/EmptyState'
import Modal from '@/components/UI/Modal'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import InputDialog from '@/components/UI/InputDialog'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'
import { showAlert } from '@/utils/notification'
import {
  getStorageInfo,
  switchStorageType,
  listBuckets,
  createBucket,
  deleteBucket,
  listFiles,
  uploadFile,
  deleteFile,
  copyFile,
  moveFile,
  getFileURL,
  getFileInfo,
  processImage,
  type FileInfo,
  type ListFilesResponse,
  type ProcessImageRequest,
} from '@/services/adminApi'
import { formatFileSize, formatDate } from '@/utils/format'
import { cn } from '@/utils/cn'
import VoicePlayer from '@/components/VoicePlayer'

const REGIONS = [
  { value: 'z0', label: 'z0 (华东)' },
  { value: 'z1', label: 'z1 (华北)' },
  { value: 'z2', label: 'z2 (华南)' },
  { value: 'na0', label: 'na0 (北美)' },
  { value: 'as0', label: 'as0 (东南亚)' },
]

const Storage = () => {
  const [storageInfo, setStorageInfo] = useState<{ storageKind: string; supported: string[] } | null>(null)
  const [buckets, setBuckets] = useState<string[]>([])
  const [selectedBucket, setSelectedBucket] = useState<string>('')
  const [files, setFiles] = useState<FileInfo[]>([])
  const [commonPrefixes, setCommonPrefixes] = useState<string[]>([])
  const [currentPrefix, setCurrentPrefix] = useState<string>('')
  const [marker, setMarker] = useState<string>('')
  const [isTruncated, setIsTruncated] = useState(false)
  const [loading, setLoading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<{ [key: string]: number }>({})
  const [searchPrefix, setSearchPrefix] = useState('')
  const [showCreateBucket, setShowCreateBucket] = useState(false)
  const [newBucketName, setNewBucketName] = useState('')
  const [newBucketRegion, setNewBucketRegion] = useState('z0')
  const [selectedFile, setSelectedFile] = useState<FileInfo | null>(null)
  const [showFileDetail, setShowFileDetail] = useState(false)
  const [previewUrl, setPreviewUrl] = useState<string>('')
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list')
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(10)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<{ type: 'file' | 'bucket'; key: string } | null>(null)
  const [showInputDialog, setShowInputDialog] = useState(false)
  const [inputDialogConfig, setInputDialogConfig] = useState<{
    title: string
    label: string
    placeholder: string
    defaultValue: string
    onConfirm: (value: string) => void
  } | null>(null)
  const [showImageProcessDialog, setShowImageProcessDialog] = useState(false)
  const [selectedImageFile, setSelectedImageFile] = useState<FileInfo | null>(null)
  const [uploadCompressEnabled, setUploadCompressEnabled] = useState(false)
  const [uploadCompressQuality, setUploadCompressQuality] = useState(100)

  // 获取存储配置信息
  const fetchStorageInfo = async () => {
    try {
      const info = await getStorageInfo()
      setStorageInfo(info)
    } catch (error: any) {
      showAlert('获取存储配置失败', 'error', error?.msg || error?.message)
    }
  }

  // 切换存储类型
  const handleSwitchStorageType = async (newStorageKind: string) => {
    if (newStorageKind === storageInfo?.storageKind) {
      return
    }
    
    try {
      setLoading(true)
      await switchStorageType(newStorageKind)
      showAlert('存储类型切换成功', 'success')
      // 重新获取存储信息和存储桶列表
      await fetchStorageInfo()
      await fetchBuckets()
      // 清空当前选择的存储桶和文件列表
      setSelectedBucket('')
      setFiles([])
      setCommonPrefixes([])
    } catch (error: any) {
      showAlert('存储类型切换失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  // 获取存储桶列表
  const fetchBuckets = async () => {
    try {
      setLoading(true)
      const bucketList = await listBuckets()
      setBuckets(bucketList)
    } catch (error: any) {
      showAlert('获取存储桶列表失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  // 获取文件列表
  const fetchFiles = async (bucket: string, prefix: string = '', nextMarker: string = '') => {
    if (!bucket) return
    try {
      setLoading(true)
      const result: ListFilesResponse = await listFiles(bucket, {
        prefix: prefix || undefined,
        marker: nextMarker || undefined,
        limit: 100,
        delimiter: '/',
      })
      if (nextMarker) {
        // 追加更多文件
        setFiles((prev) => [...prev, ...(result.files || [])])
      } else {
        // 替换文件列表
        setFiles(result.files || [])
        setCommonPrefixes(result.commonPrefixes || [])
      }
      setMarker(result.marker || '')
      setIsTruncated(result.isTruncated)
    } catch (error: any) {
      showAlert('获取文件列表失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchStorageInfo()
    fetchBuckets()
  }, [])

  useEffect(() => {
    if (selectedBucket) {
      setCurrentPrefix('')
      setMarker('')
      fetchFiles(selectedBucket)
    } else {
      setFiles([])
      setCommonPrefixes([])
    }
  }, [selectedBucket])

  // 创建存储桶
  const handleCreateBucket = async () => {
    if (!newBucketName.trim()) {
      showAlert('请输入存储桶名称', 'error')
      return
    }
    try {
      setLoading(true)
      await createBucket(newBucketName.trim(), newBucketRegion)
      showAlert('创建存储桶成功', 'success')
      setShowCreateBucket(false)
      setNewBucketName('')
      await fetchBuckets()
    } catch (error: any) {
      showAlert('创建存储桶失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  // 删除存储桶
  const handleDeleteBucket = (bucketName: string) => {
    setDeleteTarget({ type: 'bucket', key: bucketName })
    setShowDeleteConfirm(true)
  }

  const confirmDeleteBucket = async () => {
    if (!deleteTarget || deleteTarget.type !== 'bucket') return
    try {
      setLoading(true)
      await deleteBucket(deleteTarget.key)
      showAlert('删除存储桶成功', 'success')
      if (selectedBucket === deleteTarget.key) {
        setSelectedBucket('')
      }
      await fetchBuckets()
    } catch (error: any) {
      showAlert('删除存储桶失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
      setShowDeleteConfirm(false)
      setDeleteTarget(null)
    }
  }

  // 判断是否为图片文件
  const isImageFileForUpload = (file: File): boolean => {
    if (file.type?.startsWith('image/')) {
      return true
    }
    const ext = file.name.split('.').pop()?.toLowerCase()
    const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'tiff', 'tif']
    return imageExts.includes(ext || '')
  }

  // 上传文件（支持多文件）
  const handleUploadFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (!files || files.length === 0 || !selectedBucket) {
      return
    }

    const uploadPromises = Array.from(files).map(async (file) => {
      const key = currentPrefix ? `${currentPrefix}${file.name}` : file.name
      const fileId = `${key}-${Date.now()}`

      try {
        setUploadProgress((prev) => ({ ...prev, [fileId]: 0 }))
        // 模拟上传进度（实际应该从上传API获取）
        const progressInterval = setInterval(() => {
          setUploadProgress((prev) => {
            const current = prev[fileId] || 0
            if (current < 90) {
              return { ...prev, [fileId]: current + 10 }
            }
            return prev
          })
        }, 200)

        // 如果是图片且启用了压缩，传递压缩参数
        const options = isImageFileForUpload(file) && uploadCompressEnabled
          ? { compress: true, quality: uploadCompressQuality }
          : undefined

        await uploadFile(selectedBucket, key, file, options)
        setUploadProgress((prev) => ({ ...prev, [fileId]: 100 }))
        clearInterval(progressInterval)

        setTimeout(() => {
          setUploadProgress((prev) => {
            const newProgress = { ...prev }
            delete newProgress[fileId]
            return newProgress
          })
        }, 500)
      } catch (error: any) {
        showAlert(`上传文件 "${file.name}" 失败`, 'error', error?.msg || error?.message)
        setUploadProgress((prev) => {
          const newProgress = { ...prev }
          delete newProgress[fileId]
          return newProgress
        })
      }
    })

    await Promise.all(uploadPromises)
    showAlert('文件上传完成', 'success')
    await fetchFiles(selectedBucket, currentPrefix)
    // 重置文件输入
    e.target.value = ''
  }

  // 删除文件
  const handleDeleteFile = (key: string) => {
    setDeleteTarget({ type: 'file', key })
    setShowDeleteConfirm(true)
  }

  const confirmDeleteFile = async () => {
    if (!deleteTarget || deleteTarget.type !== 'file' || !selectedBucket) return
    try {
      setLoading(true)
      await deleteFile(selectedBucket, deleteTarget.key)
      showAlert('删除文件成功', 'success')
      await fetchFiles(selectedBucket, currentPrefix)
    } catch (error: any) {
      showAlert('删除文件失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
      setShowDeleteConfirm(false)
      setDeleteTarget(null)
    }
  }

  // 复制文件
  const handleCopyFile = (key: string) => {
    setInputDialogConfig({
      title: '复制文件',
      label: '目标文件路径',
      placeholder: '请输入目标文件路径',
      defaultValue: key,
      onConfirm: async (destKey: string) => {
        if (!destKey || !selectedBucket) return
        try {
          setLoading(true)
          await copyFile(selectedBucket, key, selectedBucket, destKey)
          showAlert('复制文件成功', 'success')
          await fetchFiles(selectedBucket, currentPrefix)
        } catch (error: any) {
          showAlert('复制文件失败', 'error', error?.msg || error?.message)
        } finally {
          setLoading(false)
          setShowInputDialog(false)
          setInputDialogConfig(null)
        }
      },
    })
    setShowInputDialog(true)
  }

  // 移动文件
  const handleMoveFile = (key: string) => {
    setInputDialogConfig({
      title: '移动文件',
      label: '目标文件路径',
      placeholder: '请输入目标文件路径',
      defaultValue: key,
      onConfirm: async (destKey: string) => {
        if (!destKey || !selectedBucket) return
        try {
          setLoading(true)
          await moveFile(selectedBucket, key, selectedBucket, destKey)
          showAlert('移动文件成功', 'success')
          await fetchFiles(selectedBucket, currentPrefix)
        } catch (error: any) {
          showAlert('移动文件失败', 'error', error?.msg || error?.message)
        } finally {
          setLoading(false)
          setShowInputDialog(false)
          setInputDialogConfig(null)
        }
      },
    })
    setShowInputDialog(true)
  }

  // 判断是否为图片文件
  const isImageFile = (file: FileInfo): boolean => {
    if (file.mimeType?.startsWith('image/')) {
      return true
    }
    const ext = file.key.split('.').pop()?.toLowerCase()
    const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'tiff', 'tif']
    return imageExts.includes(ext || '')
  }

  // 打开图片处理对话框
  const handleOpenImageProcess = (file: FileInfo) => {
    setSelectedImageFile(file)
    setShowImageProcessDialog(true)
  }


  // 查看文件详情
  const handleViewFileDetail = async (file: FileInfo) => {
    if (!selectedBucket) return
    try {
      const fileDetail = await getFileInfo(selectedBucket, file.key)
      setSelectedFile(fileDetail)
      setShowFileDetail(true)
      // 获取预览URL
      const { url } = await getFileURL(selectedBucket, file.key, '1h')
      setPreviewUrl(url)
    } catch (error: any) {
      showAlert('获取文件详情失败', 'error', error?.msg || error?.message)
    }
  }

  // 预览文件
  const handlePreviewFile = async (file: FileInfo) => {
    if (!selectedBucket) return
    try {
      const { url } = await getFileURL(selectedBucket, file.key, '1h')
      setPreviewUrl(url)
      setSelectedFile(file)
      setShowFileDetail(true)
    } catch (error: any) {
      showAlert('获取预览链接失败', 'error', error?.msg || error?.message)
    }
  }

  // 进入目录
  const handleEnterDirectory = (prefix: string) => {
    setCurrentPrefix(prefix)
    setMarker('')
    fetchFiles(selectedBucket, prefix)
  }

  // 返回上级目录
  const handleGoBack = () => {
    if (!currentPrefix) return
    const parentPrefix = currentPrefix.split('/').slice(0, -2).join('/')
    const newPrefix = parentPrefix ? `${parentPrefix}/` : ''
    setCurrentPrefix(newPrefix)
    setMarker('')
    fetchFiles(selectedBucket, newPrefix)
  }

  // 搜索文件
  const handleSearch = () => {
    setCurrentPrefix(searchPrefix)
    setMarker('')
    fetchFiles(selectedBucket, searchPrefix)
  }

  const getStorageKindName = (kind?: string) => {
    if (!kind) {
      return '未知'
    }
    const kindMap: { [key: string]: string } = {
      qiniu: '七牛云',
      cos: '腾讯云COS',
      oss: '阿里云OSS',
      minio: 'MinIO',
      local: '本地存储',
      s3: 'AWS S3',
    }
    return kindMap[kind.toLowerCase()] || kind.toUpperCase()
  }

  // 合并文件夹和文件，文件夹在前
  const allItems = [
    ...(commonPrefixes || []).map(prefix => ({ itemType: 'folder' as const, key: prefix, name: prefix })),
    ...(files || []).map(file => ({ itemType: 'file' as const, ...file }))
  ]
  
  // 分页计算（包含文件夹和文件）
  const paginatedItems = allItems.slice((currentPage - 1) * pageSize, currentPage * pageSize)
  const totalPages = Math.ceil(allItems.length / pageSize)

  // 切换页面
  const handlePageChange = (page: number) => {
    setCurrentPage(page)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  // 当文件列表或文件夹列表变化时，重置到第一页
  useEffect(() => {
    setCurrentPage(1)
  }, [files.length, commonPrefixes.length, currentPrefix])

  return (
    <>
      <div className="space-y-1 flex flex-col overflow-hidden" style={{ height: 'calc(100vh - 152px)', maxHeight: 'calc(100vh - 152px)' }}>
        {/* 存储信息卡片 */}
        {storageInfo && (
          <Card padding="none" className="flex-shrink-0">
            <div className="flex items-center gap-4 px-4 py-2">
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground">当前存储类型：</span>
                <Badge variant="outline" className="text-sm font-medium">
                  {getStorageKindName(storageInfo.storageKind)}
                </Badge>
              </div>
              {storageInfo.supported && storageInfo.supported.length > 0 && (
                <>
                  <span className="text-muted-foreground">|</span>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">切换存储类型：</span>
                    <div className="flex items-center gap-1 flex-wrap">
                      {storageInfo.supported.map((kind) => (
                        <Button
                          key={kind}
                          variant={storageInfo.storageKind === kind ? "default" : "outline"}
                          size="sm"
                          onClick={() => handleSwitchStorageType(kind)}
                          disabled={storageInfo.storageKind === kind || loading}
                          className="text-xs h-6 px-2"
                        >
                          {getStorageKindName(kind)}
                        </Button>
                      ))}
                    </div>
                  </div>
                </>
              )}
            </div>
          </Card>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6 flex-1 min-h-0">
          {/* 左侧：存储桶列表 */}
          <Card className="lg:col-span-1 max-h-[77vh] h-[77vh]">
            <div className="border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
              <h2 className="font-semibold text-slate-900 dark:text-slate-100">存储桶列表</h2>
              <Button 
                size="sm"
                onClick={() => setShowCreateBucket(true)} 
                leftIcon={<Plus className="w-4 h-4" />}
              >
                新建
              </Button>
            </div>
            <div>
              {loading && buckets.length === 0 ? (
                <div className="text-center py-8 text-slate-500">加载中...</div>
              ) : buckets.length === 0 ? (
                <EmptyState
                  icon={Folder}
                  title="暂无存储桶"
                  description="点击右上角按钮创建存储桶"
                />
              ) : (
                <div className="space-y-2">
                  {buckets.map((bucket) => (
                    <motion.div
                      key={bucket}
                      whileHover={{ scale: 1.02 }}
                      whileTap={{ scale: 0.98 }}
                    >
                      <div
                        className={cn(
                          'flex items-center justify-between p-3 rounded-lg cursor-pointer transition-colors',
                          selectedBucket === bucket
                            ? 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800'
                            : 'hover:bg-slate-100 dark:hover:bg-slate-800 border border-transparent'
                        )}
                        onClick={() => setSelectedBucket(bucket)}
                      >
                        <div className="flex items-center gap-2 flex-1 min-w-0">
                          <Folder
                            className={cn(
                              'w-5 h-5 flex-shrink-0',
                              selectedBucket === bucket
                                ? 'text-blue-600 dark:text-blue-400'
                                : 'text-slate-400'
                            )}
                          />
                          <span className="font-medium text-slate-900 dark:text-slate-100 truncate">
                            {bucket}
                          </span>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e: React.MouseEvent) => {
                            e.stopPropagation()
                            handleDeleteBucket(bucket)
                          }}
                          className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </div>
                    </motion.div>
                  ))}
                </div>
              )}
            </div>
          </Card>

          {/* 右侧：文件列表 */}
          <Card className="lg:col-span-3 max-h-[77vh] h-[77vh] flex flex-col overflow-hidden">
            {!selectedBucket ? (
              <div className="p-12 text-center">
                <EmptyState
                  icon={Folder}
                  title="请选择存储桶"
                  description="从左侧选择一个存储桶以查看文件"
                />
              </div>
            ) : (
              <div className="flex flex-col h-full overflow-hidden">
                {/* 工具栏 */}
                <div className="p-4 border-b border-slate-200 dark:border-slate-700 space-y-4 flex-shrink-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    {currentPrefix && (
                      <Button 
                        variant="ghost" 
                        size="sm" 
                        onClick={handleGoBack}
                        leftIcon={<ChevronRight className="w-4 h-4 rotate-180" />}
                      >
                        返回
                      </Button>
                    )}
                    <div className="flex-1 flex items-center gap-2 min-w-[200px]">
                      <Input
                        placeholder="搜索文件前缀..."
                        value={searchPrefix}
                        onChange={(e) => setSearchPrefix(e.target.value)}
                        onKeyPress={(e: React.KeyboardEvent) => e.key === 'Enter' && handleSearch()}
                        className="flex-1"
                      />
                      <Button 
                        onClick={handleSearch} 
                        size="sm"
                        leftIcon={<Search className="w-4 h-4" />}
                      >
                        <span className="hidden sm:inline">搜索</span>
                      </Button>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="relative">
                        <input
                          id="file-upload"
                          type="file"
                          multiple
                          className="hidden"
                          onChange={handleUploadFile}
                          disabled={Object.keys(uploadProgress).length > 0}
                        />
                        <Button 
                          disabled={Object.keys(uploadProgress).length > 0} 
                          type="button"
                          onClick={() => document.getElementById('file-upload')?.click()}
                          leftIcon={Object.keys(uploadProgress).length > 0 ? (
                            <RefreshCw className="w-4 h-4 animate-spin" />
                          ) : (
                            <Upload className="w-4 h-4" />
                          )}
                        >
                          {Object.keys(uploadProgress).length > 0 ? '上传中...' : '上传文件'}
                        </Button>
                      </div>
                      {/* 图片压缩选项 */}
                      <div className="flex items-center gap-2 px-3 py-1 border border-slate-200 dark:border-slate-700 rounded-lg">
                        <input
                          type="checkbox"
                          id="upload-compress"
                          checked={uploadCompressEnabled}
                          onChange={(e) => setUploadCompressEnabled(e.target.checked)}
                          disabled={Object.keys(uploadProgress).length > 0}
                          className="w-4 h-4"
                        />
                        <label htmlFor="upload-compress" className="text-sm text-slate-600 dark:text-slate-400 cursor-pointer">
                          压缩图片
                        </label>
                        {uploadCompressEnabled && (
                          <div className="flex items-center gap-2 ml-2">
                            <span className="text-xs text-slate-500">质量:</span>
                            <input
                              type="range"
                              min="1"
                              max="100"
                              value={uploadCompressQuality}
                              onChange={(e) => setUploadCompressQuality(parseInt(e.target.value))}
                              disabled={Object.keys(uploadProgress).length > 0}
                              className="w-20"
                            />
                            <span className="text-xs text-slate-500 w-8">{uploadCompressQuality}</span>
                          </div>
                        )}
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => fetchFiles(selectedBucket, currentPrefix)}
                      leftIcon={<RefreshCw className="w-4 h-4" />}
                    >
                      <span className="hidden sm:inline">刷新</span>
                    </Button>
                    <div className="flex items-center gap-1 border border-slate-200 dark:border-slate-700 rounded-lg p-1">
                      <Button
                        variant={viewMode === 'list' ? 'default' : 'ghost'}
                        size="sm"
                        onClick={() => setViewMode('list')}
                        leftIcon={<List className="w-4 h-4" />}
                        className="h-8"
                      />
                      <Button
                        variant={viewMode === 'grid' ? 'default' : 'ghost'}
                        size="sm"
                        onClick={() => setViewMode('grid')}
                        leftIcon={<Grid3x3 className="w-4 h-4" />}
                        className="h-8"
                      />
                    </div>
                  </div>
                  {currentPrefix && (
                    <div className="text-sm text-slate-600 dark:text-slate-400">
                      当前路径: <span className="font-mono">{currentPrefix}</span>
                    </div>
                  )}
                  {/* 上传进度 */}
                  {Object.keys(uploadProgress).length > 0 && (
                    <div className="space-y-1">
                      {Object.entries(uploadProgress).map(([fileId, progress]) => (
                        <div key={fileId} className="flex items-center gap-2">
                          <div className="flex-1 h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
                            <div
                              className="h-full bg-blue-500 transition-all duration-300"
                              style={{ width: `${progress}%` }}
                            />
                          </div>
                          <span className="text-xs text-slate-500">{progress}%</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* 文件列表 */}
                <div className="p-4 overflow-y-auto flex-1" style={{ maxHeight: 'calc(85vh - 180px)' }}>
                  {loading && files.length === 0 ? (
                    <div className="text-center py-8 text-slate-500">加载中...</div>
                  ) : files.length === 0 && commonPrefixes.length === 0 ? (
                    <EmptyState
                      icon={File}
                      title="暂无文件"
                      description="点击上传文件按钮添加文件"
                    />
                  ) : (
                    <>
                      {/* 列表视图 - 合并显示文件夹和文件 */}
                      {viewMode === 'list' && (
                        <div className="space-y-2">
                          {paginatedItems.map((item) => {
                            if (item.itemType === 'folder') {
                              return (
                                <motion.div
                                  key={item.key}
                                  whileHover={{ scale: 1.01 }}
                                  className="flex items-center justify-between p-3 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-700"
                                >
                                  <div
                                    className="flex items-center gap-3 flex-1 cursor-pointer"
                                    onClick={() => handleEnterDirectory(item.key)}
                                  >
                                    <FolderOpen className="w-5 h-5 text-blue-500" />
                                    <span className="font-medium text-slate-900 dark:text-slate-100">{item.key}</span>
                                    <Badge variant="outline">目录</Badge>
                                  </div>
                                  <ChevronRight className="w-4 h-4 text-slate-400" />
                                </motion.div>
                              )
                            }
                            // 文件项
                            const file = item as FileInfo
                            return (
                              <motion.div
                                key={file.key}
                                whileHover={{ scale: 1.01 }}
                                className="flex items-center justify-between p-3 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-700 cursor-pointer"
                                onClick={() => handlePreviewFile(file)}
                              >
                              <div className="flex items-center gap-3 flex-1 min-w-0">
                                <File className="w-5 h-5 text-slate-400 flex-shrink-0" />
                                <div className="flex-1 min-w-0">
                                  <div className="font-medium text-slate-900 dark:text-slate-100 truncate">
                                    {file.key.split('/').pop()}
                                  </div>
                                  <div className="text-sm text-slate-500 dark:text-slate-400 flex items-center gap-4">
                                    <span>{formatFileSize(file.size)}</span>
                                    <span>{formatDate(file.createdAt)}</span>
                                    {file.mimeType && (
                                      <Badge variant="outline" className="text-xs">
                                        {file.mimeType.split('/')[0]}
                                      </Badge>
                                    )}
                                  </div>
                                </div>
                              </div>
                              <div className="flex items-center gap-2 flex-shrink-0" onClick={(e) => e.stopPropagation()}>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleViewFileDetail(file)}
                                  title="查看详情"
                                  leftIcon={<Eye className="w-4 h-4" />}
                                />
                                {file.publicURL && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => window.open(file.publicURL, '_blank')}
                                    title="在新窗口打开"
                                    leftIcon={<ExternalLink className="w-4 h-4" />}
                                  />
                                )}
                                {isImageFile(file) && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleOpenImageProcess(file)}
                                    title="图片操作"
                                    leftIcon={<ImageIcon className="w-4 h-4" />}
                                  />
                                )}
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleCopyFile(file.key)}
                                  title="复制"
                                  leftIcon={<Copy className="w-4 h-4" />}
                                />
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleMoveFile(file.key)}
                                  title="移动"
                                  leftIcon={<Move className="w-4 h-4" />}
                                />
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleDeleteFile(file.key)}
                                  className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
                                  title="删除"
                                  leftIcon={<Trash2 className="w-4 h-4" />}
                                />
                              </div>
                              </motion.div>
                            )
                          })}
                        </div>
                      )}

                      {/* 卡片视图 - 合并显示文件夹和文件 */}
                      {viewMode === 'grid' && (
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                          {paginatedItems.map((item) => {
                            if (item.itemType === 'folder') {
                              return (
                                <motion.div
                                  key={item.key}
                                  whileHover={{ scale: 1.02 }}
                                  className="p-4 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer"
                                  onClick={() => handleEnterDirectory(item.key)}
                                >
                                  <div className="flex flex-col items-center text-center">
                                    <FolderOpen className="w-12 h-12 text-blue-500 mb-2" />
                                    <div className="font-medium text-slate-900 dark:text-slate-100 text-sm truncate w-full mb-1">
                                      {item.key}
                                    </div>
                                    <Badge variant="outline" className="text-xs">目录</Badge>
                                  </div>
                                </motion.div>
                              )
                            }
                            // 文件项
                            const file = item as FileInfo
                            return (
                            <motion.div
                              key={file.key}
                              whileHover={{ scale: 1.02 }}
                              className="p-4 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer"
                              onClick={() => handlePreviewFile(file)}
                            >
                              <div className="flex flex-col items-center text-center mb-3">
                                <File className="w-12 h-12 text-slate-400 mb-2" />
                                <div className="font-medium text-slate-900 dark:text-slate-100 text-sm truncate w-full">
                                  {file.key.split('/').pop()}
                                </div>
                              </div>
                              <div className="space-y-1 text-xs text-slate-500 dark:text-slate-400 mb-3">
                                <div className="flex justify-between">
                                  <span>大小:</span>
                                  <span>{formatFileSize(file.size)}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span>时间:</span>
                                  <span>{formatDate(file.createdAt)}</span>
                                </div>
                                {file.mimeType && (
                                  <div className="flex justify-center mt-1">
                                    <Badge variant="outline" className="text-xs">
                                      {file.mimeType.split('/')[0]}
                                    </Badge>
                                  </div>
                                )}
                              </div>
                              <div className="flex items-center justify-center gap-1 flex-wrap" onClick={(e) => e.stopPropagation()}>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleViewFileDetail(file)}
                                  title="查看详情"
                                  leftIcon={<Eye className="w-3 h-3" />}
                                />
                                {isImageFile(file) && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleOpenImageProcess(file)}
                                    title="图片操作"
                                    leftIcon={<ImageIcon className="w-3 h-3" />}
                                  />
                                )}
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleCopyFile(file.key)}
                                  title="复制"
                                  leftIcon={<Copy className="w-3 h-3" />}
                                />
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleDeleteFile(file.key)}
                                  className="text-red-600 hover:text-red-700"
                                  title="删除"
                                  leftIcon={<Trash2 className="w-3 h-3" />}
                                />
                              </div>
                              </motion.div>
                            )
                          })}
                        </div>
                      )}

                      {/* 分页 */}
                      {totalPages > 1 && (
                        <div className="mt-6 flex items-center justify-between">
                          <div className="text-sm text-slate-600 dark:text-slate-400">
                            显示 {(currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, allItems.length)} 条，共 {allItems.length} 条
                          </div>
                          <div className="flex items-center gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handlePageChange(currentPage - 1)}
                              disabled={currentPage === 1}
                            >
                              上一页
                            </Button>
                            <div className="flex items-center gap-1">
                              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                                let pageNum: number
                                if (totalPages <= 5) {
                                  pageNum = i + 1
                                } else if (currentPage <= 3) {
                                  pageNum = i + 1
                                } else if (currentPage >= totalPages - 2) {
                                  pageNum = totalPages - 4 + i
                                } else {
                                  pageNum = currentPage - 2 + i
                                }
                                return (
                                  <Button
                                    key={pageNum}
                                    variant={currentPage === pageNum ? 'default' : 'outline'}
                                    size="sm"
                                    onClick={() => handlePageChange(pageNum)}
                                    className="min-w-[2.5rem]"
                                  >
                                    {pageNum}
                                  </Button>
                                )
                              })}
                            </div>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handlePageChange(currentPage + 1)}
                              disabled={currentPage === totalPages}
                            >
                              下一页
                            </Button>
                          </div>
                        </div>
                      )}

                      {/* 加载更多（后端分页） */}
                      {isTruncated && (
                        <div className="text-center pt-4">
                          <Button
                            variant="outline"
                            onClick={() => fetchFiles(selectedBucket, currentPrefix, marker)}
                            disabled={loading}
                            leftIcon={loading ? <RefreshCw className="w-4 h-4 animate-spin" /> : undefined}
                          >
                            {loading ? '加载中...' : '加载更多'}
                          </Button>
                        </div>
                      )}
                    </>
                  )}
                </div>
              </div>
            )}
          </Card>
        </div>

        {/* 创建存储桶对话框 */}
        <Modal
          isOpen={showCreateBucket}
          onClose={() => setShowCreateBucket(false)}
          title="创建存储桶"
        >
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                存储桶名称
              </label>
              <Input
                value={newBucketName}
                onChange={(e) => setNewBucketName(e.target.value)}
                placeholder="请输入存储桶名称"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                区域 (Region)
              </label>
              <Select value={newBucketRegion} onValueChange={setNewBucketRegion}>
                <SelectTrigger>
                  <SelectValue placeholder="选择区域" />
                </SelectTrigger>
                <SelectContent>
                  {REGIONS.map((region) => (
                    <SelectItem key={region.value} value={region.value}>
                      {region.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="flex justify-end gap-2 mt-6">
            <Button variant="outline" onClick={() => setShowCreateBucket(false)}>
              取消
            </Button>
            <Button onClick={handleCreateBucket} disabled={loading}>
              {loading ? '创建中...' : '创建'}
            </Button>
          </div>
        </Modal>

        {/* 文件详情对话框 */}
        <Modal
          isOpen={showFileDetail}
          onClose={() => setShowFileDetail(false)}
          title="文件详情"
          size="xl"
        >
          {selectedFile && (
            <div className="space-y-4">
              {/* 音频预览 - 放在最上面 */}
              {previewUrl && selectedFile.mimeType?.startsWith('audio/') && (
                <div>
                  <label className="text-sm font-medium text-slate-500 dark:text-slate-400 mb-2 block">预览</label>
                  <VoicePlayer
                    audioUrl={previewUrl}
                    title={selectedFile.key.split('/').pop() || '音频文件'}
                    className="w-full"
                  />
                </div>
              )}
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-slate-500 dark:text-slate-400">文件名</label>
                  <p className="mt-1 text-slate-900 dark:text-slate-100 font-mono text-sm break-all">{selectedFile.key}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-slate-500 dark:text-slate-400">文件大小</label>
                  <p className="mt-1 text-slate-900 dark:text-slate-100">{formatFileSize(selectedFile.size)}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-slate-500 dark:text-slate-400">MIME类型</label>
                  <p className="mt-1 text-slate-900 dark:text-slate-100">{selectedFile.mimeType || '未知'}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-slate-500 dark:text-slate-400">创建时间</label>
                  <p className="mt-1 text-slate-900 dark:text-slate-100">{formatDate(selectedFile.createdAt)}</p>
                </div>
                {selectedFile.hash && (
                  <div className="col-span-2">
                    <label className="text-sm font-medium text-slate-500 dark:text-slate-400">文件哈希</label>
                    <p className="mt-1 text-slate-900 dark:text-slate-100 font-mono text-sm break-all">{selectedFile.hash}</p>
                  </div>
                )}
              </div>
              
              {/* 其他类型预览 */}
              {previewUrl && !selectedFile.mimeType?.startsWith('audio/') && (
                <div className="mt-2">
                  <label className="text-sm font-medium text-slate-500 dark:text-slate-400 mb-2 block">预览</label>
                  <div className="border border-slate-200 dark:border-slate-700 rounded-lg p-4 bg-slate-50 dark:bg-slate-900">
                    {selectedFile.mimeType?.startsWith('image/') ? (
                      <img src={previewUrl} alt={selectedFile.key} className="max-w-full max-h-96 mx-auto" />
                    ) : selectedFile.mimeType?.startsWith('video/') ? (
                      <video src={previewUrl} controls className="max-w-full max-h-96 mx-auto" />
                    ) : (
                      <div className="text-center py-8">
                        <p className="text-slate-500 dark:text-slate-400 mb-4">不支持预览此文件类型</p>
                        <Button 
                          onClick={() => window.open(previewUrl, '_blank')}
                          leftIcon={<ExternalLink className="w-4 h-4" />}
                        >
                          在新窗口打开
                        </Button>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </Modal>

        {/* 确认删除对话框 */}
        <ConfirmDialog
          isOpen={showDeleteConfirm}
          onClose={() => {
            setShowDeleteConfirm(false)
            setDeleteTarget(null)
          }}
          onConfirm={() => {
            if (deleteTarget?.type === 'bucket') {
              confirmDeleteBucket()
            } else if (deleteTarget?.type === 'file') {
              confirmDeleteFile()
            }
          }}
          title={deleteTarget?.type === 'bucket' ? '删除存储桶' : '删除文件'}
          message={
            deleteTarget?.type === 'bucket'
              ? `确定要删除存储桶 "${deleteTarget.key}" 吗？此操作不可恢复！`
              : `确定要删除文件 "${deleteTarget?.key}" 吗？`
          }
          confirmText="删除"
          cancelText="取消"
          variant="danger"
          loading={loading}
        />

        {/* 输入对话框 */}
        {inputDialogConfig && (
          <InputDialog
            isOpen={showInputDialog}
            onClose={() => {
              setShowInputDialog(false)
              setInputDialogConfig(null)
            }}
            onConfirm={inputDialogConfig.onConfirm}
            title={inputDialogConfig.title}
            label={inputDialogConfig.label}
            placeholder={inputDialogConfig.placeholder}
            defaultValue={inputDialogConfig.defaultValue}
            loading={loading}
          />
        )}

        {/* 图片编辑器 */}
        {showImageProcessDialog && selectedImageFile && selectedBucket && (
          <ImageEditor
            file={selectedImageFile}
            bucketName={selectedBucket}
            onSave={async (operations, destKey) => {
              // 应用所有操作并保存
              try {
                setLoading(true)
                const request: ProcessImageRequest = {
                  operations: operations.map(op => ({ operation: op.operation, params: op.params })),
                  destKey,
                }
                await processImage(selectedBucket, selectedImageFile.key, request)
                showAlert('图片保存成功', 'success')
                setShowImageProcessDialog(false)
                setSelectedImageFile(null)
                await fetchFiles(selectedBucket, currentPrefix)
              } catch (error: any) {
                showAlert('图片保存失败', 'error', error?.msg || error?.message)
              } finally {
                setLoading(false)
              }
            }}
            onClose={() => {
              setShowImageProcessDialog(false)
              setSelectedImageFile(null)
            }}
          />
        )}
      </div>
    </>
  )
}

// 图片编辑器组件（类似腾讯云）
interface ImageEditorProps {
  file: FileInfo
  bucketName: string
  onSave: (operations: Array<{ operation: string; params: Record<string, any> }>, destKey?: string) => Promise<void>
  onClose: () => void
}

const ImageEditor = ({ file, bucketName, onSave, onClose }: ImageEditorProps) => {
  const [originalImageUrl, setOriginalImageUrl] = useState<string>('')
  const [previewImageUrl, setPreviewImageUrl] = useState<string>('')
  const [originalSize, setOriginalSize] = useState<{ width: number; height: number; fileSize: number } | null>(null)
  const [previewSize, setPreviewSize] = useState<{ width: number; height: number; fileSize: number } | null>(null)
  const [currentOperation, setCurrentOperation] = useState<string>('')
  const [operationParams, setOperationParams] = useState<Record<string, any>>({})
  const [operations, setOperations] = useState<Array<{ operation: string; params: Record<string, any> }>>([])
  const [loading, setLoading] = useState(false)
  const [showSaveDialog, setShowSaveDialog] = useState(false)
  const [destKey, setDestKey] = useState<string>('')
  const [cropArea, setCropArea] = useState<{ x: number; y: number; width: number; height: number } | null>(null)
  const [isCropping, setIsCropping] = useState(false)
  const [cropStart, setCropStart] = useState<{ x: number; y: number } | null>(null)
  const imageRef = useRef<HTMLImageElement>(null)
  const cropRef = useRef<HTMLDivElement>(null)

  // 获取图片尺寸
  const getImageSize = (url: string, apiSize?: number): Promise<{ width: number; height: number; fileSize: number }> => {
    return new Promise((resolve) => {
      const img = new Image()
      img.onload = () => {
        // 获取文件大小
        let fileSize = 0
        if (url.startsWith('data:')) {
          // base64 大小计算：base64 字符串长度 * 3/4
          const base64Data = url.split(',')[1]
          // 减去可能的 padding（= 字符）
          const padding = (base64Data.match(/=/g) || []).length
          fileSize = Math.round((base64Data.length * 3) / 4) - padding
        } else if (apiSize !== undefined) {
          // 如果 API 返回了实际大小，使用它
          fileSize = apiSize
        } else {
          // 对于 URL，使用原文件大小
          fileSize = file.size
        }
        resolve({ width: img.width, height: img.height, fileSize })
      }
      img.onerror = () => resolve({ width: 0, height: 0, fileSize: 0 })
      img.src = url
    })
  }

  // 加载原图
  useEffect(() => {
    const loadOriginal = async () => {
      try {
        const { url } = await getFileURL(bucketName, file.key, '1h')
        setOriginalImageUrl(url)
        setPreviewImageUrl(url)
        const size = await getImageSize(url)
        setOriginalSize(size)
        setPreviewSize(size)
      } catch (error) {
        showAlert('加载图片失败', 'error')
      }
    }
    loadOriginal()
  }, [file, bucketName])

  // 更新预览尺寸
  useEffect(() => {
    if (previewImageUrl) {
      getImageSize(previewImageUrl).then(setPreviewSize)
    }
  }, [previewImageUrl])

  // 应用操作并预览
  const handleApplyOperation = async () => {
    if (!currentOperation) {
      showAlert('请选择操作', 'error')
      return
    }
    
    // 如果是裁剪操作，需要从 cropArea 获取参数
    let params = { ...operationParams }
    if (currentOperation === 'crop') {
      if (!cropArea || cropArea.width <= 0 || cropArea.height <= 0) {
        showAlert('请先选择裁剪区域', 'error')
        return
      }
      // 需要将屏幕坐标转换为图片实际坐标
      if (imageRef.current) {
        const img = imageRef.current
        const rect = img.getBoundingClientRect()
        const imgNaturalWidth = img.naturalWidth
        const imgNaturalHeight = img.naturalHeight
        const scaleX = imgNaturalWidth / rect.width
        const scaleY = imgNaturalHeight / rect.height
        
        params = {
          x: Math.round(cropArea.x * scaleX),
          y: Math.round(cropArea.y * scaleY),
          width: Math.round(cropArea.width * scaleX),
          height: Math.round(cropArea.height * scaleY),
        }
      } else {
        params = {
          x: cropArea.x,
          y: cropArea.y,
          width: cropArea.width,
          height: cropArea.height,
        }
      }
    }
    
    // 如果是压缩操作，确保质量参数存在
    if (currentOperation === 'compress') {
      if (!params.quality || params.quality < 1 || params.quality > 100) {
        params.quality = 90
      }
    }
    
    setLoading(true)
    try {
      // 构建所有操作（包括历史操作 + 当前操作）
      const allOps = [...operations, { operation: currentOperation, params }]
      
      // 调用预览 API，传递所有操作
      const request: ProcessImageRequest = {
        operations: allOps.map(op => ({ operation: op.operation, params: op.params })),
        preview: true,
      }
      const result = await processImage(bucketName, file.key, request)
      
      if (result.preview) {
        setPreviewImageUrl(result.preview)
        // 使用 API 返回的实际大小更新预览尺寸
        if (result.size) {
          getImageSize(result.preview, result.size).then(setPreviewSize)
        } else {
          getImageSize(result.preview).then(setPreviewSize)
        }
        // 添加到操作历史
        setOperations(allOps)
        // 重置当前操作
        setCurrentOperation('')
        setOperationParams({})
        setIsCropping(false)
        setCropArea(null)
      }
    } catch (error: any) {
      showAlert('预览失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  // 撤销操作
  const handleUndo = async () => {
    if (operations.length > 0) {
      const newOps = operations.slice(0, -1)
      setOperations(newOps)
      // 重新应用剩余的操作
      if (newOps.length === 0) {
        setPreviewImageUrl(originalImageUrl)
      } else {
        setLoading(true)
        try {
          const request: ProcessImageRequest = {
            operations: newOps.map(op => ({ operation: op.operation, params: op.params })),
            preview: true,
          }
          const result = await processImage(bucketName, file.key, request)
          if (result.preview) {
            setPreviewImageUrl(result.preview)
            // 使用 API 返回的实际大小
            if (result.size) {
              getImageSize(result.preview, result.size).then(setPreviewSize)
            } else {
              getImageSize(result.preview).then(setPreviewSize)
            }
          }
        } catch (error: any) {
          showAlert('撤销失败', 'error', error?.msg || error?.message)
        } finally {
          setLoading(false)
        }
      }
    }
  }

  // 保存图片
  const handleSave = async (overwrite: boolean) => {
    if (operations.length === 0) {
      showAlert('没有可保存的操作', 'error')
      return
    }
    setLoading(true)
    try {
      const saveKey = overwrite ? undefined : (destKey || file.key + '_edited')
      await onSave(operations, saveKey)
    } finally {
      setLoading(false)
    }
  }

  // 裁剪工具：鼠标事件处理
  const handleCropMouseDown = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!isCropping || !imageRef.current) return
    const rect = imageRef.current.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    setCropStart({ x, y })
    setCropArea({ x, y, width: 0, height: 0 })
  }

  const handleCropMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!isCropping || !cropStart || !imageRef.current) return
    const rect = imageRef.current.getBoundingClientRect()
    const currentX = e.clientX - rect.left
    const currentY = e.clientY - rect.top
    
    const x = Math.min(cropStart.x, currentX)
    const y = Math.min(cropStart.y, currentY)
    const width = Math.abs(currentX - cropStart.x)
    const height = Math.abs(currentY - cropStart.y)
    
    // 限制在图片范围内
    const maxX = rect.width
    const maxY = rect.height
    const finalX = Math.max(0, Math.min(x, maxX - width))
    const finalY = Math.max(0, Math.min(y, maxY - height))
    const finalWidth = Math.min(width, maxX - finalX)
    const finalHeight = Math.min(height, maxY - finalY)
    
    setCropArea({ x: finalX, y: finalY, width: finalWidth, height: finalHeight })
  }

  const handleCropMouseUp = () => {
    if (isCropping) {
      setCropStart(null)
    }
  }

  // 渲染操作参数面板
  const renderOperationPanel = () => {
    if (!currentOperation) return null

    const operationConfigs: Record<string, () => JSX.Element> = {
      crop: () => (
        <div className="space-y-4">
          <div className="text-sm text-slate-600 dark:text-slate-400">
            在预览图上拖拽选择裁剪区域
          </div>
          {cropArea && (
            <>
              <div>
                <label className="block text-sm font-medium mb-2">X坐标</label>
                <Input
                  type="number"
                  value={cropArea.x}
                  onChange={(e) => setCropArea({ ...cropArea, x: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Y坐标</label>
                <Input
                  type="number"
                  value={cropArea.y}
                  onChange={(e) => setCropArea({ ...cropArea, y: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">宽度</label>
                <Input
                  type="number"
                  value={cropArea.width}
                  onChange={(e) => setCropArea({ ...cropArea, width: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">高度</label>
                <Input
                  type="number"
                  value={cropArea.height}
                  onChange={(e) => setCropArea({ ...cropArea, height: parseInt(e.target.value) || 0 })}
                />
              </div>
            </>
          )}
        </div>
      ),
      compress: () => {
        const isPNG = file.key.toLowerCase().endsWith('.png')
        return (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2">压缩质量 (1-100)</label>
              <Input
                type="number"
                min="1"
                max="100"
                value={operationParams.quality || '90'}
                onChange={(e) => setOperationParams({ ...operationParams, quality: parseInt(e.target.value) || 90 })}
              />
              <div className="text-xs text-slate-500 mt-1">
                {isPNG ? (
                  <>PNG 格式将自动转换为 JPEG 以应用质量压缩。数值越小，文件越小，但画质越低。推荐值：90</>
                ) : (
                  <>数值越小，文件越小，但画质越低。推荐值：90</>
                )}
              </div>
            </div>
          </div>
        )
      },
      rotate: () => (
        <div>
          <label className="block text-sm font-medium mb-2">旋转角度</label>
          <Select value={operationParams.angle?.toString() || '90'} onValueChange={(v) => setOperationParams({ ...operationParams, angle: parseInt(v) })}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="90">90度</SelectItem>
              <SelectItem value="180">180度</SelectItem>
              <SelectItem value="270">270度</SelectItem>
            </SelectContent>
          </Select>
        </div>
      ),
      flip: () => (
        <div>
          <label className="block text-sm font-medium mb-2">翻转方向</label>
          <Select value={operationParams.direction || 'horizontal'} onValueChange={(v) => setOperationParams({ ...operationParams, direction: v })}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="horizontal">水平翻转</SelectItem>
              <SelectItem value="vertical">垂直翻转</SelectItem>
            </SelectContent>
          </Select>
        </div>
      ),
      filter: () => (
        <div>
          <label className="block text-sm font-medium mb-2">滤镜类型</label>
          <Select value={operationParams.filterType || 'grayscale'} onValueChange={(v) => setOperationParams({ ...operationParams, filterType: v })}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="grayscale">灰度</SelectItem>
              <SelectItem value="sepia">怀旧</SelectItem>
              <SelectItem value="invert">反色</SelectItem>
              <SelectItem value="blur">模糊</SelectItem>
              <SelectItem value="sharpen">锐化</SelectItem>
              <SelectItem value="emboss">浮雕</SelectItem>
              <SelectItem value="edge">边缘检测</SelectItem>
              <SelectItem value="vintage">复古</SelectItem>
              <SelectItem value="cool">冷色调</SelectItem>
              <SelectItem value="warm">暖色调</SelectItem>
            </SelectContent>
          </Select>
        </div>
      ),
      adjust: () => (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">亮度 (0.0 - 2.0)</label>
            <Input
              type="number"
              step="0.1"
              value={operationParams.brightness || '1.0'}
              onChange={(e) => setOperationParams({ ...operationParams, brightness: parseFloat(e.target.value) || 1.0 })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">对比度 (0.0 - 2.0)</label>
            <Input
              type="number"
              step="0.1"
              value={operationParams.contrast || '1.0'}
              onChange={(e) => setOperationParams({ ...operationParams, contrast: parseFloat(e.target.value) || 1.0 })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">饱和度 (0.0 - 2.0)</label>
            <Input
              type="number"
              step="0.1"
              value={operationParams.saturation || '1.0'}
              onChange={(e) => setOperationParams({ ...operationParams, saturation: parseFloat(e.target.value) || 1.0 })}
            />
          </div>
        </div>
      ),
      watermark: () => (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">水印文字</label>
            <Input
              value={operationParams.text || ''}
              onChange={(e) => setOperationParams({ ...operationParams, text: e.target.value })}
              placeholder="输入水印文字"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">位置</label>
            <Select value={operationParams.position || 'bottom-right'} onValueChange={(v) => setOperationParams({ ...operationParams, position: v })}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="top-left">左上</SelectItem>
                <SelectItem value="top-right">右上</SelectItem>
                <SelectItem value="bottom-left">左下</SelectItem>
                <SelectItem value="bottom-right">右下</SelectItem>
                <SelectItem value="center">居中</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      ),
      border: () => (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">边框宽度（像素）</label>
            <Input
              type="number"
              value={operationParams.width || ''}
              onChange={(e) => setOperationParams({ ...operationParams, width: parseInt(e.target.value) || 0 })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">边框颜色</label>
            <Input
              type="color"
              value={operationParams.color || '#000000'}
              onChange={(e) => setOperationParams({ ...operationParams, color: e.target.value })}
            />
          </div>
        </div>
      ),
      roundCorners: () => (
        <div>
          <label className="block text-sm font-medium mb-2">圆角半径（像素）</label>
          <Input
            type="number"
            value={operationParams.radius || ''}
            onChange={(e) => setOperationParams({ ...operationParams, radius: parseInt(e.target.value) || 0 })}
          />
        </div>
      ),
    }

    return operationConfigs[currentOperation]?.() || null
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-900 rounded-lg w-full h-full max-w-7xl max-h-[90vh] flex flex-col">
        {/* 头部 */}
        <div className="flex items-center justify-between p-4 border-b border-slate-200 dark:border-slate-700">
          <h2 className="text-lg font-semibold">图片编辑器 - {file.key.split('/').pop()}</h2>
          <Button variant="ghost" size="sm" onClick={onClose}>
            ✕
          </Button>
        </div>

        {/* 主体内容 */}
        <div className="flex-1 flex overflow-hidden">
          {/* 左侧工具栏 */}
          <div className="w-64 border-r border-slate-200 dark:border-slate-700 p-4 overflow-y-auto">
            <div className="space-y-2 mb-4">
              <Button
                variant={currentOperation === 'crop' ? 'default' : 'outline'}
                size="sm"
                className="w-full justify-start"
                onClick={() => {
                  setCurrentOperation('crop')
                  setIsCropping(true)
                }}
              >
                裁剪
              </Button>
              <Button
                variant={currentOperation === 'compress' ? 'default' : 'outline'}
                size="sm"
                className="w-full justify-start"
                onClick={() => setCurrentOperation('compress')}
              >
                压缩质量
              </Button>
              <Button
                variant={currentOperation === 'rotate' ? 'default' : 'outline'}
                size="sm"
                className="w-full justify-start"
                onClick={() => setCurrentOperation('rotate')}
              >
                旋转
              </Button>
              <Button
                variant={currentOperation === 'flip' ? 'default' : 'outline'}
                size="sm"
                className="w-full justify-start"
                onClick={() => setCurrentOperation('flip')}
              >
                翻转
              </Button>
              <Button
                variant={currentOperation === 'filter' ? 'default' : 'outline'}
                size="sm"
                className="w-full justify-start"
                onClick={() => setCurrentOperation('filter')}
              >
                滤镜
              </Button>
              <Button
                variant={currentOperation === 'adjust' ? 'default' : 'outline'}
                size="sm"
                className="w-full justify-start"
                onClick={() => setCurrentOperation('adjust')}
              >
                调整
              </Button>
              <Button
                variant={currentOperation === 'watermark' ? 'default' : 'outline'}
                size="sm"
                className="w-full justify-start"
                onClick={() => setCurrentOperation('watermark')}
              >
                水印
              </Button>
              <Button
                variant={currentOperation === 'border' ? 'default' : 'outline'}
                size="sm"
                className="w-full justify-start"
                onClick={() => setCurrentOperation('border')}
              >
                边框
              </Button>
              <Button
                variant={currentOperation === 'roundCorners' ? 'default' : 'outline'}
                size="sm"
                className="w-full justify-start"
                onClick={() => setCurrentOperation('roundCorners')}
              >
                圆角
              </Button>
            </div>

            {/* 操作参数面板 */}
            {currentOperation && (
              <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
                {renderOperationPanel()}
                <Button
                  className="w-full mt-4"
                  onClick={handleApplyOperation}
                  disabled={loading}
                >
                  {loading ? '处理中...' : '应用'}
                </Button>
              </div>
            )}

            {/* 操作历史 */}
            {operations.length > 0 && (
              <div className="border-t border-slate-200 dark:border-slate-700 pt-4 mt-4">
                <div className="text-sm font-medium mb-2">操作历史</div>
                <div className="space-y-1">
                  {operations.map((op, idx) => (
                    <div key={idx} className="text-xs text-slate-500">
                      {idx + 1}. {op.operation}
                    </div>
                  ))}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full mt-2"
                  onClick={handleUndo}
                >
                  撤销
                </Button>
              </div>
            )}
          </div>

          {/* 右侧预览区域 */}
          <div className="flex-1 p-4 overflow-hidden flex flex-col">
            <div className="grid grid-cols-2 gap-4 flex-1 min-h-0">
              <div className="flex flex-col">
                <div className="text-sm font-medium mb-2 flex items-center justify-between">
                  <span>原图</span>
                  {originalSize && (
                    <span className="text-xs text-slate-500">
                      {originalSize.width} × {originalSize.height} | {formatFileSize(originalSize.fileSize)}
                    </span>
                  )}
                </div>
                <div className="border border-slate-200 dark:border-slate-700 rounded-lg p-4 bg-slate-50 dark:bg-slate-800 flex items-center justify-center flex-1 overflow-hidden">
                  {originalImageUrl && (
                    <img 
                      src={originalImageUrl} 
                      alt="原图" 
                      className="max-w-full max-h-full object-contain"
                      style={{ maxHeight: '100%' }}
                    />
                  )}
                </div>
              </div>
              <div className="flex flex-col">
                <div className="text-sm font-medium mb-2 flex items-center justify-between">
                  <span>效果预览</span>
                  {previewSize && (
                    <span className="text-xs text-slate-500">
                      {previewSize.width} × {previewSize.height} | {formatFileSize(previewSize.fileSize)}
                    </span>
                  )}
                </div>
                <div 
                  className="border border-slate-200 dark:border-slate-700 rounded-lg p-4 bg-slate-50 dark:bg-slate-800 flex items-center justify-center flex-1 overflow-hidden relative"
                  onMouseDown={isCropping ? handleCropMouseDown : undefined}
                  onMouseMove={isCropping ? handleCropMouseMove : undefined}
                  onMouseUp={isCropping ? handleCropMouseUp : undefined}
                  onMouseLeave={isCropping ? handleCropMouseUp : undefined}
                  style={{ cursor: isCropping ? 'crosshair' : 'default' }}
                >
                  {previewImageUrl && (
                    <>
                      <img 
                        ref={imageRef}
                        src={previewImageUrl} 
                        alt="预览" 
                        className="max-w-full max-h-full object-contain"
                        style={{ maxHeight: '100%' }}
                        draggable={false}
                      />
                      {isCropping && cropArea && cropArea.width > 0 && cropArea.height > 0 && (
                        <div
                          ref={cropRef}
                          className="absolute border-2 border-blue-500 bg-blue-500/20"
                          style={{
                            left: `${cropArea.x}px`,
                            top: `${cropArea.y}px`,
                            width: `${cropArea.width}px`,
                            height: `${cropArea.height}px`,
                            pointerEvents: 'none',
                          }}
                        >
                          <div className="absolute -top-6 left-0 text-xs text-blue-600 bg-white dark:bg-slate-800 px-1 rounded">
                            {Math.round(cropArea.width)} × {Math.round(cropArea.height)}
                          </div>
                        </div>
                      )}
                    </>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* 底部操作按钮 */}
        <div className="border-t border-slate-200 dark:border-slate-700 p-4 flex items-center justify-between">
          <div className="text-sm text-slate-500">
            {operations.length > 0 ? `已应用 ${operations.length} 个操作` : '未应用任何操作'}
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={onClose}>
              取消
            </Button>
            <Button
              variant="outline"
              onClick={() => setShowSaveDialog(true)}
              disabled={operations.length === 0}
            >
              另存为
            </Button>
            <Button
              onClick={() => handleSave(true)}
              disabled={operations.length === 0}
            >
              覆盖原图
            </Button>
          </div>
        </div>
      </div>

      {/* 另存为对话框 */}
      {showSaveDialog && (
        <Modal
          isOpen={showSaveDialog}
          onClose={() => setShowSaveDialog(false)}
          title="另存为"
        >
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2">目标文件路径</label>
              <Input
                value={destKey}
                onChange={(e) => setDestKey(e.target.value)}
                placeholder={file.key + '_edited'}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setShowSaveDialog(false)}>
                取消
              </Button>
              <Button onClick={async () => {
                await handleSave(false)
                setShowSaveDialog(false)
              }}>
                保存
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

export default Storage
