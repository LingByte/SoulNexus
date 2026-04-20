# SoulNexus 靈樞

<div align="center">
<div align="center">
  <img src="docs/logo.png" alt="LingEcho Logo" width="100" height="110">
</div>

**智慧語音互動平台 - 讓 AI 擁有真實聲音**

[![Go Version](https://img.shields.io/badge/Go-1.25.1-blue.svg)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18.2.0-61dafb.svg)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.2.2-3178c6.svg)](https://www.typescriptlang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]()
[![線上示範](https://img.shields.io/badge/線上示範-lingecho.com-brightgreen.svg)](https://lingecho.com)

[English](README.md) | [简体中文](README_CN.md) | [繁體中文](README_TW.md) | [日本語](README_JA.md) | [Français](README_FR.md)

### 🌐 線上示範

**立即體驗 LingEcho**: [https://lingecho.com](https://lingecho.com)

</div>

---

## 📖 專案簡介

SoulNexus 靈樞是一個基於 Go + React 的企業級智慧語音互動平台，提供完整的 AI 語音解決方案。整合語音辨識（ASR）、語音合成（TTS）、大型語言模型（LLM）與即時通訊能力，支援即時通話、聲音克隆、知識庫管理、工作流自動化、設備管理、告警、帳單與組織管理等功能。

## ✨ 核心特性

- AI 角色即時通話（WebRTC 低延遲）
- 聲音克隆與訓練
- 視覺化工作流自動化
- 知識庫檢索與 AI 分析
- 設備管理與 OTA 升級
- 告警、帳單與配額管理
- 組織與多租戶協作
- MCP / ASR-TTS / VAD / 聲紋辨識服務

## 🚀 快速開始

### 環境需求

- Go >= 1.24.0
- Node.js >= 18
- npm >= 8 或 pnpm >= 8
- Python >= 3.10（選用服務）
- Docker & Docker Compose（建議）

### 建議安裝（Docker Compose）

```bash
git clone https://github.com/LingByte/SoulNexus-App.git
cd LingEcho-App
cp server/env.example .env
docker-compose up -d
```

服務入口：
- 前端：http://localhost:7072
- API：http://localhost:7072/api
- API 文件：http://localhost:7072/api/docs

## 📚 文件

- [安裝指南](docs/installation_CN.md)
- [功能文件](docs/features_CN.md)
- [架構文件](docs/architecture_CN.md)
- [開發指南](docs/development_CN.md)
- [服務文件](docs/services_CN.md)

## 🤝 貢獻

歡迎提交 Issue 與 PR。請先閱讀開發指南並遵循既有程式風格與提交規範。
