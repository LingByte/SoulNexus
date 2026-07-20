//! Minimal types for TinySilero / TinyTen VAD (port from voice-engine).

pub mod capi;
pub(crate) mod simd;
pub mod tiny_silero;
pub mod tiny_ten;
pub(crate) mod utils;

pub use tiny_silero::TinySilero;
pub use tiny_ten::TinyTen;

use std::any::Any;

pub type Sample = i16;
pub type PcmBuf = Vec<Sample>;

pub enum Samples {
    PCM { samples: PcmBuf },
    Empty,
}

pub struct AudioFrame {
    pub samples: Samples,
    pub sample_rate: u32,
    pub timestamp: u64,
}

#[derive(Clone, Debug)]
pub struct VADOption {
    pub samplerate: u32,
    pub voice_threshold: f32,
}

impl Default for VADOption {
    fn default() -> Self {
        Self {
            samplerate: 16000,
            voice_threshold: 0.5,
        }
    }
}

pub trait VadEngine: Send + Sync + Any {
    fn process(&mut self, frame: &mut AudioFrame) -> Option<(bool, u64)>;
}
