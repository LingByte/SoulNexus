package utils

import (
	"encoding/json"
	"fmt"
	"math"

	"go.uber.org/zap"
)

// MouseTrackPoint 鼠标轨迹点
type MouseTrackPoint struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
}

// BehaviorAnalysis 行为分析结果
type BehaviorAnalysis struct {
	IsBot                bool     `json:"isBot"`
	Confidence           float64  `json:"confidence"`           // 置信度 0-1
	StraightLineRatio    float64  `json:"straightLineRatio"`    // 直线比例
	AvgSpeed             float64  `json:"avgSpeed"`             // 平均速度
	SpeedVariance        float64  `json:"speedVariance"`        // 速度方差
	DirectionChanges     int      `json:"directionChanges"`     // 方向改变次数
	PauseCount           int      `json:"pauseCount"`           // 停顿次数
	FormFillTime         int64    `json:"formFillTime"`         // 表单填写时间(毫秒)
	KeystrokePattern     string   `json:"keystrokePattern"`     // 按键模式
	SuspiciousIndicators []string `json:"suspiciousIndicators"` // 可疑指标
}

// IntelligentRiskControl 智能风控管理器
type IntelligentRiskControl struct {
	logger *zap.Logger

	// 鼠标轨迹检测配置
	minTrackPoints       int     // 最少轨迹点数
	maxStraightLineRatio float64 // 最大直线比例阈值
	minSpeedVariance     float64 // 最小速度方差阈值
	maxAvgSpeed          float64 // 最大平均速度阈值
	minDirectionChanges  int     // 最少方向改变次数
	minPauseCount        int     // 最少停顿次数

	// 时间行为检测配置
	minFormFillTime       int64 // 最小表单填写时间(毫秒)
	maxFormFillTime       int64 // 最大表单填写时间(毫秒)
	suspiciousSpeedThresh int64 // 可疑填写速度阈值(字符/秒)

	// 风险评分权重
	mouseTrackWeight float64 // 鼠标轨迹权重
	timingWeight     float64 // 时间行为权重
	keystrokeWeight  float64 // 按键模式权重
	botThreshold     float64 // 机器人判定阈值
}

// NewIntelligentRiskControl 创建智能风控管理器
func NewIntelligentRiskControl(logger *zap.Logger) *IntelligentRiskControl {
	return &IntelligentRiskControl{
		logger: logger,

		// 鼠标轨迹检测配置
		minTrackPoints:       10,   // 至少10个轨迹点
		maxStraightLineRatio: 0.8,  // 直线比例不超过80%
		minSpeedVariance:     100,  // 速度方差至少100
		maxAvgSpeed:          2000, // 平均速度不超过2000px/s
		minDirectionChanges:  3,    // 至少3次方向改变
		minPauseCount:        1,    // 至少1次停顿

		// 时间行为检测配置
		minFormFillTime:       5000,   // 最少5秒填写时间
		maxFormFillTime:       300000, // 最多5分钟填写时间
		suspiciousSpeedThresh: 10,     // 每秒超过10个字符可疑

		// 风险评分权重
		mouseTrackWeight: 0.4,
		timingWeight:     0.3,
		keystrokeWeight:  0.3,
		botThreshold:     0.6, // 60%以上置信度判定为机器人 (降低阈值使其更敏感)
	}
}

// AnalyzeBehavior 分析用户行为
func (irc *IntelligentRiskControl) AnalyzeBehavior(
	mouseTrack []MouseTrackPoint,
	formFillTime int64,
	keystrokePattern string,
	formData map[string]string,
) (*BehaviorAnalysis, error) {

	analysis := &BehaviorAnalysis{
		FormFillTime:         formFillTime,
		KeystrokePattern:     keystrokePattern,
		SuspiciousIndicators: make([]string, 0),
	}

	// 1. 分析鼠标轨迹
	mouseScore := irc.analyzeMouseTrack(mouseTrack, analysis)

	// 2. 分析时间行为
	timingScore := irc.analyzeTimingBehavior(formFillTime, formData, analysis)

	// 3. 分析按键模式
	keystrokeScore := irc.analyzeKeystrokePattern(keystrokePattern, analysis)

	// 4. 计算综合风险评分
	totalScore := mouseScore*irc.mouseTrackWeight +
		timingScore*irc.timingWeight +
		keystrokeScore*irc.keystrokeWeight

	analysis.Confidence = totalScore
	analysis.IsBot = totalScore >= irc.botThreshold

	irc.logger.Info("Behavior analysis completed",
		zap.Float64("mouseScore", mouseScore),
		zap.Float64("timingScore", timingScore),
		zap.Float64("keystrokeScore", keystrokeScore),
		zap.Float64("totalScore", totalScore),
		zap.Bool("isBot", analysis.IsBot),
		zap.Strings("indicators", analysis.SuspiciousIndicators))

	return analysis, nil
}

// analyzeMouseTrack 分析鼠标轨迹
func (irc *IntelligentRiskControl) analyzeMouseTrack(track []MouseTrackPoint, analysis *BehaviorAnalysis) float64 {
	if len(track) < irc.minTrackPoints {
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "insufficient_mouse_track_points")
		return 0.9 // 轨迹点太少，高度可疑
	}

	// 计算轨迹特征
	straightLineRatio := irc.calculateStraightLineRatio(track)
	avgSpeed, speedVariance := irc.calculateSpeedMetrics(track)
	directionChanges := irc.calculateDirectionChanges(track)
	pauseCount := irc.calculatePauseCount(track)

	analysis.StraightLineRatio = straightLineRatio
	analysis.AvgSpeed = avgSpeed
	analysis.SpeedVariance = speedVariance
	analysis.DirectionChanges = directionChanges
	analysis.PauseCount = pauseCount

	suspiciousScore := 0.0

	// 检查直线比例
	if straightLineRatio > irc.maxStraightLineRatio {
		suspiciousScore += 0.3
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "high_straight_line_ratio")
	}

	// 检查速度方差
	if speedVariance < irc.minSpeedVariance {
		suspiciousScore += 0.25
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "low_speed_variance")
	}

	// 检查平均速度
	if avgSpeed > irc.maxAvgSpeed {
		suspiciousScore += 0.2
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "high_average_speed")
	}

	// 检查方向改变次数
	if directionChanges < irc.minDirectionChanges {
		suspiciousScore += 0.15
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "few_direction_changes")
	}

	// 检查停顿次数
	if pauseCount < irc.minPauseCount {
		suspiciousScore += 0.1
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "few_pauses")
	}

	return math.Min(suspiciousScore, 1.0)
}

// calculateStraightLineRatio 计算直线比例
func (irc *IntelligentRiskControl) calculateStraightLineRatio(track []MouseTrackPoint) float64 {
	if len(track) < 3 {
		return 1.0 // 点太少，认为是直线
	}

	straightSegments := 0
	totalSegments := len(track) - 2

	for i := 1; i < len(track)-1; i++ {
		// 计算三点是否接近直线
		p1, p2, p3 := track[i-1], track[i], track[i+1]

		// 使用向量叉积判断是否接近直线
		v1x, v1y := p2.X-p1.X, p2.Y-p1.Y
		v2x, v2y := p3.X-p2.X, p3.Y-p2.Y

		crossProduct := math.Abs(float64(v1x*v2y - v1y*v2x))

		// 如果叉积很小，说明接近直线
		if crossProduct < 50 { // 阈值可调整
			straightSegments++
		}
	}

	return float64(straightSegments) / float64(totalSegments)
}

// calculateSpeedMetrics 计算速度指标
func (irc *IntelligentRiskControl) calculateSpeedMetrics(track []MouseTrackPoint) (avgSpeed, speedVariance float64) {
	if len(track) < 2 {
		return 0, 0
	}

	speeds := make([]float64, 0, len(track)-1)

	for i := 1; i < len(track); i++ {
		p1, p2 := track[i-1], track[i]

		// 计算距离
		dx := float64(p2.X - p1.X)
		dy := float64(p2.Y - p1.Y)
		distance := math.Sqrt(dx*dx + dy*dy)

		// 计算时间差(秒)
		timeDiff := float64(p2.Timestamp-p1.Timestamp) / 1000.0
		if timeDiff > 0 {
			speed := distance / timeDiff
			speeds = append(speeds, speed)
		}
	}

	if len(speeds) == 0 {
		return 0, 0
	}

	// 计算平均速度
	sum := 0.0
	for _, speed := range speeds {
		sum += speed
	}
	avgSpeed = sum / float64(len(speeds))

	// 计算方差
	varianceSum := 0.0
	for _, speed := range speeds {
		diff := speed - avgSpeed
		varianceSum += diff * diff
	}
	speedVariance = varianceSum / float64(len(speeds))

	return avgSpeed, speedVariance
}

// calculateDirectionChanges 计算方向改变次数
func (irc *IntelligentRiskControl) calculateDirectionChanges(track []MouseTrackPoint) int {
	if len(track) < 3 {
		return 0
	}

	changes := 0
	prevDirection := 0.0

	for i := 2; i < len(track); i++ {
		p1, p2, p3 := track[i-2], track[i-1], track[i]

		// 计算两个向量的角度
		v1x, v1y := p2.X-p1.X, p2.Y-p1.Y
		v2x, v2y := p3.X-p2.X, p3.Y-p2.Y

		angle1 := math.Atan2(float64(v1y), float64(v1x))
		angle2 := math.Atan2(float64(v2y), float64(v2x))

		angleDiff := math.Abs(angle2 - angle1)
		if angleDiff > math.Pi {
			angleDiff = 2*math.Pi - angleDiff
		}

		// 如果角度变化超过30度，认为是方向改变
		if angleDiff > math.Pi/6 && prevDirection != 0 {
			changes++
		}
		prevDirection = angle2
	}

	return changes
}

// calculatePauseCount 计算停顿次数
func (irc *IntelligentRiskControl) calculatePauseCount(track []MouseTrackPoint) int {
	if len(track) < 2 {
		return 0
	}

	pauses := 0
	pauseThreshold := int64(200) // 200ms以上认为是停顿

	for i := 1; i < len(track); i++ {
		timeDiff := track[i].Timestamp - track[i-1].Timestamp
		if timeDiff > pauseThreshold {
			pauses++
		}
	}

	return pauses
}

// analyzeTimingBehavior 分析时间行为
func (irc *IntelligentRiskControl) analyzeTimingBehavior(formFillTime int64, formData map[string]string, analysis *BehaviorAnalysis) float64 {
	suspiciousScore := 0.0

	// 检查填写时间
	if formFillTime < irc.minFormFillTime {
		suspiciousScore += 0.4
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "too_fast_form_fill")
	} else if formFillTime > irc.maxFormFillTime {
		suspiciousScore += 0.2
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "too_slow_form_fill")
	}

	// 计算输入速度
	totalChars := 0
	for _, value := range formData {
		totalChars += len(value)
	}

	if totalChars > 0 && formFillTime > 0 {
		charsPerSecond := float64(totalChars) / (float64(formFillTime) / 1000.0)
		if charsPerSecond > float64(irc.suspiciousSpeedThresh) {
			suspiciousScore += 0.3
			analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "suspicious_typing_speed")
		}
	}

	return math.Min(suspiciousScore, 1.0)
}

// analyzeKeystrokePattern 分析按键模式
func (irc *IntelligentRiskControl) analyzeKeystrokePattern(pattern string, analysis *BehaviorAnalysis) float64 {
	if pattern == "" {
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "no_keystroke_pattern")
		return 0.5
	}

	suspiciousScore := 0.0

	// 解析按键模式（假设格式为JSON数组，包含按键间隔时间）
	var intervals []int64
	if err := json.Unmarshal([]byte(pattern), &intervals); err != nil {
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "invalid_keystroke_pattern")
		return 0.3
	}

	if len(intervals) < 2 {
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "insufficient_keystroke_data")
		return 0.4
	}

	// 计算按键间隔的方差
	sum := int64(0)
	for _, interval := range intervals {
		sum += interval
	}
	avgInterval := float64(sum) / float64(len(intervals))

	variance := 0.0
	for _, interval := range intervals {
		diff := float64(interval) - avgInterval
		variance += diff * diff
	}
	variance /= float64(len(intervals))

	// 如果方差太小，说明按键间隔过于规律，可能是机器人
	if variance < 100 { // 阈值可调整
		suspiciousScore += 0.4
		analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "regular_keystroke_intervals")
	}

	// 检查是否有异常快速的按键
	for _, interval := range intervals {
		if interval < 50 { // 50ms以下的按键间隔异常
			suspiciousScore += 0.2
			analysis.SuspiciousIndicators = append(analysis.SuspiciousIndicators, "abnormal_fast_keystrokes")
			break
		}
	}

	return math.Min(suspiciousScore, 1.0)
}

// CheckRegistrationRisk 检查注册风险
func (irc *IntelligentRiskControl) CheckRegistrationRisk(
	mouseTrack []MouseTrackPoint,
	formFillTime int64,
	keystrokePattern string,
	formData map[string]string,
) error {

	analysis, err := irc.AnalyzeBehavior(mouseTrack, formFillTime, keystrokePattern, formData)
	if err != nil {
		return fmt.Errorf("behavior analysis failed: %w", err)
	}

	if analysis.IsBot {
		return fmt.Errorf("registration blocked: suspicious bot behavior detected (confidence: %.2f)", analysis.Confidence)
	}

	// 记录分析结果用于后续优化
	irc.logger.Info("Registration risk analysis",
		zap.Bool("isBot", analysis.IsBot),
		zap.Float64("confidence", analysis.Confidence),
		zap.Strings("indicators", analysis.SuspiciousIndicators))

	return nil
}

// 全局智能风控管理器
var GlobalIntelligentRiskControl *IntelligentRiskControl

// InitGlobalIntelligentRiskControl 初始化全局智能风控管理器
func InitGlobalIntelligentRiskControl(logger *zap.Logger) {
	GlobalIntelligentRiskControl = NewIntelligentRiskControl(logger)
}
