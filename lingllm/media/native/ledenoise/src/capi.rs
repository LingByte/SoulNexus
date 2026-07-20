//! C ABI for Go (`media/denoise`, `-tags ledenoise`).

use crate::{NoiseReducer, NoiseSnr};
use std::os::raw::{c_char, c_int};
use std::panic::{catch_unwind, AssertUnwindSafe};
use std::slice;

pub const LD_OK: c_int = 0;
pub const LD_ERR_NULL: c_int = -1;
pub const LD_ERR_INIT: c_int = -3;
pub const LD_ERR_PANIC: c_int = -5;
pub const LD_ERR_BUF: c_int = -6;

pub struct LdDenoise {
    inner: NoiseReducer,
}

#[unsafe(no_mangle)]
pub extern "C" fn ld_denoise_open(sample_rate: u32) -> *mut LdDenoise {
    match catch_unwind(AssertUnwindSafe(|| NoiseReducer::new(sample_rate as usize))) {
        Ok(Ok(inner)) => Box::into_raw(Box::new(LdDenoise { inner })),
        Ok(Err(_)) | Err(_) => std::ptr::null_mut(),
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn ld_denoise_close(h: *mut LdDenoise) {
    if !h.is_null() {
        unsafe {
            drop(Box::from_raw(h));
        }
    }
}

/// Process mono PCM16. Writes up to out_cap samples into out; returns sample count (>=0)
/// or a negative error code.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn ld_denoise_process(
    h: *mut LdDenoise,
    pcm: *const i16,
    n_samples: usize,
    out: *mut i16,
    out_cap: usize,
) -> c_int {
    if h.is_null() || out.is_null() {
        return LD_ERR_NULL;
    }
    if pcm.is_null() && n_samples != 0 {
        return LD_ERR_NULL;
    }
    let result = catch_unwind(AssertUnwindSafe(|| unsafe {
        let input = if n_samples == 0 {
            &[][..]
        } else {
            slice::from_raw_parts(pcm, n_samples)
        };
        (*h).inner.process(input)
    }));
    match result {
        Ok(cleaned) => {
            if cleaned.len() > out_cap {
                return LD_ERR_BUF;
            }
            unsafe {
                if !cleaned.is_empty() {
                    std::ptr::copy_nonoverlapping(cleaned.as_ptr(), out, cleaned.len());
                }
            }
            cleaned.len() as c_int
        }
        Err(_) => LD_ERR_PANIC,
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn ld_denoise_sample_rate(h: *const LdDenoise) -> u32 {
    if h.is_null() {
        return 0;
    }
    unsafe { (*h).inner.sample_rate() as u32 }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn ld_denoise_reset(h: *mut LdDenoise) {
    if h.is_null() {
        return;
    }
    let _ = catch_unwind(AssertUnwindSafe(|| unsafe {
        (*h).inner.reset();
    }));
}

#[unsafe(no_mangle)]
pub extern "C" fn ld_version() -> *const c_char {
    concat!(env!("CARGO_PKG_VERSION"), "\0").as_ptr() as *const c_char
}

// --- SNR estimator (time-domain noise floor; no WebRTC APM dependency) ------

pub struct LdSnr {
    inner: NoiseSnr,
}

#[unsafe(no_mangle)]
pub extern "C" fn ld_snr_open(sample_rate: u32) -> *mut LdSnr {
    match catch_unwind(AssertUnwindSafe(|| NoiseSnr::new(sample_rate as usize))) {
        Ok(inner) => Box::into_raw(Box::new(LdSnr { inner })),
        Err(_) => std::ptr::null_mut(),
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn ld_snr_close(h: *mut LdSnr) {
    if !h.is_null() {
        unsafe {
            drop(Box::from_raw(h));
        }
    }
}

/// Process mono PCM16. Writes smoothed SNR dB into *out_snr_db when non-null.
/// Returns 1 when ready, 0 when warming up, or a negative error code.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn ld_snr_process(
    h: *mut LdSnr,
    pcm: *const i16,
    n_samples: usize,
    out_snr_db: *mut f32,
) -> c_int {
    if h.is_null() {
        return LD_ERR_NULL;
    }
    if pcm.is_null() && n_samples != 0 {
        return LD_ERR_NULL;
    }
    let result = catch_unwind(AssertUnwindSafe(|| unsafe {
        let input = if n_samples == 0 {
            &[][..]
        } else {
            slice::from_raw_parts(pcm, n_samples)
        };
        let snr = (*h).inner.process(input);
        let ready = (*h).inner.ready();
        (snr, ready)
    }));
    match result {
        Ok((snr, ready)) => {
            if !out_snr_db.is_null() && snr.is_finite() {
                unsafe {
                    *out_snr_db = snr as f32;
                }
            }
            if ready {
                1
            } else {
                0
            }
        }
        Err(_) => LD_ERR_PANIC,
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn ld_snr_reset(h: *mut LdSnr) {
    if h.is_null() {
        return;
    }
    let _ = catch_unwind(AssertUnwindSafe(|| unsafe {
        (*h).inner.reset();
    }));
}
