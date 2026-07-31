package main

import (
	"os"
	"strconv"
)

// 默认阈值（与历史硬编码一致，未设环境变量时零回归）
const (
	defaultYoloScoreThreshold = float32(0.2)
	defaultFenceTolerance     = float32(0.03)
	defaultLowerHeadPitch     = float32(-9)
	defaultTurnHeadYawAbs     = float32(50)
	defaultTurnHeadRollAbs    = float32(25)
	defaultTurnHeadPitchMax   = float32(15)
)

var (
	yoloPersonThreshold    = defaultYoloScoreThreshold
	yoloPhoneThreshold     = defaultYoloScoreThreshold
	yoloRemoteThreshold    = defaultYoloScoreThreshold
	yoloBookThreshold      = defaultYoloScoreThreshold
	fenceTolerance         = defaultFenceTolerance
	lowerHeadPitchThresh   = defaultLowerHeadPitch
	turnHeadYawAbsThresh   = defaultTurnHeadYawAbs
	turnHeadRollAbsThresh  = defaultTurnHeadRollAbs
	turnHeadPitchMaxThresh = defaultTurnHeadPitchMax
)

func envFloatOrDefault(key string, def float32) float32 {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 32)
	if err != nil {
		getLogger().Warn("invalid_env_threshold", "key", key, "value", raw)
		return def
	}
	return float32(v)
}

// loadDetectorThresholds 从环境变量覆盖包级阈值，默认值与历史行为一致。
// 仅应在 InitDetector 中调用一次；AnalyzeImage 依赖 Init 完成后的可见性，勿在运行中热更。
func loadDetectorThresholds() {
	yoloPersonThreshold = envFloatOrDefault("YKS_YOLO_PERSON_THRESHOLD", defaultYoloScoreThreshold)
	yoloPhoneThreshold = envFloatOrDefault("YKS_YOLO_PHONE_THRESHOLD", defaultYoloScoreThreshold)
	yoloRemoteThreshold = envFloatOrDefault("YKS_YOLO_REMOTE_THRESHOLD", defaultYoloScoreThreshold)
	yoloBookThreshold = envFloatOrDefault("YKS_YOLO_BOOK_THRESHOLD", defaultYoloScoreThreshold)
	fenceTolerance = envFloatOrDefault("YKS_FENCE_TOLERANCE", defaultFenceTolerance)
	lowerHeadPitchThresh = envFloatOrDefault("YKS_LOWER_HEAD_PITCH", defaultLowerHeadPitch)
	turnHeadYawAbsThresh = envFloatOrDefault("YKS_TURN_HEAD_YAW", defaultTurnHeadYawAbs)
	turnHeadRollAbsThresh = envFloatOrDefault("YKS_TURN_HEAD_ROLL", defaultTurnHeadRollAbs)
	turnHeadPitchMaxThresh = envFloatOrDefault("YKS_TURN_HEAD_PITCH_MAX", defaultTurnHeadPitchMax)

	getLogger().Info("detector_thresholds_loaded",
		"yolo_person", yoloPersonThreshold,
		"yolo_phone", yoloPhoneThreshold,
		"yolo_remote", yoloRemoteThreshold,
		"yolo_book", yoloBookThreshold,
		"fence_tolerance", fenceTolerance,
		"lower_head_pitch", lowerHeadPitchThresh,
		"turn_head_yaw", turnHeadYawAbsThresh,
		"turn_head_roll", turnHeadRollAbsThresh,
		"turn_head_pitch_max", turnHeadPitchMaxThresh,
	)
}

// yoloParseScoreFloor 解析阶段下限：取各类别阈值最小值，避免过早丢弃低阈值类别
func yoloParseScoreFloor() float32 {
	floor := yoloPersonThreshold
	for _, t := range []float32{yoloPhoneThreshold, yoloRemoteThreshold, yoloBookThreshold} {
		if t < floor {
			floor = t
		}
	}
	return floor
}

func yoloThresholdForClass(name string) (float32, bool) {
	switch name {
	case "person":
		return yoloPersonThreshold, true
	case "cell phone":
		return yoloPhoneThreshold, true
	case "remote":
		return yoloRemoteThreshold, true
	case "book":
		return yoloBookThreshold, true
	default:
		return 0, false
	}
}
