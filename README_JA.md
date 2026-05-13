# SoulNexus - インテリジェント音声対話プラットフォーム

<div align="center">
<div align="center">
  <img src="docs/logo.png" alt="SoulNexus Logo" width="100" height="110">
</div>

**AI に本物の声を与える音声対話基盤**

[![Go Version](https://img.shields.io/badge/Go-1.25.1-blue.svg)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18.2.0-61dafb.svg)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.2.2-3178c6.svg)](https://www.typescriptlang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]()
[![Live Demo](https://img.shields.io/badge/Live%20Demo-lingecho.com-brightgreen.svg)](https://lingecho.com)

[English](README.md) | [简体中文](README_CN.md) | [繁體中文](README_TW.md) | [日本語](README_JA.md) | [Français](README_FR.md)

### 🌐 デモ

**SoulNexus をオンラインで体験**: [https://lingecho.com](https://lingecho.com)

</div>

---

## 📖 プロジェクト概要

SoulNexus は Go + React ベースのエンタープライズ向け音声対話プラットフォームです。音声認識（ASR）、音声合成（TTS）、LLM、リアルタイム通信を統合し、リアルタイム通話、音声クローン、ナレッジベース、ワークフロー自動化、デバイス管理、アラート、課金などを提供します。

## ✨ 主な機能

- WebRTC ベースの低遅延リアルタイム通話
- 音声クローンと学習
- ビジュアルなワークフローデザイナー
- ナレッジベース管理と AI 検索
- デバイス管理と OTA 更新
- アラート、課金、クォータ管理
- 組織/マルチテナント管理
- MCP / ASR-TTS / VAD / 音紋認識サービス

## 🚀 クイックスタート

### 要件

- Go >= 1.24.0
- Node.js >= 18
- npm >= 8 または pnpm >= 8
- Python >= 3.10（オプションサービス）
- Docker & Docker Compose（推奨）

### Docker Compose（推奨）

```bash
git clone https://github.com/LingByte/SoulNexus-App.git
cd SoulNexus-App
cp server/env.example .env
docker-compose up -d
```

アクセス先：
- フロントエンド：http://localhost:7072
- API：http://localhost:7072/api
- API ドキュメント：http://localhost:7072/api/docs

## 📚 ドキュメント

- [インストールガイド](docs/installation.md)
- [機能一覧](docs/features.md)
- [アーキテクチャ](docs/architecture.md)
- [開発ガイド](docs/development.md)
- [サービス構成](docs/services.md)

## 🤝 コントリビュート

Issue / PR を歓迎します。開発ガイドを確認のうえ、既存のコーディング規約に従ってください。
