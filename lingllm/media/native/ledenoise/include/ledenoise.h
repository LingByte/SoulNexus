/* LingEchoX ledenoise native C ABI (-tags ledenoise). */
#ifndef LEDENOISE_H
#define LEDENOISE_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define LD_OK        0
#define LD_ERR_NULL -1
#define LD_ERR_INIT -3
#define LD_ERR_PANIC -5
#define LD_ERR_BUF  -6

typedef struct LdDenoise LdDenoise;

const char *ld_version(void);

LdDenoise *ld_denoise_open(uint32_t sample_rate);
void ld_denoise_close(LdDenoise *h);

/* Returns output sample count (>=0) or negative error. */
int ld_denoise_process(LdDenoise *h, const int16_t *pcm, size_t n_samples,
                       int16_t *out, size_t out_cap);

uint32_t ld_denoise_sample_rate(const LdDenoise *h);
void ld_denoise_reset(LdDenoise *h);

/* Time-domain SNR estimator (no WebRTC APM / Unix-socket process). */
typedef struct LdSnr LdSnr;

LdSnr *ld_snr_open(uint32_t sample_rate);
void ld_snr_close(LdSnr *h);
/* Returns 1 when SNR ready, 0 warming up, or negative error. */
int ld_snr_process(LdSnr *h, const int16_t *pcm, size_t n_samples, float *out_snr_db);
void ld_snr_reset(LdSnr *h);

#ifdef __cplusplus
}
#endif

#endif /* LEDENOISE_H */
