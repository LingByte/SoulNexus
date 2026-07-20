#![no_main]

use lecodec::{CodecKind, create_decoder, create_encoder};
use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    if data.len() < 2 {
        return;
    }
    let kind = match data[0] % 6 {
        0 => CodecKind::Pcmu,
        1 => CodecKind::Pcma,
        2 => CodecKind::G722,
        3 => CodecKind::G729,
        #[cfg(feature = "opus")]
        4 => CodecKind::Opus,
        _ => CodecKind::TelephoneEvent,
    };
    let payload = &data[1..];
    let mut decoder = create_decoder(kind);
    let mut encoder = create_encoder(kind);
    let _ = decoder.decode(payload);
    let n = ((payload.len() / 2).min(960)).max(0);
    let samples: Vec<i16> = payload
        .chunks_exact(2)
        .take(n)
        .map(|c| i16::from_le_bytes([c[0], c[1]]))
        .collect();
    let encoded = encoder.encode(&samples);
    let _ = decoder.decode(&encoded);
});
