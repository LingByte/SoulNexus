//! Batch RNNoise denoise at arbitrary mono PCM rates (resample ↔ 48 kHz).
//! Port of voice-engine `media::denoiser::NoiseReducer`.

use crate::resample::Resampler;
use crate::{PcmBuf, Sample};
use anyhow::Result;
use nnnoiseless::DenoiseState;

pub struct NoiseReducer {
    sample_rate: usize,
    up: Resampler,   // input → 48 kHz
    down: Resampler, // 48 kHz → input
    denoise: Box<DenoiseState<'static>>,
}

impl NoiseReducer {
    pub fn new(input_sample_rate: usize) -> Result<Self> {
        let sr = if input_sample_rate == 0 {
            16000
        } else {
            input_sample_rate
        };
        Ok(Self {
            sample_rate: sr,
            up: Resampler::new(sr, 48000),
            down: Resampler::new(48000, sr),
            denoise: DenoiseState::new(),
        })
    }

    pub fn sample_rate(&self) -> usize {
        self.sample_rate
    }

    /// Denoise mono PCM16. Matches voice-engine: up→48k, RNNoise frames (zero-pad
    /// remainder), down to original rate. Output length may differ slightly.
    pub fn process(&mut self, input: &[Sample]) -> PcmBuf {
        if input.is_empty() {
            return Vec::new();
        }
        if self.sample_rate == 48000 {
            return self.denoise_48k_f32(input);
        }

        let samples = self.up.resample(input);
        let cleaned = self.denoise_48k_f32(&samples);
        self.down.resample(&cleaned)
    }

    fn denoise_48k_f32(&mut self, samples: &[Sample]) -> PcmBuf {
        let input_size = samples.len();
        if input_size == 0 {
            return Vec::new();
        }
        let frame = DenoiseState::FRAME_SIZE;
        let mut output_buf = vec![0.0_f32; input_size + frame];
        let input_f32: Vec<f32> = samples.iter().map(|&s| s as f32).collect();

        let mut offset = 0;
        let mut pad = vec![0.0_f32; frame];
        while offset < input_size {
            let remaining = input_size - offset;
            let chunk_len = remaining.min(frame);
            let end = offset + chunk_len;
            let input_chunk = if chunk_len < frame {
                pad[..chunk_len].copy_from_slice(&input_f32[offset..end]);
                pad[chunk_len..].fill(0.0);
                &pad[..]
            } else {
                &input_f32[offset..end]
            };
            self.denoise
                .process_frame(&mut output_buf[offset..offset + frame], input_chunk);
            offset += chunk_len;
        }

        output_buf[..input_size]
            .iter()
            .map(|&s| s.clamp(i16::MIN as f32, i16::MAX as f32) as i16)
            .collect()
    }

    pub fn reset(&mut self) {
        self.up.reset();
        self.down.reset();
        self.denoise = DenoiseState::new();
    }
}
