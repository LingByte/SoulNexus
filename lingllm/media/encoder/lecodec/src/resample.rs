use crate::{PcmBuf, Sample};
use std::f64::consts::PI as PI_F64;

pub struct Resampler {
    input_rate: usize,
    output_rate: usize,
    ratio: f64,
    coeffs: Vec<f32>,
    num_phases: usize,
    taps_per_phase: usize,
    history: Vec<f32>,
    current_pos: f64,
}

fn bessel_i0(x: f64) -> f64 {
    let mut sum = 1.0_f64;
    let mut term = 1.0_f64;
    let x_sq = x * x * 0.25;

    for m in 1..=30 {
        term *= x_sq / (m * m) as f64;
        sum += term;
        if term < 1e-15 * sum {
            break;
        }
    }
    sum
}

fn kaiser_window(n: usize, n_total: usize, beta: f64) -> f64 {
    if n_total <= 1 {
        return 1.0;
    }
    let alpha = (n_total - 1) as f64 / 2.0;
    let x = (n as f64 - alpha) / alpha;
    let arg = beta * (1.0 - x * x).sqrt();
    bessel_i0(arg) / bessel_i0(beta)
}

impl Resampler {
    pub fn new(input_rate: usize, output_rate: usize) -> Self {
        let ratio = output_rate as f64 / input_rate as f64;

        const NUM_PHASES: usize = 256;
        const TAPS_PER_PHASE: usize = 24;
        const KAISER_BETA: f64 = 7.0;

        let num_phases = NUM_PHASES;
        let taps_per_phase = TAPS_PER_PHASE;
        let filter_len = num_phases * taps_per_phase;

        let mut raw_coeffs = vec![0.0_f32; filter_len];

        let cutoff = if ratio < 1.0 {
            ratio * 0.5 * 0.95
        } else {
            0.5 * 0.95
        };

        let center = (taps_per_phase as f64 - 1.0) / 2.0;

        // Design the polyphase filter
        for p in 0..num_phases {
            let mut phase_coeffs = vec![0.0_f64; taps_per_phase];
            let mut sum = 0.0_f64;

            for t in 0..taps_per_phase {
                let x = t as f64 - center - (p as f64 / num_phases as f64);

                let sinc_val = if x.abs() < 1e-10 {
                    2.0 * cutoff
                } else {
                    let x_pi = x * PI_F64;
                    (x_pi * 2.0 * cutoff).sin() / x_pi
                };

                let full_filter_idx = t * num_phases + p;
                let window = kaiser_window(full_filter_idx, filter_len, KAISER_BETA);

                phase_coeffs[t] = sinc_val * window;
                sum += phase_coeffs[t];
            }

            for t in 0..taps_per_phase {
                let normalized = (phase_coeffs[t] / sum) as f32;
                raw_coeffs[p * taps_per_phase + t] = normalized;
            }
        }

        Self {
            input_rate,
            output_rate,
            ratio,
            coeffs: raw_coeffs,
            num_phases,
            taps_per_phase,
            history: vec![0.0; taps_per_phase],
            current_pos: 0.0,
        }
    }

    #[inline(always)]
    fn dot_product(a: &[f32], b: &[f32]) -> f32 {
        debug_assert_eq!(a.len(), 24);
        debug_assert_eq!(b.len(), 24);

        #[cfg(target_arch = "aarch64")]
        {
            // ARM NEON: 24 taps = 6 iterations of 4-wide vectors
            unsafe {
                use std::arch::aarch64::*;
                let mut sumv = vdupq_n_f32(0.0);
                for i in (0..24).step_by(4) {
                    let av = vld1q_f32(a.as_ptr().add(i));
                    let bv = vld1q_f32(b.as_ptr().add(i));
                    sumv = vfmaq_f32(sumv, av, bv);
                }
                vaddvq_f32(sumv)
            }
        }
        #[cfg(all(target_arch = "x86_64", target_feature = "avx"))]
        {
            unsafe {
                use std::arch::x86_64::*;
                let mut sumv = _mm256_setzero_ps();
                for i in (0..24).step_by(8) {
                    let av = _mm256_loadu_ps(a.as_ptr().add(i));
                    let bv = _mm256_loadu_ps(b.as_ptr().add(i));
                    sumv = _mm256_add_ps(sumv, _mm256_mul_ps(av, bv));
                }
                // Horizontal sum
                let x128 = _mm_add_ps(_mm256_extractf128_ps(sumv, 1), _mm256_castps256_ps128(sumv));
                let x64 = _mm_add_ps(x128, _mm_movehl_ps(x128, x128));
                let x32 = _mm_add_ss(x64, _mm_shuffle_ps(x64, x64, 0x55));
                _mm_cvtss_f32(x32)
            }
        }
        #[cfg(all(
            target_arch = "x86_64",
            target_feature = "sse2",
            not(target_feature = "avx")
        ))]
        {
            unsafe {
                use std::arch::x86_64::*;
                let mut sumv = _mm_setzero_ps();
                for i in (0..24).step_by(4) {
                    let av = _mm_loadu_ps(a.as_ptr().add(i));
                    let bv = _mm_loadu_ps(b.as_ptr().add(i));
                    sumv = _mm_add_ps(sumv, _mm_mul_ps(av, bv));
                }
                let x64 = _mm_add_ps(sumv, _mm_shuffle_ps(sumv, sumv, 0x4e));
                let x32 = _mm_add_ss(x64, _mm_shuffle_ps(x64, x64, 0x11));
                _mm_cvtss_f32(x32)
            }
        }
        #[cfg(not(any(
            target_arch = "aarch64",
            all(target_arch = "x86_64", target_feature = "sse2")
        )))]
        {
            a.iter().zip(b.iter()).map(|(x, y)| x * y).sum()
        }
    }

    pub fn resample(&mut self, input: &[Sample]) -> PcmBuf {
        if self.input_rate == self.output_rate {
            return input.to_vec();
        }

        let mut output = Vec::with_capacity((input.len() as f64 * self.ratio) as usize + 1);
        let inv_ratio = 1.0 / self.ratio;
        let taps = self.taps_per_phase;
        let num_phases_f = self.num_phases as f64;

        for &sample in input {
            self.history.copy_within(1..taps, 0);
            self.history[taps - 1] = sample as f32;

            while self.current_pos < 1.0 {
                let phase_idx = (self.current_pos * num_phases_f) as usize;
                let phase_idx = phase_idx.min(self.num_phases - 1); // Safety clamp
                let offset = phase_idx * taps;
                let phase_coeffs = &self.coeffs[offset..offset + taps];

                let out_sample = Self::dot_product(phase_coeffs, &self.history);

                output.push(out_sample.clamp(i16::MIN as f32, i16::MAX as f32) as i16);
                self.current_pos += inv_ratio;
            }
            self.current_pos -= 1.0;
        }

        output
    }

    pub fn reset(&mut self) {
        self.history.fill(0.0);
        self.current_pos = 0.0;
    }
}

pub fn resample(input: &[Sample], input_sample_rate: u32, output_sample_rate: u32) -> PcmBuf {
    if input_sample_rate == output_sample_rate {
        return input.to_vec();
    }
    let mut r = Resampler::new(input_sample_rate as usize, output_sample_rate as usize);
    r.resample(input)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::f32::consts::PI as PI_F32;
    use std::time::Instant;

    #[test]
    fn test_resample_8k_to_16k() {
        let mut resampler = Resampler::new(8000, 16000);
        let input = vec![1000i16; 80];
        let output = resampler.resample(&input);
        assert!(output.len() >= 150 && output.len() <= 170);
        for &s in &output[48..output.len().saturating_sub(48)] {
            assert!((s - 1000).abs() < 100, "Value {} is too far from 1000", s);
        }
    }

    #[test]
    fn test_resample_16k_to_8k() {
        let mut resampler = Resampler::new(16000, 8000);
        let input = vec![1000i16; 160];
        let output = resampler.resample(&input);
        assert!(output.len() >= 75 && output.len() <= 85);
        let skip = output.len() / 4;
        for &s in &output[skip..output.len() - skip] {
            assert!((s - 1000).abs() < 100, "Value {} is too far from 1000", s);
        }
    }

    #[test]
    fn test_frequency_response_downsample() {
        let mut resampler = Resampler::new(16000, 8000);
        let freq = 2000.0_f32; // Well below 4kHz Nyquist
        let samples: Vec<i16> = (0..160)
            .map(|i| ((i as f32 * freq * 2.0 * PI_F32 / 16000.0).sin() * 10000.0) as i16)
            .collect();

        let output = resampler.resample(&samples);

        // Output should have similar amplitude (allowing for some attenuation)
        let input_rms: f32 = samples
            .iter()
            .map(|&s| (s as f32).powi(2))
            .sum::<f32>()
            .sqrt()
            / samples.len() as f32;
        let output_rms: f32 = output
            .iter()
            .map(|&s| (s as f32).powi(2))
            .sum::<f32>()
            .sqrt()
            / output.len() as f32;

        assert!(
            output_rms > input_rms * 0.7,
            "Too much attenuation: input_rms={}, output_rms={}",
            input_rms,
            output_rms
        );
    }

    #[test]
    fn test_aliasing_suppression() {
        let mut resampler = Resampler::new(16000, 8000);
        let freq = 7000.0_f32; // Above 4kHz Nyquist of output
        let samples: Vec<i16> = (0..1600)
            .map(|i| ((i as f32 * freq * 2.0 * PI_F32 / 16000.0).sin() * 10000.0) as i16)
            .collect();

        let output = resampler.resample(&samples);

        let output_rms: f32 =
            (output.iter().map(|&s| (s as f32).powi(2)).sum::<f32>() / output.len() as f32).sqrt();
        let input_rms: f32 = 10000.0 / 1.414; // Expected RMS of sine wave with amplitude 10000

        assert!(
            output_rms < input_rms / 50.0,
            "Aliasing not sufficiently suppressed: output_rms={}",
            output_rms
        );
    }

    #[test]
    fn test_performance_48k_to_8k() {
        let mut resampler = Resampler::new(48000, 8000);
        let input = vec![0i16; 48000];

        let start = Instant::now();
        let iterations = 100;
        for _ in 0..iterations {
            let _ = resampler.resample(&input);
            resampler.reset();
        }
        let duration = start.elapsed();
        let per_second = duration.as_secs_f64() / iterations as f64;
        println!(
            "Resampling 1s of 48kHz to 8kHz (24 taps) took: {:.4}ms",
            per_second * 1000.0
        );
        assert!(
            per_second < 0.1,
            "Performance regression: {}ms",
            per_second * 1000.0
        );
    }

    #[test]
    fn test_continuity_between_chunks() {
        let input_rate = 16000;
        let output_rate = 8000;

        let freq = 1000.0_f32;
        let total_samples = 3200;
        let input: Vec<i16> = (0..total_samples)
            .map(|i| ((i as f32 * freq * 2.0 * PI_F32 / input_rate as f32).sin() * 5000.0) as i16)
            .collect();

        let mut resampler1 = Resampler::new(input_rate, output_rate);
        let output1 = resampler1.resample(&input);

        let mut resampler2 = Resampler::new(input_rate, output_rate);
        let mid = input.len() / 2;
        let mut output2 = resampler2.resample(&input[..mid]);
        output2.extend_from_slice(&resampler2.resample(&input[mid..]));

        assert_eq!(output1.len(), output2.len(), "Output lengths differ");

        let max_diff: i16 = output1
            .iter()
            .zip(output2.iter())
            .map(|(a, b)| (a - b).abs())
            .max()
            .unwrap_or(0);

        assert!(
            max_diff < 100,
            "Large discontinuity between chunks: max_diff={}",
            max_diff
        );
    }
}
