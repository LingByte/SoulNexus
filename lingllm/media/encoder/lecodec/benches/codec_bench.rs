use std::hint::black_box;

use criterion::{Criterion, criterion_group, criterion_main};
use lecodec::{CodecKind, create_decoder, create_encoder};

fn bench_codec(c: &mut Criterion) {
    let codecs = [
        CodecKind::Pcmu,
        CodecKind::Pcma,
        CodecKind::G722,
        CodecKind::G729,
        #[cfg(feature = "opus")]
        CodecKind::Opus,
        CodecKind::TelephoneEvent,
    ];

    for codec in codecs {
        let name = format!("{:?}", codec);
        let mut group = c.benchmark_group(&name);
        let mut encoder = create_encoder(codec);
        let mut decoder = create_decoder(codec);
        let sample_rate = encoder.sample_rate();
        let channels = encoder.channels();
        let samples_count = (sample_rate as f64 * 0.02) as usize * channels as usize;
        let pcm_samples = vec![0i16; samples_count];
        let encoded_data = encoder.encode(&pcm_samples);

        group.bench_function("encode_20ms", |b| {
            b.iter(|| encoder.encode(black_box(&pcm_samples)))
        });
        group.bench_function("decode_20ms", |b| {
            b.iter(|| decoder.decode(black_box(&encoded_data)))
        });
        group.finish();
    }
}

criterion_group!(benches, bench_codec);
criterion_main!(benches);
