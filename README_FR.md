# SoulNexus - Plateforme d'interaction vocale intelligente

<div align="center">
<div align="center">
  <img src="docs/logo.png" alt="SoulNexus Logo" width="100" height="110">
</div>

**Donner une vraie voix a l'IA**

[![Go Version](https://img.shields.io/badge/Go-1.25.1-blue.svg)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18.2.0-61dafb.svg)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.2.2-3178c6.svg)](https://www.typescriptlang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]()
[![Demo](https://img.shields.io/badge/Live%20Demo-lingecho.com-brightgreen.svg)](https://lingecho.com)

[English](README.md) | [简体中文](README_CN.md) | [繁體中文](README_TW.md) | [日本語](README_JA.md) | [Français](README_FR.md)

### 🌐 Demo en ligne

**Essayez SoulNexus** : [https://lingecho.com](https://lingecho.com)

</div>

---

## 📖 Apercu du projet

SoulNexus est une plateforme d'interaction vocale de niveau entreprise basee sur Go + React. Elle integre ASR, TTS, LLM et communication temps reel pour offrir : appels vocaux IA, clonage de voix, base de connaissances, automatisation des workflows, gestion des appareils, alertes et facturation.

## ✨ Fonctionnalites principales

- Appels vocaux IA en temps reel (WebRTC)
- Clonage et entrainement de voix
- Concepteur de workflows visuel
- Gestion de base de connaissances
- Gestion des appareils et OTA
- Alertes, quotas et facturation
- Gestion d'organisation multi-tenant
- Services MCP / ASR-TTS / VAD / empreinte vocale

## 🚀 Demarrage rapide

### Prerequis

- Go >= 1.24.0
- Node.js >= 18
- npm >= 8 ou pnpm >= 8
- Python >= 3.10 (services optionnels)
- Docker & Docker Compose (recommande)

### Installation recommandee (Docker Compose)

```bash
git clone https://github.com/LingByte/SoulNexus-App.git
cd SoulNexus-App
cp server/env.example .env
docker-compose up -d
```

Acces :
- Frontend : http://localhost:7072
- API : http://localhost:7072/api
- Docs API : http://localhost:7072/api/docs

## 📚 Documentation

- [Guide d'installation](docs/installation.md)
- [Fonctionnalites](docs/features.md)
- [Architecture](docs/architecture.md)
- [Guide de developpement](docs/development.md)
- [Documentation des services](docs/services.md)

## 🤝 Contribution

Les contributions sont les bienvenues via Issue et Pull Request. Merci de lire le guide de developpement avant de soumettre des changements.
