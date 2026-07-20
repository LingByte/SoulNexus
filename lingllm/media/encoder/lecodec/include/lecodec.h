/* LingEchoX encoder native C ABI (-tags lingcodec). */
#ifndef LECODEC_H
#define LECODEC_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define LE_CODEC_PCMU             0
#define LE_CODEC_PCMA             8
#define LE_CODEC_G722             9
#define LE_CODEC_G729             18
#define LE_CODEC_TELEPHONE_EVENT  101
#define LE_CODEC_OPUS             111

#define LE_OPUS_VOIP      0
#define LE_OPUS_AUDIO     1
#define LE_OPUS_LOWDELAY  2

#define LE_OK          0
#define LE_ERR_NULL   -1
#define LE_ERR_CODEC  -2
#define LE_ERR_PANIC  -5
#define LE_ERR_BUF    -6

typedef struct LeEncoder LeEncoder;
typedef struct LeDecoder LeDecoder;

const char *le_version(void);
void le_free(uint8_t *ptr, size_t len);
void le_free_samples(int16_t *ptr, size_t len);

LeEncoder *le_encoder_open(uint8_t codec);
LeEncoder *le_opus_encoder_open(uint32_t sample_rate, uint16_t channels, int application);
void le_encoder_close(LeEncoder *enc);
int le_encoder_encode(LeEncoder *enc, const int16_t *samples, size_t n_samples,
                      uint8_t **out, size_t *out_len);
int le_encoder_encode_into(LeEncoder *enc, const int16_t *samples, size_t n_samples,
                           uint8_t *out, size_t out_cap);
uint32_t le_encoder_sample_rate(const LeEncoder *enc);
uint16_t le_encoder_channels(const LeEncoder *enc);

LeDecoder *le_decoder_open(uint8_t codec);
LeDecoder *le_opus_decoder_open(uint32_t sample_rate, uint16_t channels);
void le_decoder_close(LeDecoder *dec);
int le_decoder_decode(LeDecoder *dec, const uint8_t *data, size_t data_len,
                      int16_t **out, size_t *out_len);
int le_decoder_decode_into(LeDecoder *dec, const uint8_t *data, size_t data_len,
                           int16_t *out, size_t out_cap);
uint32_t le_decoder_sample_rate(const LeDecoder *dec);
uint16_t le_decoder_channels(const LeDecoder *dec);

#ifdef __cplusplus
}
#endif

#endif /* LECODEC_H */
