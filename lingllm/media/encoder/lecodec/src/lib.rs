//! LingEchoX lecodec — VoIP codecs with a C ABI for Go (`package encoder`, `-tags lingcodec`).

pub type Sample = i16;
pub type PcmBuf = Vec<Sample>;

pub mod capi;
pub mod codec;
pub mod resample;

pub use codec::g722;
pub use codec::g729;
#[cfg(feature = "opus")]
pub use codec::opus;
pub use codec::pcma;
pub use codec::pcmu;
pub use codec::telephone_event;
pub use resample::{resample, Resampler};

#[derive(Debug, Clone, Copy, Eq, Ord, PartialEq, PartialOrd)]
pub enum CodecKind {
    Pcmu,
    Pcma,
    G722,
    G729,
    #[cfg(feature = "opus")]
    Opus,
    TelephoneEvent,
}

// Keep CodecType alias for benches / older call sites.
pub type CodecType = CodecKind;

pub trait Decoder: Send + Sync {
    fn decode(&mut self, data: &[u8]) -> PcmBuf;
    fn sample_rate(&self) -> u32;
    fn channels(&self) -> u16;
}

pub trait Encoder: Send + Sync {
    fn encode(&mut self, samples: &[Sample]) -> Vec<u8>;
    fn sample_rate(&self) -> u32;
    fn channels(&self) -> u16;
}

pub fn open_decoder(kind: CodecKind) -> Box<dyn Decoder> {
    match kind {
        CodecKind::Pcmu => Box::new(pcmu::PcmuDecoder::new()),
        CodecKind::Pcma => Box::new(pcma::PcmaDecoder::new()),
        CodecKind::G722 => Box::new(g722::G722Decoder::new()),
        CodecKind::G729 => Box::new(g729::G729Decoder::new()),
        #[cfg(feature = "opus")]
        CodecKind::Opus => Box::new(opus::OpusDecoder::new_default()),
        CodecKind::TelephoneEvent => Box::new(telephone_event::TelephoneEventDecoder::new()),
    }
}

pub fn open_encoder(kind: CodecKind) -> Box<dyn Encoder> {
    match kind {
        CodecKind::Pcmu => Box::new(pcmu::PcmuEncoder::new()),
        CodecKind::Pcma => Box::new(pcma::PcmaEncoder::new()),
        CodecKind::G722 => Box::new(g722::G722Encoder::new()),
        CodecKind::G729 => Box::new(g729::G729Encoder::new()),
        #[cfg(feature = "opus")]
        CodecKind::Opus => Box::new(opus::OpusEncoder::new_default()),
        CodecKind::TelephoneEvent => Box::new(telephone_event::TelephoneEventEncoder::new()),
    }
}

/// Legacy names used by older call sites / benches.
pub fn create_decoder(codec: CodecKind) -> Box<dyn Decoder> {
    open_decoder(codec)
}
pub fn create_encoder(codec: CodecKind) -> Box<dyn Encoder> {
    open_encoder(codec)
}

#[cfg(feature = "opus")]
pub fn open_opus_encoder(
    sample_rate: u32,
    channels: u16,
    application: opus::OpusApplication,
) -> Box<dyn Encoder> {
    Box::new(opus::OpusEncoder::new_with_application(
        sample_rate,
        channels,
        application,
    ))
}

#[cfg(feature = "opus")]
pub fn open_opus_decoder(sample_rate: u32, channels: u16) -> Box<dyn Decoder> {
    Box::new(opus::OpusDecoder::new(sample_rate, channels))
}

#[cfg(feature = "opus")]
pub fn create_opus_encoder(
    sample_rate: u32,
    channels: u16,
    application: opus::OpusApplication,
) -> Box<dyn Encoder> {
    open_opus_encoder(sample_rate, channels, application)
}

#[cfg(feature = "opus")]
pub fn create_opus_decoder(sample_rate: u32, channels: u16) -> Box<dyn Decoder> {
    open_opus_decoder(sample_rate, channels)
}

impl CodecKind {
    pub fn mime_type(self) -> &'static str {
        match self {
            CodecKind::Pcmu => "audio/PCMU",
            CodecKind::Pcma => "audio/PCMA",
            CodecKind::G722 => "audio/G722",
            CodecKind::G729 => "audio/G729",
            #[cfg(feature = "opus")]
            CodecKind::Opus => "audio/opus",
            CodecKind::TelephoneEvent => "audio/telephone-event",
        }
    }

    pub fn clock_rate(self) -> u32 {
        match self {
            #[cfg(feature = "opus")]
            CodecKind::Opus => 48000,
            _ => 8000,
        }
    }

    pub fn sample_rate(self) -> u32 {
        match self {
            CodecKind::G722 => 16000,
            #[cfg(feature = "opus")]
            CodecKind::Opus => 48000,
            _ => 8000,
        }
    }

    pub fn channels(self) -> u16 {
        match self {
            #[cfg(feature = "opus")]
            CodecKind::Opus => 2,
            _ => 1,
        }
    }

    pub fn payload_type(self) -> u8 {
        match self {
            CodecKind::Pcmu => 0,
            CodecKind::Pcma => 8,
            CodecKind::G722 => 9,
            CodecKind::G729 => 18,
            #[cfg(feature = "opus")]
            CodecKind::Opus => 111,
            CodecKind::TelephoneEvent => 101,
        }
    }
}

impl TryFrom<u8> for CodecKind {
    type Error = anyhow::Error;

    fn try_from(value: u8) -> Result<Self, Self::Error> {
        match value {
            0 => Ok(CodecKind::Pcmu),
            8 => Ok(CodecKind::Pcma),
            9 => Ok(CodecKind::G722),
            18 => Ok(CodecKind::G729),
            101 => Ok(CodecKind::TelephoneEvent),
            #[cfg(feature = "opus")]
            111 => Ok(CodecKind::Opus),
            _ => Err(anyhow::anyhow!("unknown codec payload type: {value}")),
        }
    }
}

#[cfg(target_endian = "little")]
pub fn samples_to_bytes(samples: &[Sample]) -> Vec<u8> {
    unsafe {
        std::slice::from_raw_parts(
            samples.as_ptr() as *const u8,
            samples.len() * std::mem::size_of::<Sample>(),
        )
        .to_vec()
    }
}

#[cfg(target_endian = "big")]
pub fn samples_to_bytes(samples: &[Sample]) -> Vec<u8> {
    samples.iter().flat_map(|s| s.to_le_bytes()).collect()
}

#[cfg(target_endian = "little")]
pub fn bytes_to_samples(u8_data: &[u8]) -> PcmBuf {
    unsafe {
        std::slice::from_raw_parts(
            u8_data.as_ptr() as *const Sample,
            u8_data.len() / std::mem::size_of::<Sample>(),
        )
        .to_vec()
    }
}

#[cfg(target_endian = "big")]
pub fn bytes_to_samples(u8_data: &[u8]) -> PcmBuf {
    u8_data
        .chunks(2)
        .map(|chunk| (chunk[0] as i16) | ((chunk[1] as i16) << 8))
        .collect()
}
