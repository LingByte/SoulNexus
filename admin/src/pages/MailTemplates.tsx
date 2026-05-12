// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Edit, Trash2, RefreshCw } from 'lucide-react'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Badge from '@/components/UI/Badge'
import EmptyState from '@/components/UI/EmptyState'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import { showAlert } from '@/utils/notification'
import {
  listMailTemplates,
  deleteMailTemplate,
  type MailTemplate,
} from '@/services/notificationsApi'

const MailTemplatesPage = () => {
  const navigate = useNavigate()
  const [list, setList] = useState<MailTemplate[]>([])
  const [loading, setLoading] = useState(false)
  const [showDelete, setShowDelete] = useState(false)
  const [deleting, setDeleting] = useState<MailTemplate | null>(null)

  const fetchList = async () => {
    setLoading(true)
    try {
      const res = await listMailTemplates({ page: 1, pageSize: 200 })
      setList(res.list || [])
    } catch (e: any) {
      showAlert('加载模板失败', 'error', e?.msg || e?.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchList() }, [])

  const submitDelete = async () => {
    if (!deleting) return
    try {
      await deleteMailTemplate(deleting.id); showAlert('删除成功', 'success')
      setShowDelete(false); setDeleting(null); fetchList()
    } catch (e: any) { showAlert('删除失败', 'error', e?.msg || e?.message) }
  }

  return (
    <>
      <div className="p-4 md:p-6 space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold">邮件模板</h1>
            <p className="text-sm text-muted-foreground">业务方按 code 引用模板，运行期渲染 {`{{Name}}`} 占位符。</p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={fetchList}><RefreshCw className="w-4 h-4 mr-1" />刷新</Button>
            <Button onClick={() => navigate('/mail-templates/new')}><Plus className="w-4 h-4 mr-1" />新建模板</Button>
          </div>
        </div>

        <Card>
          {loading ? <div className="p-8 text-center text-muted-foreground">加载中...</div> :
            list.length === 0 ? <EmptyState title="暂无模板" description="点击右上角新建模板" /> :
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead className="bg-muted/50">
                  <tr>
                    <th className="text-left px-4 py-2">code</th>
                    <th className="text-left px-4 py-2">名称</th>
                    <th className="text-left px-4 py-2">语言</th>
                    <th className="text-left px-4 py-2">状态</th>
                    <th className="text-left px-4 py-2">更新时间</th>
                    <th className="text-right px-4 py-2">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {list.map((t) => (
                    <tr key={t.id} className="border-t">
                      <td className="px-4 py-2 font-mono text-xs">{t.code}</td>
                      <td className="px-4 py-2">{t.name}</td>
                      <td className="px-4 py-2">{t.locale || '-'}</td>
                      <td className="px-4 py-2">{t.enabled ? <Badge>启用</Badge> : <Badge variant="outline">禁用</Badge>}</td>
                      <td className="px-4 py-2 text-muted-foreground">{t.updatedAt || t.createdAt || '-'}</td>
                      <td className="px-4 py-2 text-right space-x-1">
                        <Button size="sm" variant="ghost" onClick={() => navigate(`/mail-templates/${t.id}/edit`)}><Edit className="w-4 h-4" /></Button>
                        <Button size="sm" variant="ghost" onClick={() => { setDeleting(t); setShowDelete(true) }}><Trash2 className="w-4 h-4" /></Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>}
        </Card>

        <ConfirmDialog
          isOpen={showDelete}
          title="删除模板"
          message={`确定要删除模板 “${deleting?.name}” 吗？`}
          onClose={() => setShowDelete(false)}
          onConfirm={submitDelete}
          variant="danger"
        />
      </div>
    </>
  )
}

export default MailTemplatesPage
