// Web Audio API 音频效果系统
class AudioEffectManager {
    private audioContext: AudioContext | null = null
    private masterGain: GainNode | null = null
    private isEnabled = true
    private volume = 0.3

    constructor() {
        this.initAudioContext()
    }

    private async initAudioContext() {
        try {
            this.audioContext = new (window.AudioContext || (window as any).webkitAudioContext)()
            this.masterGain = this.audioContext.createGain()
            this.masterGain.connect(this.audioContext.destination)
            this.masterGain.gain.value = this.volume
        } catch (error) {
            console.warn('Web Audio API not supported:', error)
        }
    }

    // 启用/禁用音频
    setEnabled(enabled: boolean) {
        this.isEnabled = enabled
    }

    // 设置音量
    setVolume(volume: number) {
        this.volume = Math.max(0, Math.min(1, volume))
        if (this.masterGain) {
            this.masterGain.gain.value = this.volume
        }
    }

    // 播放点击音效
    playClickSound() {
        if (!this.isEnabled || !this.audioContext || !this.masterGain) return

        const oscillator = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator.connect(gainNode)
        gainNode.connect(this.masterGain)

        oscillator.frequency.setValueAtTime(800, this.audioContext.currentTime)
        oscillator.frequency.exponentialRampToValueAtTime(400, this.audioContext.currentTime + 0.1)

        gainNode.gain.setValueAtTime(0.3, this.audioContext.currentTime)
        gainNode.gain.exponentialRampToValueAtTime(0.01, this.audioContext.currentTime + 0.1)

        oscillator.start(this.audioContext.currentTime)
        oscillator.stop(this.audioContext.currentTime + 0.1)
    }

    // 播放成功音效
    playSuccessSound() {
        if (!this.isEnabled || !this.audioContext || !this.masterGain) return

        const oscillator1 = this.audioContext.createOscillator()
        const oscillator2 = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator1.connect(gainNode)
        oscillator2.connect(gainNode)
        gainNode.connect(this.masterGain)

        oscillator1.frequency.setValueAtTime(523.25, this.audioContext.currentTime) // C5
        oscillator2.frequency.setValueAtTime(659.25, this.audioContext.currentTime) // E5

        oscillator1.frequency.setValueAtTime(659.25, this.audioContext.currentTime + 0.1) // E5
        oscillator2.frequency.setValueAtTime(783.99, this.audioContext.currentTime + 0.1) // G5

        oscillator1.frequency.setValueAtTime(783.99, this.audioContext.currentTime + 0.2) // G5
        oscillator2.frequency.setValueAtTime(1046.50, this.audioContext.currentTime + 0.2) // C6

        gainNode.gain.setValueAtTime(0.2, this.audioContext.currentTime)
        gainNode.gain.exponentialRampToValueAtTime(0.01, this.audioContext.currentTime + 0.5)

        oscillator1.start(this.audioContext.currentTime)
        oscillator2.start(this.audioContext.currentTime)
        oscillator1.stop(this.audioContext.currentTime + 0.5)
        oscillator2.stop(this.audioContext.currentTime + 0.5)
    }

    // 播放错误音效
    playErrorSound() {
        if (!this.isEnabled || !this.audioContext || !this.masterGain) return

        const oscillator = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator.connect(gainNode)
        gainNode.connect(this.masterGain)

        oscillator.frequency.setValueAtTime(200, this.audioContext.currentTime)
        oscillator.frequency.exponentialRampToValueAtTime(100, this.audioContext.currentTime + 0.3)

        gainNode.gain.setValueAtTime(0.3, this.audioContext.currentTime)
        gainNode.gain.exponentialRampToValueAtTime(0.01, this.audioContext.currentTime + 0.3)

        oscillator.start(this.audioContext.currentTime)
        oscillator.stop(this.audioContext.currentTime + 0.3)
    }

    // 播放警告音效
    playWarningSound() {
        if (!this.isEnabled || !this.audioContext || !this.masterGain) return

        const oscillator = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator.connect(gainNode)
        gainNode.connect(this.masterGain)

        // 创建两个短促的音调
        oscillator.frequency.setValueAtTime(600, this.audioContext.currentTime)
        oscillator.frequency.setValueAtTime(600, this.audioContext.currentTime + 0.1)
        oscillator.frequency.setValueAtTime(600, this.audioContext.currentTime + 0.2)

        gainNode.gain.setValueAtTime(0, this.audioContext.currentTime)
        gainNode.gain.setValueAtTime(0.2, this.audioContext.currentTime + 0.01)
        gainNode.gain.setValueAtTime(0, this.audioContext.currentTime + 0.05)
        gainNode.gain.setValueAtTime(0.2, this.audioContext.currentTime + 0.11)
        gainNode.gain.setValueAtTime(0, this.audioContext.currentTime + 0.15)
        gainNode.gain.setValueAtTime(0.2, this.audioContext.currentTime + 0.21)
        gainNode.gain.setValueAtTime(0, this.audioContext.currentTime + 0.25)

        oscillator.start(this.audioContext.currentTime)
        oscillator.stop(this.audioContext.currentTime + 0.3)
    }

    // 播放信息音效
    playInfoSound() {
        if (!this.isEnabled || !this.audioContext || !this.masterGain) return

        const oscillator = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator.connect(gainNode)
        gainNode.connect(this.masterGain)

        oscillator.frequency.setValueAtTime(440, this.audioContext.currentTime) // A4
        oscillator.frequency.setValueAtTime(554.37, this.audioContext.currentTime + 0.1) // C#5

        gainNode.gain.setValueAtTime(0.2, this.audioContext.currentTime)
        gainNode.gain.exponentialRampToValueAtTime(0.01, this.audioContext.currentTime + 0.2)

        oscillator.start(this.audioContext.currentTime)
        oscillator.stop(this.audioContext.currentTime + 0.2)
    }

    // 播放悬停音效
    playHoverSound() {
        if (!this.isEnabled || !this.audioContext || !this.masterGain) return

        const oscillator = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator.connect(gainNode)
        gainNode.connect(this.masterGain)

        oscillator.frequency.setValueAtTime(1000, this.audioContext.currentTime)
        oscillator.frequency.exponentialRampToValueAtTime(1200, this.audioContext.currentTime + 0.05)

        gainNode.gain.setValueAtTime(0.1, this.audioContext.currentTime)
        gainNode.gain.exponentialRampToValueAtTime(0.01, this.audioContext.currentTime + 0.05)

        oscillator.start(this.audioContext.currentTime)
        oscillator.stop(this.audioContext.currentTime + 0.05)
    }

    // 播放页面切换音效
    playPageTransitionSound() {
        if (!this.isEnabled || !this.audioContext || !this.masterGain) return

        const oscillator = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator.connect(gainNode)
        gainNode.connect(this.masterGain)

        oscillator.frequency.setValueAtTime(200, this.audioContext.currentTime)
        oscillator.frequency.exponentialRampToValueAtTime(800, this.audioContext.currentTime + 0.3)

        gainNode.gain.setValueAtTime(0.2, this.audioContext.currentTime)
        gainNode.gain.exponentialRampToValueAtTime(0.01, this.audioContext.currentTime + 0.3)

        oscillator.start(this.audioContext.currentTime)
        oscillator.stop(this.audioContext.currentTime + 0.3)
    }

    // 播放魔法音效
    playMagicSound() {
        if (!this.isEnabled || !this.audioContext || !this.masterGain) return

        const oscillator1 = this.audioContext.createOscillator()
        const oscillator2 = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator1.connect(gainNode)
        oscillator2.connect(gainNode)
        gainNode.connect(this.masterGain)

        // 创建上升的音调
        oscillator1.frequency.setValueAtTime(261.63, this.audioContext.currentTime) // C4
        oscillator1.frequency.exponentialRampToValueAtTime(523.25, this.audioContext.currentTime + 0.5) // C5

        oscillator2.frequency.setValueAtTime(329.63, this.audioContext.currentTime) // E4
        oscillator2.frequency.exponentialRampToValueAtTime(659.25, this.audioContext.currentTime + 0.5) // E5

        gainNode.gain.setValueAtTime(0.15, this.audioContext.currentTime)
        gainNode.gain.exponentialRampToValueAtTime(0.01, this.audioContext.currentTime + 0.5)

        oscillator1.start(this.audioContext.currentTime)
        oscillator2.start(this.audioContext.currentTime)
        oscillator1.stop(this.audioContext.currentTime + 0.5)
        oscillator2.stop(this.audioContext.currentTime + 0.5)
    }

    // 播放粒子音效
    playParticleSound() {
        if (!this.isEnabled || !this.audioContext || !this.masterGain) return

        const oscillator = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator.connect(gainNode)
        gainNode.connect(this.masterGain)

        oscillator.frequency.setValueAtTime(800 + Math.random() * 400, this.audioContext.currentTime)
        oscillator.frequency.exponentialRampToValueAtTime(200, this.audioContext.currentTime + 0.1)

        gainNode.gain.setValueAtTime(0.1, this.audioContext.currentTime)
        gainNode.gain.exponentialRampToValueAtTime(0.01, this.audioContext.currentTime + 0.1)

        oscillator.start(this.audioContext.currentTime)
        oscillator.stop(this.audioContext.currentTime + 0.1)
    }
}

// 创建全局音频管理器实例
export const audioManager = new AudioEffectManager()

// 便捷函数
export const playClickSound = () => audioManager.playClickSound()
export const playSuccessSound = () => audioManager.playSuccessSound()
export const playErrorSound = () => audioManager.playErrorSound()
export const playWarningSound = () => audioManager.playWarningSound()
export const playInfoSound = () => audioManager.playInfoSound()
export const playHoverSound = () => audioManager.playHoverSound()
export const playPageTransitionSound = () => audioManager.playPageTransitionSound()
export const playMagicSound = () => audioManager.playMagicSound()
export const playParticleSound = () => audioManager.playParticleSound()

// 音频设置
export const setAudioEnabled = (enabled: boolean) => audioManager.setEnabled(enabled)
export const setAudioVolume = (volume: number) => audioManager.setVolume(volume)
