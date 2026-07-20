//! C ABI for Go (`lingllm/vad/local`, `-tags levad`).

use crate::{AudioFrame, Samples, TinySilero, TinyTen, VADOption, VadEngine};
use std::os::raw::{c_char, c_float, c_int};
use std::panic::{catch_unwind, AssertUnwindSafe};
use std::slice;

pub const LV_KIND_SILERO: u8 = 0;
pub const LV_KIND_TEN: u8 = 1;

pub const LV_OK: c_int = 0;
pub const LV_NEED_MORE: c_int = 1;
pub const LV_ERR_NULL: c_int = -1;
pub const LV_ERR_KIND: c_int = -2;
pub const LV_ERR_INIT: c_int = -3;
pub const LV_ERR_PANIC: c_int = -5;

enum Engine {
    Silero(TinySilero),
    Ten(TinyTen),
}

pub struct LvVad {
    engine: Engine,
    sample_rate: u32,
    threshold: f32,
}

fn open_inner(kind: u8, sample_rate: u32, threshold: f32) -> Result<LvVad, anyhow::Error> {
    let sr = if sample_rate == 0 { 16000 } else { sample_rate };
    let thr = if threshold <= 0.0 || threshold > 1.0 {
        0.5
    } else {
        threshold
    };
    let config = VADOption {
        samplerate: sr,
        voice_threshold: thr,
    };
    let engine = match kind {
        LV_KIND_SILERO => Engine::Silero(TinySilero::new(config)?),
        LV_KIND_TEN => Engine::Ten(TinyTen::new(config)?),
        _ => anyhow::bail!("unknown vad kind"),
    };
    Ok(LvVad {
        engine,
        sample_rate: sr,
        threshold: thr,
    })
}

#[unsafe(no_mangle)]
pub extern "C" fn lv_vad_open(kind: u8, sample_rate: u32, threshold: c_float) -> *mut LvVad {
    match catch_unwind(AssertUnwindSafe(|| open_inner(kind, sample_rate, threshold))) {
        Ok(Ok(v)) => Box::into_raw(Box::new(v)),
        Ok(Err(_)) | Err(_) => std::ptr::null_mut(),
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn lv_vad_close(vad: *mut LvVad) {
    if !vad.is_null() {
        unsafe {
            drop(Box::from_raw(vad));
        }
    }
}

/// Feed PCM16LE mono samples. On success returns LV_OK and sets *out_speech when at least
/// one chunk was classified; returns LV_NEED_MORE when more samples are required.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn lv_vad_process(
    vad: *mut LvVad,
    pcm: *const i16,
    n_samples: usize,
    out_speech: *mut c_int,
) -> c_int {
    if vad.is_null() || out_speech.is_null() {
        return LV_ERR_NULL;
    }
    if pcm.is_null() && n_samples != 0 {
        return LV_ERR_NULL;
    }
    let result = catch_unwind(AssertUnwindSafe(|| unsafe {
        let samples: Vec<i16> = if n_samples == 0 {
            Vec::new()
        } else {
            slice::from_raw_parts(pcm, n_samples).to_vec()
        };
        let sr = (*vad).sample_rate;
        let mut frame = AudioFrame {
            samples: Samples::PCM { samples },
            sample_rate: sr,
            timestamp: 0,
        };
        let mut got = false;
        let mut any_speech = false;
        loop {
            let decision = match &mut (*vad).engine {
                Engine::Silero(e) => e.process(&mut frame),
                Engine::Ten(e) => e.process(&mut frame),
            };
            match decision {
                Some((speech, _)) => {
                    got = true;
                    any_speech |= speech;
                    frame.samples = Samples::PCM {
                        samples: Vec::new(),
                    };
                }
                None => break,
            }
        }
        (got, any_speech)
    }));
    match result {
        Ok((got, speech)) => {
            if got {
                unsafe {
                    *out_speech = if speech { 1 } else { 0 };
                }
                LV_OK
            } else {
                unsafe {
                    *out_speech = 0;
                }
                LV_NEED_MORE
            }
        }
        Err(_) => LV_ERR_PANIC,
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn lv_vad_sample_rate(vad: *const LvVad) -> u32 {
    if vad.is_null() {
        return 0;
    }
    unsafe { (*vad).sample_rate }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn lv_vad_threshold(vad: *const LvVad) -> c_float {
    if vad.is_null() {
        return 0.0;
    }
    unsafe { (*vad).threshold }
}

#[unsafe(no_mangle)]
pub extern "C" fn lv_version() -> *const c_char {
    concat!(env!("CARGO_PKG_VERSION"), "\0").as_ptr() as *const c_char
}
