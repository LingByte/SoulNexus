//! LingEchoX ledenoise — RNNoise uplink denoise (voice-engine NoiseReducer) with C ABI.
//! Also exposes a lightweight time-domain SNR estimator (`ld_snr_*`).

pub type Sample = i16;
pub type PcmBuf = Vec<Sample>;

pub mod capi;
pub mod denoiser;
pub mod resample;
pub mod snr;

pub use denoiser::NoiseReducer;
pub use resample::{resample, Resampler};
pub use snr::NoiseSnr;
