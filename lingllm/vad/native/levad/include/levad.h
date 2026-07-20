/* LingEchoX levad native C ABI (-tags levad). */
#ifndef LEVAD_H
#define LEVAD_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define LV_KIND_SILERO  0
#define LV_KIND_TEN     1

#define LV_OK         0
#define LV_NEED_MORE  1
#define LV_ERR_NULL  -1
#define LV_ERR_KIND  -2
#define LV_ERR_INIT  -3
#define LV_ERR_PANIC -5

typedef struct LvVad LvVad;

const char *lv_version(void);

/* threshold in (0,1]; use 0 for default 0.5. sample_rate 0 => 16000. */
LvVad *lv_vad_open(uint8_t kind, uint32_t sample_rate, float threshold);
void lv_vad_close(LvVad *vad);

/* PCM16LE mono. LV_OK => *out_speech is 0/1; LV_NEED_MORE => not enough samples yet. */
int lv_vad_process(LvVad *vad, const int16_t *pcm, size_t n_samples, int *out_speech);

uint32_t lv_vad_sample_rate(const LvVad *vad);
float lv_vad_threshold(const LvVad *vad);

#ifdef __cplusplus
}
#endif

#endif /* LEVAD_H */
