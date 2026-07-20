use crate::{Decoder, Encoder, PcmBuf, Sample};
pub use opus_rs::Application as OpusApplication;
use opus_rs::{Application, OpusDecoder as OpusDecoderRaw, OpusEncoder as OpusEncoderRaw};

pub struct OpusDecoder {
    decoder: OpusDecoderRaw,
    sample_rate: u32,
    channels: u16,
    w_output_f32: Vec<f32>,
    w_pcm_i16: Vec<i16>,
}

impl OpusDecoder {
    /// Create a new Opus decoder instance
    pub fn new(sample_rate: u32, channels: u16) -> Self {
        let decoder = OpusDecoderRaw::new(sample_rate as i32, channels as usize)
            .expect("Failed to create Opus decoder");

        Self {
            decoder,
            sample_rate,
            channels,
            w_output_f32: Vec::new(),
            w_pcm_i16: Vec::new(),
        }
    }

    /// Create a default Opus decoder (48kHz, stereo)
    pub fn new_default() -> Self {
        Self::new(48000, 2)
    }

    pub fn decode_into(&mut self, data: &[u8], output: &mut [i16]) -> usize {
        if data.is_empty() {
            return 0;
        }

        // Detect the actual channel count from the Opus packet's TOC byte
        // (bit 2 is the stereo flag per RFC 6716). opus-rs 0.1.19+ rejects
        // decoding when the packet channel count doesn't match the decoder's,
        // so we adapt the decoder here to avoid returning empty PCM.
        let packet_channels = if data[0] & 0x04 != 0 { 2usize } else { 1 };
        if self.channels as usize != packet_channels {
            self.channels = packet_channels as u16;
            self.decoder = OpusDecoderRaw::new(self.sample_rate as i32, packet_channels)
                .expect("Failed to create Opus decoder");
        }

        let channels = usize::from(self.channels);
        let frame_size = (self.sample_rate as usize * 20) / 1000;
        let max_samples = frame_size * channels;
        if self.w_output_f32.len() < max_samples {
            self.w_output_f32.resize(max_samples, 0.0);
        }

        match self
            .decoder
            .decode(data, frame_size, &mut self.w_output_f32[..max_samples])
        {
            Ok(len) => {
                let total_samples = len * channels;
                if total_samples == 0 {
                    return 0;
                }

                if self.w_pcm_i16.len() < total_samples {
                    self.w_pcm_i16.resize(total_samples, 0);
                }

                for i in 0..total_samples {
                    self.w_pcm_i16[i] =
                        (self.w_output_f32[i] * 32768.0).clamp(-32768.0, 32767.0) as i16;
                }

                let n = total_samples.min(output.len());
                output[..n].copy_from_slice(&self.w_pcm_i16[..n]);
                n
            }
            Err(_) => 0,
        }
    }
}

impl Decoder for OpusDecoder {
    fn decode(&mut self, data: &[u8]) -> PcmBuf {
        if data.is_empty() {
            return Vec::new();
        }
        let packet_channels = if data[0] & 0x04 != 0 { 2usize } else { 1 };

        let frame_size = (self.sample_rate as usize * 20) / 1000;
        let max_samples = frame_size * packet_channels;
        let mut pcm = vec![0i16; max_samples];
        let n = self.decode_into(data, &mut pcm);
        pcm.truncate(n);
        if usize::from(self.channels) == 2 {
            pcm = pcm
                .chunks_exact(2)
                .map(|chunk| ((chunk[0] as i32 + chunk[1] as i32) / 2) as i16)
                .collect();
        }
        pcm
    }

    fn sample_rate(&self) -> u32 {
        self.sample_rate
    }

    fn channels(&self) -> u16 {
        self.channels
    }
}

pub struct OpusEncoder {
    encoder: OpusEncoderRaw,
    sample_rate: u32,
    channels: u16,
    w_input_f32: Vec<f32>,
    w_packet: Vec<u8>,
    /// Reusable buffer for mono→stereo upmix.  Avoids a per-frame allocation
    /// in `encode()` when `channels == 2`.
    w_stereo: Vec<i16>,
}

impl OpusEncoder {
    pub fn new_with_application(sample_rate: u32, channels: u16, application: Application) -> Self {
        let encoder = OpusEncoderRaw::new(sample_rate as i32, channels as usize, application)
            .expect("Failed to create Opus encoder");

        Self {
            encoder,
            sample_rate,
            channels,
            w_input_f32: Vec::new(),
            w_packet: vec![0u8; 1275],
            w_stereo: Vec::new(),
        }
    }

    /// Create a new Opus encoder instance.
    ///
    /// Keep backward-compatible defaults with pre-0.3.31 behavior:
    /// - VoIP application
    /// - caller can provide mono PCM even when encoder is configured as stereo;
    ///   `encode()` duplicates mono samples to stereo.
    pub fn new(sample_rate: u32, channels: u16) -> Self {
        let mut enc = Self::new_with_application(sample_rate, channels, Application::Voip);
        enc.encoder.bitrate_bps = if channels == 2 { 64000 } else { 48000 };
        enc.encoder.complexity = 5;
        enc.encoder.use_cbr = true;
        enc
    }

    /// Create a default Opus encoder (48kHz, stereo)
    pub fn new_default() -> Self {
        Self::new(48000, 2)
    }

    /// Set the encoder bitrate in bits per second.
    pub fn set_bitrate(&mut self, bitrate_bps: i32) {
        self.encoder.bitrate_bps = bitrate_bps;
    }

    /// Set the encoder complexity (0-10).
    pub fn set_complexity(&mut self, complexity: i32) {
        self.encoder.complexity = complexity;
    }

    /// Enable or disable constant bitrate (CBR) mode.
    pub fn set_cbr(&mut self, cbr: bool) {
        self.encoder.use_cbr = cbr;
    }

    /// Encode into a caller-provided packet buffer.
    ///
    /// Returns `Some(bytes_written)` on success.
    pub fn encode_into(&mut self, samples: &[Sample], output: &mut [u8]) -> Option<usize> {
        let channels = usize::from(self.channels);
        if samples.is_empty() || channels == 0 || samples.len() % channels != 0 {
            return None;
        }

        let frame_size = samples.len() / channels;

        if self.w_input_f32.len() < samples.len() {
            self.w_input_f32.resize(samples.len(), 0.0);
        }

        for (dst, &s) in self.w_input_f32[..samples.len()]
            .iter_mut()
            .zip(samples.iter())
        {
            *dst = s as f32 / 32768.0;
        }

        self.encoder
            .encode(&self.w_input_f32[..samples.len()], frame_size, output)
            .ok()
    }

    fn encode_raw(&mut self, samples: &[Sample]) -> Vec<u8> {
        let channels = usize::from(self.channels);
        if samples.is_empty() || channels == 0 || samples.len() % channels != 0 {
            return Vec::new();
        }

        let frame_size = samples.len() / channels;

        if self.w_input_f32.len() < samples.len() {
            self.w_input_f32.resize(samples.len(), 0.0);
        }

        for (dst, &s) in self.w_input_f32[..samples.len()]
            .iter_mut()
            .zip(samples.iter())
        {
            *dst = s as f32 / 32768.0;
        }

        match self.encoder.encode(
            &self.w_input_f32[..samples.len()],
            frame_size,
            &mut self.w_packet,
        ) {
            Ok(len) => {
                let mut out = Vec::with_capacity(len);
                out.extend_from_slice(&self.w_packet[..len]);
                out
            }
            Err(_) => Vec::new(),
        }
    }
}

impl Encoder for OpusEncoder {
    fn encode(&mut self, samples: &[Sample]) -> Vec<u8> {
        if self.channels == 2 {
            let mut stereo = std::mem::take(&mut self.w_stereo);
            stereo.resize(samples.len() * 2, 0);
            for (i, &sample) in samples.iter().enumerate() {
                stereo[2 * i] = sample;
                stereo[2 * i + 1] = sample;
            }
            let out = self.encode_raw(&stereo);
            self.w_stereo = stereo;
            return out;
        }
        self.encode_raw(samples)
    }

    fn sample_rate(&self) -> u32 {
        self.sample_rate
    }

    fn channels(&self) -> u16 {
        self.channels
    }
}

#[cfg(test)]
mod tests {
    use crate::*;

    fn make_mono_pcm_20ms() -> Vec<i16> {
        (0..960).map(|i| ((i * 100) % 32767) as i16).collect()
    }

    #[test]
    fn test_opus_encode_decode_20ms_produces_960_mono_samples() {
        let mut enc = OpusEncoder::new_default();
        let pcm = make_mono_pcm_20ms();
        let opus_pkt = enc.encode(&pcm);
        assert!(!opus_pkt.is_empty(), "encoder should produce output");

        // TOC byte bit 2 should be set (stereo flag from mono→stereo upmix)
        assert!(opus_pkt[0] & 0x04 != 0, "packet should be stereo");

        let mut dec = OpusDecoder::new_default();
        let decoded = dec.decode(&opus_pkt);

        // 20ms at 48kHz mono = 960 samples
        assert_eq!(
            decoded.len(),
            960,
            "decoder should downmix stereo→mono and output 960 samples, got {}",
            decoded.len()
        );
    }

    #[test]
    fn test_opus_consecutive_frames_all_produce_960() {
        let mut enc = OpusEncoder::new_default();
        let pcm = make_mono_pcm_20ms();
        let opus_pkt = enc.encode(&pcm);

        let mut dec = OpusDecoder::new_default();
        for i in 0..5 {
            let decoded = dec.decode(&opus_pkt);
            assert_eq!(
                decoded.len(),
                960,
                "frame {} should produce 960 mono samples, got {}",
                i,
                decoded.len()
            );
        }
    }

    #[test]
    fn test_opus_decoder_output_has_reasonable_energy() {
        let mut enc = OpusEncoder::new_default();
        // 440Hz sine at 48kHz, 20ms
        let pcm: Vec<i16> = (0..960)
            .map(|i| {
                let t = i as f64 / 48000.0;
                (16384.0 * (2.0 * std::f64::consts::PI * 440.0 * t).sin()) as i16
            })
            .collect();
        let opus_pkt = enc.encode(&pcm);

        let mut dec = OpusDecoder::new_default();
        let decoded = dec.decode(&opus_pkt);
        assert_eq!(decoded.len(), 960);

        let energy: f64 =
            decoded.iter().map(|&s| (s as f64).powi(2)).sum::<f64>() / decoded.len() as f64;
        let rms = energy.sqrt();
        assert!(
            rms > 100.0 && rms < 20000.0,
            "decoded audio RMS {} should be in reasonable range",
            rms
        );
    }

    #[test]
    fn test_opus_decoder_handles_mono_packet_gracefully() {
        // Create a mono Opus encoder, encode PCM → mono Opus packet
        let mut mono_enc = OpusEncoder::new(48000, 1);
        let pcm = make_mono_pcm_20ms();
        let mono_pkt = mono_enc.encode(&pcm);
        assert!(!mono_pkt.is_empty());
        // Mono packet TOC bit 2 should be 0
        assert!(
            mono_pkt[0] & 0x04 == 0,
            "mono encoder should produce mono packet"
        );

        // Decode with default stereo decoder — should handle mono→stereo transition
        let mut dec = OpusDecoder::new_default();
        let decoded = dec.decode(&mono_pkt);
        assert_eq!(
            decoded.len(),
            960,
            "first mono packet should produce exactly 960 mono samples, got {}",
            decoded.len()
        );

        // Second mono packet should also be 960
        let decoded2 = dec.decode(&mono_pkt);
        assert_eq!(
            decoded2.len(),
            960,
            "second mono packet should also produce 960 mono samples, got {}",
            decoded2.len()
        );
    }

    #[test]
    fn test_opus_stereo_packet_downmix_produces_960() {
        let mut enc = OpusEncoder::new_default();
        let pcm = make_mono_pcm_20ms();
        let stereo_pkt = enc.encode(&pcm);
        assert!(stereo_pkt[0] & 0x04 != 0);

        let mut dec = OpusDecoder::new_default();
        let decoded = dec.decode(&stereo_pkt);
        assert_eq!(
            decoded.len(),
            960,
            "stereo packet should downmix to 960 mono samples"
        );
    }
}
