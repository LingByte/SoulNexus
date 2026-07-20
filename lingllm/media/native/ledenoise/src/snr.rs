//! Lightweight time-domain noise-floor SNR (WebRTC-style minimum statistics).
//! FFT-free; suitable for 8/16 kHz SIP 20 ms frames via C ABI.

const EPS: f64 = 1e-12;
const NOISE_ALPHA: f64 = 0.05;
const NOISE_BETA: f64 = 0.25;
const SNR_ALPHA: f64 = 0.15;

pub struct NoiseSnr {
    sample_rate: usize,
    noise_power: f64,
    snr_ema: f64,
    have_noise: bool,
    have_snr: bool,
    frames: u32,
}

impl NoiseSnr {
    pub fn new(sample_rate: usize) -> Self {
        let sr = if sample_rate == 0 { 8000 } else { sample_rate };
        Self {
            sample_rate: sr,
            noise_power: 1e-6,
            snr_ema: 0.0,
            have_noise: false,
            have_snr: false,
            frames: 0,
        }
    }

    pub fn sample_rate(&self) -> usize {
        self.sample_rate
    }

    pub fn reset(&mut self) {
        self.noise_power = 1e-6;
        self.snr_ema = 0.0;
        self.have_noise = false;
        self.have_snr = false;
        self.frames = 0;
    }

    /// Observe one mono PCM16 frame; returns smoothed SNR in dB (or NaN until ready).
    pub fn process(&mut self, pcm: &[i16]) -> f64 {
        if pcm.is_empty() {
            return if self.have_snr {
                self.snr_ema
            } else {
                f64::NAN
            };
        }
        let mut sum = 0.0f64;
        for &s in pcm {
            let v = f64::from(s) / 32768.0;
            sum += v * v;
        }
        let mut frame_power = sum / pcm.len() as f64;
        if frame_power < EPS {
            frame_power = EPS;
        }
        self.frames = self.frames.saturating_add(1);

        if !self.have_noise {
            self.noise_power = frame_power;
            self.have_noise = true;
        } else if frame_power < self.noise_power {
            self.noise_power =
                (1.0 - NOISE_BETA) * self.noise_power + NOISE_BETA * frame_power;
        } else if frame_power < self.noise_power * 4.0 {
            self.noise_power =
                (1.0 - NOISE_ALPHA) * self.noise_power + NOISE_ALPHA * frame_power;
        }
        if self.noise_power < EPS {
            self.noise_power = EPS;
        }

        let mut snr = 10.0 * (frame_power / self.noise_power).log10();
        if snr < -10.0 {
            snr = -10.0;
        }
        if snr > 60.0 {
            snr = 60.0;
        }
        if !self.have_snr {
            self.snr_ema = snr;
            self.have_snr = true;
        } else {
            self.snr_ema = (1.0 - SNR_ALPHA) * self.snr_ema + SNR_ALPHA * snr;
        }
        self.snr_ema
    }

    pub fn snr_db(&self) -> f64 {
        if self.have_snr {
            self.snr_ema
        } else {
            f64::NAN
        }
    }

    pub fn ready(&self) -> bool {
        self.have_snr
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn speech_over_quiet_has_higher_snr() {
        let mut m = NoiseSnr::new(8000);
        let quiet: Vec<i16> = (0..160).map(|i| (40.0 * (i as f64).sin()) as i16).collect();
        for _ in 0..8 {
            m.process(&quiet);
        }
        let speech: Vec<i16> = (0..160)
            .map(|i| (12000.0 * (2.0 * std::f64::consts::PI * i as f64 / 20.0).sin()) as i16)
            .collect();
        for _ in 0..30 {
            m.process(&speech);
        }
        assert!(m.snr_db() > 15.0, "snr={}", m.snr_db());
    }
}
