/*
 * SPDX-FileCopyrightText: 2026 LingByte. All rights reserved.
 * SPDX-License-Identifier: AGPL-3.0
 */

#ifndef DENOISE_H
#define DENOISE_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stdbool.h>

/**
 * @brief Denoise processor handle
 */
typedef void* denoise_handle_t;

/**
 * @brief Denoise configuration structure
 */
typedef struct {
    bool aec_enable;          /*!< Enable Acoustic Echo Cancellation (AEC) */
    bool agc_enable;          /*!< Enable Automatic Gain Control (AGC) */
    int sample_rate;          /*!< Sample rate in Hz (e.g., 16000) */
    int channels;             /*!< Number of audio channels (e.g., 1 for mono) */
    int bits_per_sample;      /*!< Bits per sample (e.g., 16) */
} denoise_config_t;

/**
 * @brief Create a denoise processor
 *
 * @param[in] config Pointer to denoise configuration
 * @return Handle to the denoise processor, or NULL on failure
 */
denoise_handle_t denoise_create(const denoise_config_t *config);

/**
 * @brief Process audio data with denoise
 *
 * @param[in] handle Denoise processor handle
 * @param[in] input Input audio data (PCM format)
 * @param[in] input_len Length of input data in bytes
 * @param[out] output Output audio data (denoised)
 * @return Number of bytes processed, or negative value on error
 */
int denoise_process(denoise_handle_t handle, const uint8_t *input, int input_len, uint8_t *output);

/**
 * @brief Process audio data in-place
 *
 * @param[in] handle Denoise processor handle
 * @param[in,out] data Audio data to process (will be modified in-place)
 * @param[in] data_len Length of data in bytes
 * @return Number of bytes processed, or negative value on error
 */
int denoise_process_inplace(denoise_handle_t handle, uint8_t *data, int data_len);

/**
 * @brief Reset denoise processor state
 *
 * @param[in] handle Denoise processor handle
 * @return 0 on success, negative value on error
 */
int denoise_reset(denoise_handle_t handle);

/**
 * @brief Set AEC enable status
 *
 * @param[in] handle Denoise processor handle
 * @param[in] enable Enable or disable AEC
 * @return 0 on success, negative value on error
 */
int denoise_set_aec_enable(denoise_handle_t handle, bool enable);

/**
 * @brief Set AGC enable status
 *
 * @param[in] handle Denoise processor handle
 * @param[in] enable Enable or disable AGC
 * @return 0 on success, negative value on error
 */
int denoise_set_agc_enable(denoise_handle_t handle, bool enable);

/**
 * @brief Destroy denoise processor
 *
 * @param[in] handle Denoise processor handle
 * @return 0 on success, negative value on error
 */
int denoise_destroy(denoise_handle_t handle);

/**
 * @brief Get denoise library version
 *
 * @return Version string
 */
const char* denoise_version(void);

#ifdef __cplusplus
}
#endif

#endif /* DENOISE_H */
