import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import PetStudioLayout from '@/components/PetStudio/PetStudioLayout'
import { jsTemplateService } from '@/api/jsTemplate'
import { notifyApiError, notifyApiResult } from '@/utils/apiFeedback'
import { showAlert } from '@/utils/notification'
import {
  createProjectFromTemplate,
  createProjectFromTemplateWithAssets,
  DEFAULT_STARTER_TEMPLATE_ID,
  getStarterTemplate,
  petNameFromProject,
} from './projectUtils'
import { clearPetDraft, loadPetDraft, savePetDraft } from './petDraftStorage'
import {
  ensureSpriteAssetsInProject,
  fetchDefaultSpriteAssetsForProject,
  hasSpriteAssets,
  isSpriteProject,
  needsSpriteRuntimeUpgrade,
  upgradeSpriteRuntime,
} from './spriteProjectUtils'
import type { PetProjectV1 } from './types'
import { PROJECT_FILES } from './types'

function projectFromApi(entry: string, files: Record<string, string>): PetProjectV1 {
  return {
    v: 1,
    entry: entry || PROJECT_FILES.entry,
    files: { ...files },
  }
}

function resolveTemplateId(routeId: string | undefined, stateId: string | null, refId: string | null): string | null {
  if (routeId && routeId !== 'new') return routeId
  return stateId ?? refId
}

async function finalizeSpriteProject(
  project: PetProjectV1,
  displayName: string,
  templateId?: string | null,
): Promise<PetProjectV1> {
  let next = project
  if (needsSpriteRuntimeUpgrade(next)) {
    next = upgradeSpriteRuntime(next, displayName)
  }
  if (!hasSpriteAssets(next)) {
    const sprites = await fetchDefaultSpriteAssetsForProject(next, templateId)
    if (Object.keys(sprites).length > 0) {
      next = { ...next, files: { ...next.files, ...sprites } }
    } else {
      next = ensureSpriteAssetsInProject(next)
    }
  }
  return next
}

export default function PetStudioPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const isNewRoute = id === 'new' || !id

  const starterTemplateId = useMemo(
    () => searchParams.get('template') || DEFAULT_STARTER_TEMPLATE_ID,
    [searchParams],
  )
  const starterTemplate = useMemo(() => getStarterTemplate(starterTemplateId), [starterTemplateId])

  const [loading, setLoading] = useState(!isNewRoute)
  const [saving, setSaving] = useState(false)
  const [templateId, setTemplateId] = useState<string | null>(null)
  const [jsSourceId, setJsSourceId] = useState<string | null>(null)
  const [templateName, setTemplateName] = useState('我的桌宠')
  const [project, setProject] = useState<PetProjectV1>(() => createProjectFromTemplate(starterTemplateId))
  const [savedRevision, setSavedRevision] = useState(0)
  const [storageHint, setStorageHint] = useState<string | null>(null)
  const draftTimer = useRef<number | null>(null)
  const templateIdRef = useRef<string | null>(null)
  const savingRef = useRef(false)

  useEffect(() => {
    if (isNewRoute) {
      templateIdRef.current = null
      setStorageHint(null)
      void createProjectFromTemplateWithAssets(starterTemplateId).then(async (p) => {
        const name = petNameFromProject(p, starterTemplate.name)
        setProject(await finalizeSpriteProject(p, name, starterTemplateId))
      })
      return
    }
    templateIdRef.current = id!
    let cancelled = false
    ;(async () => {
      setLoading(true)
      try {
        const [tplRes, projRes] = await Promise.all([
          jsTemplateService.getTemplate(id!),
          jsTemplateService.getProject(id!),
        ])
        if (cancelled) return
        if (tplRes.code !== 200 || !tplRes.data) {
          notifyApiResult(tplRes, { silentSuccess: true })
          navigate('/js-templates')
          return
        }
        const tpl = tplRes.data
        setTemplateId(tpl.id)
        setJsSourceId(tpl.jsSourceId)
        templateIdRef.current = tpl.id
        setTemplateName(tpl.name)

        if (projRes.code === 200 && projRes.data?.prefix) {
          setStorageHint(`对象存储 · ${projRes.data.prefix}`)
        } else if (tpl.content?.includes('"storage":"object"')) {
          try {
            const meta = JSON.parse(tpl.content) as { prefix?: string }
            if (meta.prefix) setStorageHint(`对象存储 · ${meta.prefix}`)
          } catch {
            /* ignore */
          }
        }

        let loaded: PetProjectV1
        const draft = loadPetDraft(tpl.id)
        if (draft) {
          loaded = draft
        } else if (projRes.code === 200 && projRes.data?.files && Object.keys(projRes.data.files).length > 0) {
          const { entry, files } = projRes.data
          loaded = projectFromApi(entry, files)
        } else {
          loaded = createProjectFromTemplate(DEFAULT_STARTER_TEMPLATE_ID, tpl.name)
        }

        if (isSpriteProject(loaded)) {
          loaded = await finalizeSpriteProject(loaded, tpl.name, starterTemplateId)
        }
        if (!cancelled) setProject(loaded)
      } catch (e: unknown) {
        const msg = (e as { msg?: string })?.msg
        showAlert(msg || '加载桌宠项目失败', 'error')
        navigate('/js-templates')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [id, isNewRoute, navigate, starterTemplateId])

  const handleProjectChange = useCallback(
    (next: PetProjectV1) => {
      setProject(next)
      const tid = resolveTemplateId(id, templateId, templateIdRef.current)
      if (!tid) return
      if (draftTimer.current) window.clearTimeout(draftTimer.current)
      draftTimer.current = window.setTimeout(() => savePetDraft(tid, next), 400)
    },
    [id, templateId],
  )

  const handleSave = useCallback(async () => {
    if (savingRef.current || loading) return

    const name = petNameFromProject(project, templateName).trim() || templateName.trim()
    if (!name) {
      showAlert('请在 manifest.json 中设置 name 或填写项目名称', 'warning')
      return
    }

    const readme = project.files[PROJECT_FILES.readme] || ''
    const projectPayload = {
      name,
      usage: readme,
      entry: project.entry || PROJECT_FILES.entry,
      files: project.files,
    }

    savingRef.current = true
    setSaving(true)
    try {
      let tid = resolveTemplateId(id, templateId, templateIdRef.current)

      if (!tid) {
        const res = await jsTemplateService.createStudioProject(projectPayload)
        if (!notifyApiResult(res, { successMessage: '桌宠项目已创建并保存到对象存储' })) return
        if (!res.data?.template?.id) return
        tid = res.data.template.id
        setTemplateId(tid)
        setJsSourceId(res.data.template.jsSourceId)
        templateIdRef.current = tid
        if (res.data.prefix) setStorageHint(`对象存储 · ${res.data.prefix}`)
        clearPetDraft(tid)
        setSavedRevision((r) => r + 1)
        navigate(`/js-templates/${tid}/edit`, { replace: true })
        return
      }

      const saveRes = await jsTemplateService.saveProject(tid, projectPayload)
      if (!notifyApiResult(saveRes, { successMessage: '已保存到对象存储' })) return

      if (saveRes.data?.prefix) setStorageHint(`对象存储 · ${saveRes.data.prefix}`)
      clearPetDraft(tid)
      setSavedRevision((r) => r + 1)
    } catch (e: unknown) {
      notifyApiError(e, '保存失败')
    } finally {
      savingRef.current = false
      setSaving(false)
    }
  }, [project, templateName, id, templateId, loading, navigate])

  const missingSpriteAssets = useMemo(
    () => isSpriteProject(project) && !hasSpriteAssets(project),
    [project],
  )

  const handleImportSpriteReadme = useCallback(() => {
    setProject(ensureSpriteAssetsInProject(project))
    showAlert('已添加 assets/sprites/README.md，请上传 PNG 帧图', 'info')
  }, [project])

  if (loading) {
    return (
      <div className="h-screen flex items-center justify-center bg-[#1e1e1e] text-[#858585]">
        加载项目中…
      </div>
    )
  }

  return (
    <PetStudioLayout
      project={project}
      templateName={templateName}
      isNew={isNewRoute && !templateIdRef.current}
      saving={saving}
      savedRevision={savedRevision}
      storageHint={storageHint}
      jsSourceId={jsSourceId}
      templateId={templateId}
      starterTemplateLabel={isNewRoute ? `${starterTemplate.badge} · ${starterTemplate.name}` : undefined}
      missingModelAssets={missingSpriteAssets}
      importingModelAssets={false}
      onImportModelAssets={handleImportSpriteReadme}
      onProjectChange={handleProjectChange}
      onTemplateNameChange={setTemplateName}
      onSave={handleSave}
      onBack={() => navigate('/js-templates')}
    />
  )
}
