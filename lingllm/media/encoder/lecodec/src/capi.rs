//! C ABI for Go (`encoder` package, `-tags lingcodec`).
//! Prefer `le_*_into` variants when the caller owns a scratch buffer.

use crate::{create_decoder, create_encoder, CodecKind, Decoder, Encoder};
use std::os::raw::{c_char, c_int};
use std::panic::{catch_unwind, AssertUnwindSafe};
use std::slice;

#[cfg(feature = "opus")]
use crate::{create_opus_decoder, create_opus_encoder, opus::OpusApplication};

pub const LE_CODEC_PCMU: u8 = 0;
pub const LE_CODEC_PCMA: u8 = 8;
pub const LE_CODEC_G722: u8 = 9;
pub const LE_CODEC_G729: u8 = 18;
pub const LE_CODEC_TELEPHONE_EVENT: u8 = 101;
#[cfg(feature = "opus")]
pub const LE_CODEC_OPUS: u8 = 111;

#[cfg(feature = "opus")]
pub const LE_OPUS_VOIP: c_int = 0;
#[cfg(feature = "opus")]
pub const LE_OPUS_AUDIO: c_int = 1;
#[cfg(feature = "opus")]
pub const LE_OPUS_LOWDELAY: c_int = 2;

pub const LE_OK: c_int = 0;
pub const LE_ERR_NULL: c_int = -1;
pub const LE_ERR_CODEC: c_int = -2;
pub const LE_ERR_PANIC: c_int = -5;
pub const LE_ERR_BUF: c_int = -6;

pub struct LeEncoder {
    inner: Box<dyn Encoder>,
}

pub struct LeDecoder {
    inner: Box<dyn Decoder>,
}

fn kind_from_pt(v: u8) -> Option<CodecKind> {
    CodecKind::try_from(v).ok()
}

unsafe fn give_bytes(buf: Vec<u8>, out: *mut *mut u8, out_len: *mut usize) {
    let mut buf = buf;
    buf.shrink_to_fit();
    let len = buf.len();
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    unsafe {
        *out = ptr;
        *out_len = len;
    }
}

unsafe fn give_samples(buf: Vec<i16>, out: *mut *mut i16, out_len: *mut usize) {
    let mut buf = buf;
    buf.shrink_to_fit();
    let len = buf.len();
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    unsafe {
        *out = ptr;
        *out_len = len;
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_free(ptr: *mut u8, len: usize) {
    if ptr.is_null() || len == 0 {
        return;
    }
    unsafe {
        let _ = Vec::from_raw_parts(ptr, len, len);
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_free_samples(ptr: *mut i16, len: usize) {
    if ptr.is_null() || len == 0 {
        return;
    }
    unsafe {
        let _ = Vec::from_raw_parts(ptr, len, len);
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn le_encoder_open(codec: u8) -> *mut LeEncoder {
    let Some(kind) = kind_from_pt(codec) else {
        return std::ptr::null_mut();
    };
    match catch_unwind(AssertUnwindSafe(|| create_encoder(kind))) {
        Ok(inner) => Box::into_raw(Box::new(LeEncoder { inner })),
        Err(_) => std::ptr::null_mut(),
    }
}

#[cfg(feature = "opus")]
#[unsafe(no_mangle)]
pub extern "C" fn le_opus_encoder_open(
    sample_rate: u32,
    channels: u16,
    application: c_int,
) -> *mut LeEncoder {
    let app = match application {
        LE_OPUS_VOIP => OpusApplication::Voip,
        LE_OPUS_AUDIO => OpusApplication::Audio,
        LE_OPUS_LOWDELAY => OpusApplication::RestrictedLowDelay,
        _ => return std::ptr::null_mut(),
    };
    match catch_unwind(AssertUnwindSafe(|| {
        create_opus_encoder(sample_rate, channels, app)
    })) {
        Ok(inner) => Box::into_raw(Box::new(LeEncoder { inner })),
        Err(_) => std::ptr::null_mut(),
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_encoder_close(enc: *mut LeEncoder) {
    if !enc.is_null() {
        unsafe {
            drop(Box::from_raw(enc));
        }
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_encoder_encode(
    enc: *mut LeEncoder,
    samples: *const i16,
    n_samples: usize,
    out: *mut *mut u8,
    out_len: *mut usize,
) -> c_int {
    if enc.is_null() || out.is_null() || out_len.is_null() {
        return LE_ERR_NULL;
    }
    if samples.is_null() && n_samples != 0 {
        return LE_ERR_NULL;
    }
    unsafe {
        *out = std::ptr::null_mut();
        *out_len = 0;
    }
    let result = catch_unwind(AssertUnwindSafe(|| unsafe {
        let pcm = if n_samples == 0 {
            &[][..]
        } else {
            slice::from_raw_parts(samples, n_samples)
        };
        (*enc).inner.encode(pcm)
    }));
    match result {
        Ok(bytes) => unsafe {
            give_bytes(bytes, out, out_len);
            LE_OK
        },
        Err(_) => LE_ERR_PANIC,
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_encoder_encode_into(
    enc: *mut LeEncoder,
    samples: *const i16,
    n_samples: usize,
    out: *mut u8,
    out_cap: usize,
) -> c_int {
    if enc.is_null() || out.is_null() {
        return LE_ERR_NULL;
    }
    if samples.is_null() && n_samples != 0 {
        return LE_ERR_NULL;
    }
    let result = catch_unwind(AssertUnwindSafe(|| unsafe {
        let pcm = if n_samples == 0 {
            &[][..]
        } else {
            slice::from_raw_parts(samples, n_samples)
        };
        (*enc).inner.encode(pcm)
    }));
    match result {
        Ok(bytes) => {
            if bytes.len() > out_cap {
                return LE_ERR_BUF;
            }
            unsafe {
                std::ptr::copy_nonoverlapping(bytes.as_ptr(), out, bytes.len());
            }
            bytes.len() as c_int
        }
        Err(_) => LE_ERR_PANIC,
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_encoder_sample_rate(enc: *const LeEncoder) -> u32 {
    if enc.is_null() {
        return 0;
    }
    unsafe { (*enc).inner.sample_rate() }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_encoder_channels(enc: *const LeEncoder) -> u16 {
    if enc.is_null() {
        return 0;
    }
    unsafe { (*enc).inner.channels() }
}

#[unsafe(no_mangle)]
pub extern "C" fn le_decoder_open(codec: u8) -> *mut LeDecoder {
    let Some(kind) = kind_from_pt(codec) else {
        return std::ptr::null_mut();
    };
    match catch_unwind(AssertUnwindSafe(|| create_decoder(kind))) {
        Ok(inner) => Box::into_raw(Box::new(LeDecoder { inner })),
        Err(_) => std::ptr::null_mut(),
    }
}

#[cfg(feature = "opus")]
#[unsafe(no_mangle)]
pub extern "C" fn le_opus_decoder_open(sample_rate: u32, channels: u16) -> *mut LeDecoder {
    match catch_unwind(AssertUnwindSafe(|| create_opus_decoder(sample_rate, channels))) {
        Ok(inner) => Box::into_raw(Box::new(LeDecoder { inner })),
        Err(_) => std::ptr::null_mut(),
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_decoder_close(dec: *mut LeDecoder) {
    if !dec.is_null() {
        unsafe {
            drop(Box::from_raw(dec));
        }
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_decoder_decode(
    dec: *mut LeDecoder,
    data: *const u8,
    data_len: usize,
    out: *mut *mut i16,
    out_len: *mut usize,
) -> c_int {
    if dec.is_null() || out.is_null() || out_len.is_null() {
        return LE_ERR_NULL;
    }
    if data.is_null() && data_len != 0 {
        return LE_ERR_NULL;
    }
    unsafe {
        *out = std::ptr::null_mut();
        *out_len = 0;
    }
    let result = catch_unwind(AssertUnwindSafe(|| unsafe {
        let bytes = if data_len == 0 {
            &[][..]
        } else {
            slice::from_raw_parts(data, data_len)
        };
        (*dec).inner.decode(bytes)
    }));
    match result {
        Ok(pcm) => unsafe {
            give_samples(pcm, out, out_len);
            LE_OK
        },
        Err(_) => LE_ERR_PANIC,
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_decoder_decode_into(
    dec: *mut LeDecoder,
    data: *const u8,
    data_len: usize,
    out: *mut i16,
    out_cap: usize,
) -> c_int {
    if dec.is_null() || out.is_null() {
        return LE_ERR_NULL;
    }
    if data.is_null() && data_len != 0 {
        return LE_ERR_NULL;
    }
    let result = catch_unwind(AssertUnwindSafe(|| unsafe {
        let bytes = if data_len == 0 {
            &[][..]
        } else {
            slice::from_raw_parts(data, data_len)
        };
        (*dec).inner.decode(bytes)
    }));
    match result {
        Ok(pcm) => {
            if pcm.len() > out_cap {
                return LE_ERR_BUF;
            }
            unsafe {
                std::ptr::copy_nonoverlapping(pcm.as_ptr(), out, pcm.len());
            }
            pcm.len() as c_int
        }
        Err(_) => LE_ERR_PANIC,
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_decoder_sample_rate(dec: *const LeDecoder) -> u32 {
    if dec.is_null() {
        return 0;
    }
    unsafe { (*dec).inner.sample_rate() }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn le_decoder_channels(dec: *const LeDecoder) -> u16 {
    if dec.is_null() {
        return 0;
    }
    unsafe { (*dec).inner.channels() }
}

#[unsafe(no_mangle)]
pub extern "C" fn le_version() -> *const c_char {
    concat!(env!("CARGO_PKG_VERSION"), "\0").as_ptr() as *const c_char
}
