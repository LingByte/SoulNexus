/*
 * SPDX-FileCopyrightText: 2026 LingByte. All rights reserved.
 * SPDX-License-Identifier: AGPL-3.0
 */

#include "denoise.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

#define DENOISE_VERSION "1.0.0"

/**
 * @brief Internal denoise processor structure
 */
typedef struct {
    denoise_config_t config;
    int16_t *aec_buffer;
    int16_t *agc_buffer;
    int buffer_size;
    bool initialized;
} denoise_processor_t;

/**
 * @brief Create a denoise processor
 */
denoise_handle_t denoise_create(const denoise_config_t *config)
{
    if (config == NULL) {
        return NULL;
    }

    denoise_processor_t *processor = (denoise_processor_t *)malloc(sizeof(denoise_processor_t));
    if (processor == NULL) {
        return NULL;
    }

    memcpy(&processor->config, config, sizeof(denoise_config_t));

    // Calculate buffer size based on sample rate and channels
    // Assuming 20ms frames at given sample rate
    int frame_samples = (config->sample_rate / 1000) * 20;
    processor->buffer_size = frame_samples * config->channels * (config->bits_per_sample / 8);

    // Allocate buffers for AEC and AGC processing
    processor->aec_buffer = (int16_t *)malloc(processor->buffer_size);
    processor->agc_buffer = (int16_t *)malloc(processor->buffer_size);

    if (processor->aec_buffer == NULL || processor->agc_buffer == NULL) {
        free(processor->aec_buffer);
        free(processor->agc_buffer);
        free(processor);
        return NULL;
    }

    processor->initialized = true;

    return (denoise_handle_t)processor;
}

/**
 * @brief Simple AEC processing (echo cancellation)
 * This is a placeholder implementation. Real AEC would use more sophisticated algorithms.
 */
static void aec_process(int16_t *input, int16_t *output, int len)
{
    // Simple echo suppression: reduce amplitude by 50%
    for (int i = 0; i < len; i++) {
        output[i] = (int16_t)(input[i] * 0.5f);
    }
}

/**
 * @brief Simple AGC processing (automatic gain control)
 * This is a placeholder implementation. Real AGC would use more sophisticated algorithms.
 */
static void agc_process(int16_t *input, int16_t *output, int len)
{
    // Find peak level
    int16_t peak = 0;
    for (int i = 0; i < len; i++) {
        int16_t abs_val = input[i] < 0 ? -input[i] : input[i];
        if (abs_val > peak) {
            peak = abs_val;
        }
    }

    // Calculate gain to normalize to 80% of max
    float target_level = 32767 * 0.8f;
    float gain = (peak > 0) ? (target_level / peak) : 1.0f;

    // Apply gain
    for (int i = 0; i < len; i++) {
        int32_t sample = (int32_t)(input[i] * gain);
        // Clamp to int16 range
        if (sample > 32767) sample = 32767;
        if (sample < -32768) sample = -32768;
        output[i] = (int16_t)sample;
    }
}

/**
 * @brief Process audio data with denoise
 */
int denoise_process(denoise_handle_t handle, const uint8_t *input, int input_len, uint8_t *output)
{
    if (handle == NULL || input == NULL || output == NULL) {
        return -1;
    }

    denoise_processor_t *processor = (denoise_processor_t *)handle;

    if (!processor->initialized) {
        return -2;
    }

    // Convert input bytes to int16 samples
    int16_t *input_samples = (int16_t *)input;
    int16_t *output_samples = (int16_t *)output;
    int sample_count = input_len / sizeof(int16_t);

    // Apply AEC if enabled
    if (processor->config.aec_enable) {
        aec_process(input_samples, processor->aec_buffer, sample_count);
        memcpy(output_samples, processor->aec_buffer, input_len);
    } else {
        memcpy(output_samples, input_samples, input_len);
    }

    // Apply AGC if enabled
    if (processor->config.agc_enable) {
        agc_process(output_samples, processor->agc_buffer, sample_count);
        memcpy(output_samples, processor->agc_buffer, input_len);
    }

    return input_len;
}

/**
 * @brief Process audio data in-place
 */
int denoise_process_inplace(denoise_handle_t handle, uint8_t *data, int data_len)
{
    if (handle == NULL || data == NULL) {
        return -1;
    }

    denoise_processor_t *processor = (denoise_processor_t *)handle;

    if (!processor->initialized) {
        return -2;
    }

    int16_t *samples = (int16_t *)data;
    int sample_count = data_len / sizeof(int16_t);

    // Apply AEC if enabled
    if (processor->config.aec_enable) {
        aec_process(samples, processor->aec_buffer, sample_count);
        memcpy(samples, processor->aec_buffer, data_len);
    }

    // Apply AGC if enabled
    if (processor->config.agc_enable) {
        agc_process(samples, processor->agc_buffer, sample_count);
        memcpy(samples, processor->agc_buffer, data_len);
    }

    return data_len;
}

/**
 * @brief Reset denoise processor state
 */
int denoise_reset(denoise_handle_t handle)
{
    if (handle == NULL) {
        return -1;
    }

    denoise_processor_t *processor = (denoise_processor_t *)handle;

    // Clear buffers
    if (processor->aec_buffer != NULL) {
        memset(processor->aec_buffer, 0, processor->buffer_size);
    }
    if (processor->agc_buffer != NULL) {
        memset(processor->agc_buffer, 0, processor->buffer_size);
    }

    return 0;
}

/**
 * @brief Set AEC enable status
 */
int denoise_set_aec_enable(denoise_handle_t handle, bool enable)
{
    if (handle == NULL) {
        return -1;
    }

    denoise_processor_t *processor = (denoise_processor_t *)handle;
    processor->config.aec_enable = enable;

    return 0;
}

/**
 * @brief Set AGC enable status
 */
int denoise_set_agc_enable(denoise_handle_t handle, bool enable)
{
    if (handle == NULL) {
        return -1;
    }

    denoise_processor_t *processor = (denoise_processor_t *)handle;
    processor->config.agc_enable = enable;

    return 0;
}

/**
 * @brief Destroy denoise processor
 */
int denoise_destroy(denoise_handle_t handle)
{
    if (handle == NULL) {
        return -1;
    }

    denoise_processor_t *processor = (denoise_processor_t *)handle;

    if (processor->aec_buffer != NULL) {
        free(processor->aec_buffer);
    }
    if (processor->agc_buffer != NULL) {
        free(processor->agc_buffer);
    }

    free(processor);

    return 0;
}

/**
 * @brief Get denoise library version
 */
const char* denoise_version(void)
{
    return DENOISE_VERSION;
}
