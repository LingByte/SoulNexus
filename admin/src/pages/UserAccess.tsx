import { useState } from 'react'
import { UserCog } from 'lucide-react'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Badge from '@/components/UI/Badge'
import { showAlert } from '@/utils/notification'
import { getUserAccess, setUserAccess } from '@/services/userAccessApi'
import { listRoles, type Role } from '@/services/roleApi'
import { listPermissions, type PermissionListItem } from '@/services/permissionApi'

const UserAccess = () => {
  const [userIdInput, setUserIdInput] = useState('')
  const [loadedUserId, setLoadedUserId] = useState<number | null>(null)
  const [email, setEmail] = useState('')
  const [legacyRole, setLegacyRole] = useState('')
  const [effectiveKeys, setEffectiveKeys] = useState<string[]>([])
  const [roleIds, setRoleIds] = useState<number[]>([])
  const [permIds, setPermIds] = useState<number[]>([])
  const [rolesCatalog, setRolesCatalog] = useState<Role[]>([])
  const [permsCatalog, setPermsCatalog] = useState<PermissionListItem[]>([])
  const [loading, setLoading] = useState(false)

  const ensureCatalog = async () => {
    if (rolesCatalog.length === 0) {
      const r = await listRoles({ page: 1, pageSize: 200 })
      setRolesCatalog(r.items || [])
    }
    if (permsCatalog.length === 0) {
      const p = await listPermissions({ page: 1, pageSize: 500 })
      setPermsCatalog(p.items || [])
    }
  }

  const loadAccess = async () => {
    const uid = parseInt(userIdInput.trim(), 10)
    if (!uid || uid <= 0) {
      showAlert('请输入有效用户 ID', 'error')
      return
    }
    setLoading(true)
    try {
      await ensureCatalog()
      const data = await getUserAccess(uid)
      setLoadedUserId(uid)
      setEmail(data.user.email)
      setLegacyRole(data.user.legacyRole || '')
      setEffectiveKeys(data.effectivePermissionKeys || [])
      setRoleIds((data.roles || []).map((r) => r.id))
      setPermIds((data.extraPermissions || []).map((p) => p.id))
      showAlert('已加载用户授权', 'success')
    } catch (e: any) {
      showAlert(e?.message || '加载失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const save = async () => {
    if (!loadedUserId) {
      showAlert('请先加载用户', 'error')
      return
    }
    try {
      await setUserAccess(loadedUserId, { roleIds, permissionIds: permIds })
      const data = await getUserAccess(loadedUserId)
      setEffectiveKeys(data.effectivePermissionKeys || [])
      showAlert('已保存', 'success')
    } catch (e: any) {
      showAlert(e?.message || '保存失败', 'error')
    }
  }

  const toggleRole = (id: number) => {
    setRoleIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  const togglePerm = (id: number) => {
    setPermIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  return (
    <>
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center gap-3">
          <UserCog className="w-8 h-8 text-indigo-600" />
          <div>
            <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">用户授权</h1>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              聚合接口加载角色（含权限明细）、附加权限与最终权限列表；保存时一次写入。
            </p>
          </div>
        </div>

        <Card className="p-4 space-y-4">
          <div className="flex flex-wrap gap-2 items-end">
            <div>
              <label className="block text-xs text-slate-500 mb-1">用户 ID</label>
              <Input value={userIdInput} onChange={(e) => setUserIdInput(e.target.value)} placeholder="例如 2" />
            </div>
            <Button variant="secondary" onClick={() => void loadAccess()} disabled={loading}>
              加载
            </Button>
            <Button onClick={() => void save()} disabled={!loadedUserId}>
              保存
            </Button>
          </div>

          {loadedUserId != null && (
            <>
              <div className="text-sm border-t border-slate-200 dark:border-slate-700 pt-4 space-y-1">
                <div>
                  <span className="text-slate-500">用户：</span>
                  {email}（ID {loadedUserId}）
                </div>
                <div>
                  <span className="text-slate-500">legacy role 字段：</span>
                  {legacyRole || '—'}
                </div>
              </div>

              <div>
                <h3 className="font-medium text-slate-900 dark:text-white mb-2">合并后的权限 key</h3>
                <div className="flex flex-wrap gap-1">
                  {effectiveKeys.map((k) => (
                    <Badge key={k} variant="muted" className="text-xs font-mono">
                      {k}
                    </Badge>
                  ))}
                  {effectiveKeys.length === 0 && <span className="text-xs text-slate-500">无</span>}
                </div>
              </div>

              <div>
                <h3 className="font-medium text-slate-900 dark:text-white mb-2">角色（多选）</h3>
                <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-2">
                  {rolesCatalog.map((r) => (
                    <label key={r.id} className="flex items-center gap-2 text-sm cursor-pointer">
                      <input type="checkbox" checked={roleIds.includes(r.id)} onChange={() => toggleRole(r.id)} />
                      <span className="font-mono text-xs">{r.slug}</span>
                      <span>{r.name}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <h3 className="font-medium text-slate-900 dark:text-white mb-2">附加权限（user_permissions）</h3>
                <div className="max-h-64 overflow-y-auto border border-slate-200 dark:border-slate-700 rounded-lg p-3 grid sm:grid-cols-2 gap-2">
                  {permsCatalog.map((p) => (
                    <label key={p.id} className="flex items-start gap-2 text-xs cursor-pointer">
                      <input
                        type="checkbox"
                        checked={permIds.includes(p.id)}
                        onChange={() => togglePerm(p.id)}
                        className="mt-0.5"
                      />
                      <span>
                        <span className="font-mono text-indigo-600">{p.key}</span>
                        <span className="text-slate-600 dark:text-slate-300 ml-1">{p.name}</span>
                      </span>
                    </label>
                  ))}
                </div>
              </div>
            </>
          )}
        </Card>
      </div>
    </>
  )
}

export default UserAccess
