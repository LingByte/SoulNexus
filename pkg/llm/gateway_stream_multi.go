// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 多渠道流式 fallback：与 RelayOpenAIStreamWithCapture / RelayAnthropicStreamWithCapture 的差别在于，
// 在 *拿到 upstream 响应头* 之前不向 client 写入任何字节。当 status >= 400 时关闭连接换下一个 channel；
// 一旦 2xx 就提交（commit）该 channel，复用单 channel 的 SSE 透传逻辑。
package llm

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

var errStreamNoChannel = errors.New("no channel available")

// shouldFailoverFromStatus 仅在常见的"上游可重试"错误码上做 fallback：
// 5xx + 429 + 408 视为可重试；其它一律 commit（避免误把客户端错误也试一遍）。
func shouldFailoverFromStatus(status int) bool {
	if status >= 500 {
		return true
	}
	if status == http.StatusTooManyRequests || status == http.StatusRequestTimeout {
		return true
	}
	return false
}

// RelayOpenAIStreamMulti 在 channels 上做 fallback 流式转发。
// 返回值与单 channel 版兼容：cap 包含最终成功 channel 的统计，attempts 记录所有尝试。
func RelayOpenAIStreamMulti(ctx context.Context, body []byte, accept string, channels []UpstreamChannel, w http.ResponseWriter) (*OpenAIStreamCapture, []UsageChannelAttempt, int, error) {
	if len(channels) == 0 {
		return nil, nil, 0, errStreamNoChannel
	}
	attempts := make([]UsageChannelAttempt, 0, len(channels))
	var lastErr error
	for i, ch := range channels {
		t0 := time.Now()
		bu := ch.baseURLString()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIChatCompletionsURL(ch.BaseURL), bytes.NewReader(body))
		if err != nil {
			attempts = append(attempts, UsageChannelAttempt{
				Order: i + 1, ChannelID: ch.ID, BaseURL: bu, Success: false,
				ErrorCode: "request_build_error", ErrorMessage: truncateRelayAttemptMsg(err.Error(), maxRelayAttemptErrBytes),
				LatencyMs: time.Since(t0).Milliseconds(),
			})
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if strings.TrimSpace(accept) != "" {
			req.Header.Set("Accept", accept)
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(ch.Key))
		if ch.OpenAIOrganization != nil && strings.TrimSpace(*ch.OpenAIOrganization) != "" {
			req.Header.Set("OpenAI-Organization", strings.TrimSpace(*ch.OpenAIOrganization))
		}
		resp, err := relayUpstreamHTTPClient.Do(req)
		if err != nil {
			attempts = append(attempts, UsageChannelAttempt{
				Order: i + 1, ChannelID: ch.ID, BaseURL: bu, Success: false,
				ErrorCode: "transport_error", ErrorMessage: truncateRelayAttemptMsg(err.Error(), maxRelayAttemptErrBytes),
				LatencyMs: time.Since(t0).Milliseconds(),
			})
			lastErr = err
			continue
		}
		// 上游错误：关闭并 fallback；只有最后一个 channel 才把错误体写回客户端。
		if resp.StatusCode >= 400 && shouldFailoverFromStatus(resp.StatusCode) && i < len(channels)-1 {
			buf, _ := io.ReadAll(io.LimitReader(resp.Body, maxRelayAttemptErrBytes*4))
			resp.Body.Close()
			attempts = append(attempts, UsageChannelAttempt{
				Order: i + 1, ChannelID: ch.ID, BaseURL: bu, Success: false,
				StatusCode: resp.StatusCode, ErrorCode: "upstream_error",
				ErrorMessage: truncateRelayAttemptMsg(string(buf), maxRelayAttemptErrBytes),
				LatencyMs:    time.Since(t0).Milliseconds(),
			})
			continue
		}
		// 提交该 channel：开始向客户端写
		cap := &OpenAIStreamCapture{}
		overall := time.Now()
		cap.StartedAtMs = overall.UnixMilli()
		cap.StatusCode = resp.StatusCode
		CopyRelayResponseHeaders(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		fl, _ := w.(http.Flusher)
		br := bufio.NewReader(resp.Body)
		var firstTokenRecorded bool
		var streamErr error
		for {
			line, rerr := br.ReadBytes('\n')
			if len(line) > 0 {
				if _, werr := w.Write(line); werr != nil {
					streamErr = werr
					break
				}
				if fl != nil {
					fl.Flush()
				}
				parseOpenAISSELine(line, cap, &firstTokenRecorded)
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				streamErr = rerr
				break
			}
		}
		resp.Body.Close()
		cap.CompletedAtMs = time.Now().UnixMilli()
		cap.WallLatencyMs = time.Since(overall).Milliseconds()
		if cap.FirstTokenAtMs == 0 {
			cap.FirstTokenAtMs = cap.CompletedAtMs
		}
		attempts = append(attempts, UsageChannelAttempt{
			Order: i + 1, ChannelID: ch.ID, BaseURL: bu,
			Success:    streamErr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300,
			StatusCode: resp.StatusCode, LatencyMs: time.Since(t0).Milliseconds(),
			ErrorCode:    errCodeIfErr(streamErr, "stream_error"),
			ErrorMessage: errMsgIfErr(streamErr),
		})
		return cap, attempts, ch.ID, streamErr
	}
	return nil, attempts, 0, lastErr
}

// RelayAnthropicStreamMulti 与 OpenAI 版相同语义。
func RelayAnthropicStreamMulti(ctx context.Context, body []byte, anthropicVersion, anthropicBeta string, channels []UpstreamChannel, w http.ResponseWriter) (*AnthropicStreamCapture, []UsageChannelAttempt, int, error) {
	if len(channels) == 0 {
		return nil, nil, 0, errStreamNoChannel
	}
	attempts := make([]UsageChannelAttempt, 0, len(channels))
	var lastErr error
	for i, ch := range channels {
		t0 := time.Now()
		bu := ch.baseURLString()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicMessagesURL(ch.BaseURL), bytes.NewReader(body))
		if err != nil {
			attempts = append(attempts, UsageChannelAttempt{
				Order: i + 1, ChannelID: ch.ID, BaseURL: bu, Success: false,
				ErrorCode: "request_build_error", ErrorMessage: truncateRelayAttemptMsg(err.Error(), maxRelayAttemptErrBytes),
				LatencyMs: time.Since(t0).Milliseconds(),
			})
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if strings.TrimSpace(anthropicVersion) != "" {
			req.Header.Set("anthropic-version", anthropicVersion)
		} else {
			req.Header.Set("anthropic-version", "2023-06-01")
		}
		if strings.TrimSpace(anthropicBeta) != "" {
			req.Header.Set("anthropic-beta", anthropicBeta)
		}
		req.Header.Set("x-api-key", strings.TrimSpace(ch.Key))
		resp, err := relayUpstreamHTTPClient.Do(req)
		if err != nil {
			attempts = append(attempts, UsageChannelAttempt{
				Order: i + 1, ChannelID: ch.ID, BaseURL: bu, Success: false,
				ErrorCode: "transport_error", ErrorMessage: truncateRelayAttemptMsg(err.Error(), maxRelayAttemptErrBytes),
				LatencyMs: time.Since(t0).Milliseconds(),
			})
			lastErr = err
			continue
		}
		if resp.StatusCode >= 400 && shouldFailoverFromStatus(resp.StatusCode) && i < len(channels)-1 {
			buf, _ := io.ReadAll(io.LimitReader(resp.Body, maxRelayAttemptErrBytes*4))
			resp.Body.Close()
			attempts = append(attempts, UsageChannelAttempt{
				Order: i + 1, ChannelID: ch.ID, BaseURL: bu, Success: false,
				StatusCode: resp.StatusCode, ErrorCode: "upstream_error",
				ErrorMessage: truncateRelayAttemptMsg(string(buf), maxRelayAttemptErrBytes),
				LatencyMs:    time.Since(t0).Milliseconds(),
			})
			continue
		}
		cap := &AnthropicStreamCapture{}
		overall := time.Now()
		cap.StartedAtMs = overall.UnixMilli()
		cap.StatusCode = resp.StatusCode
		CopyRelayResponseHeaders(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		fl, _ := w.(http.Flusher)
		br := bufio.NewReader(resp.Body)
		var firstContent bool
		var streamErr error
		for {
			line, rerr := br.ReadBytes('\n')
			if len(line) > 0 {
				if _, werr := w.Write(line); werr != nil {
					streamErr = werr
					break
				}
				if fl != nil {
					fl.Flush()
				}
				parseAnthropicSSELine(line, cap, &firstContent)
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				streamErr = rerr
				break
			}
		}
		resp.Body.Close()
		cap.CompletedAtMs = time.Now().UnixMilli()
		cap.WallLatencyMs = time.Since(overall).Milliseconds()
		if cap.FirstTokenAtMs == 0 {
			cap.FirstTokenAtMs = cap.CompletedAtMs
		}
		attempts = append(attempts, UsageChannelAttempt{
			Order: i + 1, ChannelID: ch.ID, BaseURL: bu,
			Success:    streamErr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300,
			StatusCode: resp.StatusCode, LatencyMs: time.Since(t0).Milliseconds(),
			ErrorCode:    errCodeIfErr(streamErr, "stream_error"),
			ErrorMessage: errMsgIfErr(streamErr),
		})
		return cap, attempts, ch.ID, streamErr
	}
	return nil, attempts, 0, lastErr
}

func errCodeIfErr(err error, code string) string {
	if err == nil {
		return ""
	}
	return code
}

func errMsgIfErr(err error) string {
	if err == nil {
		return ""
	}
	return truncateRelayAttemptMsg(err.Error(), maxRelayAttemptErrBytes)
}
