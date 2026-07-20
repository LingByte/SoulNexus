#!/usr/bin/env bash
# Split working tree into many small commits. Skips empty stages.
set -euo pipefail
cd "$(dirname "$0")/.."

commit_msg() {
  local msg="$1"
  shift
  local paths=("$@")
  if ((${#paths[@]} == 0)); then return 0; fi
  git add -A -- "${paths[@]}" 2>/dev/null || true
  if git diff --cached --quiet; then
    echo "— skip (no staged): $msg"
    return 0
  fi
  git commit -m "$msg"
  echo "✓ $msg"
}

# --- Root & infra ---
commit_msg "chore: update Makefile and container build files" Makefile Dockerfile docker-compose.yml .dockerignore
commit_msg "chore: add golangci config and update gitignore" .golangci.yml .gitignore LICENSE
commit_msg "ci: add GitHub Actions workflows" .github
commit_msg "deploy: add helm, nginx, and entrypoint assets" deploy
commit_msg "docs: rewrite README and add zh guide" README.md README_zh.md
commit_msg "docs: add deployment and operations guides" docs/deployment.md docs/ops-single-node.md docs/distributed.md docs/commercialization.md
commit_msg "docs: add architecture and refactor notes" docs/architecture-model.md docs/refactor-architecture.md docs/refactor-history.md docs/refactor-progress.md docs/refactor-progress-zh.md docs/refactor-rfc.md docs/project-overview.md docs/dialog-engine-migration.md
commit_msg "docs: add feature, MCP, and NLU documentation" docs/feature-recommendations.md docs/mcp-market.md docs/mcp-tenant-tools.md docs/nlu.md docs/knowledge-latency.md docs/knowledge-ops-closed-loop-zh.md docs/future-development.md docs/design-critique.md docs/roadmap.md docs/RNNOISE.md docs/soulpet-package-spec.md docs/sip_gap_analysis.md
commit_msg "docs: update architecture images and diagrams" docs/ArchitectureDiagram.png docs/core.png docs/core-process.png docs/applicationclient.png docs/logo.png docs/page-workflow.png docs/page-voice-clone.png docs/page-debug-assistant.png docs/page-js-template.png docs/lingoroutine.excalidraw docs/project.excalidraw docs/deployBtns docs/voice-server-core.png docs/sse.png

# --- Remove legacy admin ---
commit_msg "chore: remove legacy admin console application" admin

# --- Go module & cmd ---
commit_msg "build: update Go module to SoulNexus" go.mod go.sum
commit_msg "feat(cmd): update server main and bootstrap" cmd/server cmd/bootstrap
commit_msg "feat(cmd): add backfill and schema tools" cmd/backfill cmd/tools
commit_msg "test(cmd): add bootstrap unit tests" cmd/bootstrap/banner_test.go cmd/bootstrap/database_test.go cmd/bootstrap/jwks_test.go cmd/bootstrap/printer_test.go cmd/bootstrap/seeds_test.go

# --- internal ---
commit_msg "refactor(internal): update handlers server package" internal/handlers/server
commit_msg "refactor(internal): update voice and dialog handlers" internal/handlers/voice internal/handlers/voice_session.go internal/handlers/voice_clone.go internal/handlers/voice_catalog.go internal/handlers/voice_catalog_test.go internal/handlers/voiceprint.go internal/handlers/voiceprint_speaker.go internal/handlers/voices.go
commit_msg "refactor(internal): update workflow HTTP handlers" internal/handlers/workflow_user.go internal/handlers/workflow_triggers.go internal/handlers/workflow_plugins.go internal/handlers/workflow_instances.go internal/handlers/workflows.go
commit_msg "refactor(internal): update tenant and auth handlers" internal/handlers/tenants.go internal/handlers/tenant_users.go internal/handlers/tenant_users_me.go internal/handlers/tenant_organization.go internal/handlers/tenant_webhooks.go internal/handlers/tenant_nlu.go internal/handlers/tenant_nlu_train.go internal/handlers/tenant_llm_stream.go
commit_msg "refactor(internal): update remaining HTTP handlers" internal/handlers
commit_msg "refactor(internal): update models and listeners" internal/models internal/listeners
commit_msg "refactor(internal): update config, tasks, and workflow engine" internal/config internal/task internal/workflow internal/constants internal/signals internal/middleware internal/services internal/storage internal/embeddings internal/i18n internal/metrics internal/notify internal/oplog internal/platform internal/rbac internal/repository internal/scheduler internal/search internal/seed internal/signal internal/tenant internal/types internal/utils internal/version internal/voice internal/webhook internal/ws

# --- pkg (split) ---
commit_msg "refactor(pkg): update dialog engine" pkg/dialog
commit_msg "refactor(pkg): update voice server stack" pkg/voiceserver
commit_msg "refactor(pkg): update voice, workflow, and knowledge" pkg/voice pkg/voicedialog pkg/voiceprint pkg/workflow pkg/knowledge
commit_msg "refactor(pkg): update LLM, agent, and middleware" pkg/llm pkg/agent pkg/middleware
commit_msg "refactor(pkg): update notification, censor, and billing" pkg/notification pkg/censor pkg/billing
commit_msg "refactor(pkg): update shared packages" pkg

# --- lingllm & assets ---
commit_msg "refactor(lingllm): sync submodule package changes" lingllm
commit_msg "chore: update bundled static and resource assets" static resources examples packages

# --- desktop pet ---
commit_msg "feat(desktop-pet): update Electron client and settings UI" desktop-pet

# --- web toolchain ---
commit_msg "build(web): update Vite, Tailwind, and package config" web/package.json web/package-lock.json web/pnpm-lock.yaml web/vite.config.ts web/tailwind.config.js web/tsconfig.json web/tsconfig.node.json web/postcss.config.js web/index.html web/.env.example web/eslint.config.js web/components.json
commit_msg "feat(web): update app shell, routes, and providers" web/src/App.tsx web/src/AppRoutes.tsx web/src/main.tsx web/src/index.css web/src/vite-env.d.ts web/src/providers web/src/context web/src/contexts web/src/i18n web/src/constants web/src/types web/src/lib web/src/styles web/src/hooks web/src/stores
commit_msg "feat(web): update i18n locale catalogs" web/src/locales web/src/i18n
commit_msg "feat(web): update API client modules" web/src/api
commit_msg "feat(web): update shared utils and theme" web/src/utils web/src/config
commit_msg "feat(web): add marketing landing and home components" web/src/pages/LandingPage.tsx web/src/pages/Home.tsx web/src/pages/NotFound.tsx web/src/components/Home web/public/images
commit_msg "feat(web): update auth layout and carousel" web/src/components/Auth web/src/components/auth web/src/pages/TenantAuth.tsx web/src/pages/TenantLogin.tsx web/src/pages/TenantRegister.tsx web/src/pages/ResetPassword.tsx web/src/pages/AccountDeletionRequest.tsx web/src/pages/AccountDeletionRevoke.tsx
commit_msg "feat(web): update layout, sidebar, and navigation" web/src/components/Layout web/src/components/layout web/src/components/PWA web/src/components/pwa web/src/components/error-boundary web/src/components/dev
commit_msg "feat(web): update workflow editor UI" web/src/components/Workflow web/src/components/workflow web/src/pages/WorkflowEditorPage.tsx web/src/pages/WorkflowManager.tsx web/src/pages/WorkflowPluginMarket.tsx web/src/pages/NodePluginMarket.tsx
commit_msg "feat(web): update voice and assistant UI" web/src/components/Voice web/src/components/voice web/src/components/assistant web/src/pages/VoiceCloneManager.tsx web/src/pages/VoiceprintManager.tsx web/src/pages/VoiceprintManagement.tsx web/src/pages/PlatformVoiceManagement.tsx web/src/pages/PlatformVoiceprintManagement.tsx web/src/pages/VoiceAssistant.tsx web/src/pages/assistants web/src/pages/AssistantManager.tsx web/src/pages/AssistantTools.tsx
commit_msg "feat(web): update knowledge base pages" web/src/pages/knowledge web/src/pages/KnowledgeBaseDetail.tsx web/src/pages/KnowledgeNamespaces.tsx web/src/pages/KnowledgeDocumentEdit.tsx web/src/pages/KnowledgeDocumentChunks.tsx web/src/pages/KnowledgeOpsTabs.tsx web/src/pages/KnowledgeEvalTab.tsx web/src/pages/KnowledgeSyncTab.tsx web/src/components/knowledge
commit_msg "feat(web): update tenant and platform admin pages" web/src/pages/TenantManagement.tsx web/src/pages/TenantMembers.tsx web/src/pages/TenantDepartments.tsx web/src/pages/TenantRolePermissions.tsx web/src/pages/TenantBills.tsx web/src/pages/TenantAiConfig.tsx web/src/pages/PlatformAdmins.tsx web/src/pages/SystemConfigs.tsx web/src/pages/OverviewDashboard.tsx
commit_msg "feat(web): update billing, access, and ops pages" web/src/pages/Billing.tsx web/src/pages/BillingPlan.tsx web/src/pages/AccessKeys.tsx web/src/pages/UsageMetrics.tsx web/src/pages/OperationLogs.tsx web/src/pages/AIInvocationLogs.tsx web/src/pages/Inbox.tsx web/src/pages/NotificationCenter.tsx
commit_msg "feat(web): update MCP, market, and JS template pages" web/src/pages/McpMarket.tsx web/src/pages/Market.tsx web/src/pages/JSTemplateManager.tsx web/src/pages/JSTemplateEditorPage.tsx web/src/pages/JSTemplateNew.tsx web/src/pages/JSTemplateEdit.tsx web/src/components/js-template web/src/components/mcp
commit_msg "feat(web): update NLU, profile, and misc console pages" web/src/pages/NluModelsPage.tsx web/src/pages/NluModelDetailPage.tsx web/src/pages/Profile.tsx web/src/pages/profile web/src/components/profile web/src/pages/ComponentsDemo.tsx web/src/pages/CredentialManager.tsx web/src/pages/DeviceManagement.tsx web/src/pages/DeviceDetail.tsx web/src/pages/pet-market web/src/pages/platform
commit_msg "feat(web): update remaining console pages" web/src/pages
commit_msg "feat(web): update UI component library" web/src/components/ui web/src/components/UI web/src/components/Captcha web/src/components/Chart web/src/components/Data web/src/components/Forms web/src/components/SEO web/src/components/UX web/src/components/Animations web/src/components/Notifications web/src/components/Tenant web/src/components/WorldInfo web/src/components/Group web/src/components/Inbox web/src/components/Mail web/src/components/Platform web/src/components/Settings web/src/components/Table web/src/components/Tabs web/src/components/WorkflowMarket web/src/components/embed web/src/components/icons web/src/components/loading web/src/components/markdown web/src/components/modal web/src/components/search web/src/components/select web/src/components/table web/src/components/tags web/src/components/tooltip web/src/components/upload web/src/components/wizard web/src/components
commit_msg "chore(web): update public assets and PWA files" web/public

# --- Cursor rules/skills (if tracked) ---
commit_msg "chore: add Cursor rules and project skills" .cursor/rules .cursor/skills

# --- Sweep leftovers in chunks ---
part=1
while ! git diff --quiet || ! git diff --cached --quiet || [ -n "$(git status --porcelain)" ]; do
  mapfile -t rest < <(git status --porcelain | sed 's/^.. //' | sed 's/^"//;s/"$//' | head -60)
  if ((${#rest[@]} == 0)); then break; fi
  git add -A -- "${rest[@]}" 2>/dev/null || git add -A
  if git diff --cached --quiet; then break; fi
  git commit -m "chore: SoulNexus refactor remainder (part ${part})"
  echo "✓ remainder part ${part} (${#rest[@]} paths)"
  part=$((part + 1))
  if ((part > 30)); then
    echo "Stopping sweep after 30 remainder commits; check git status."
    break
  fi
done

echo ""
echo "Done. Remaining:"
git status -sb | head -20
