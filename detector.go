package main

import (
	"fmt"
	"image"
	"math"
	"os"
	"sort"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

const (
	yoloInputSize              = 640
	yoloScoreThreshold         = 0.5
	yoloIouThreshold           = 0.45
	fenceMarginLeft            = 0.2
	fenceMarginRight           = 0.2
	fenceMarginTop             = 0.2
	fenceMarginBottom          = 0.2
	faceInputSize              = 640
	faceScoreThreshold         = 0.6
	faceNmsThreshold           = 0.3
	faceMatchThreshold         = 0.4
	faceChangeQualityThreshold = 0.7
	lowerHeadPitchThreshold    = -20
	turnHeadYawThreshold       = 30
)

const (
	codeNobody       = 1001
	codeMultiPerson  = 1002
	codePhone        = 1003
	codeBook         = 1004
	codeChangePerson = 1005
	codeLowerHead    = 2001
	codeTurnHead     = 2002
	codeRange        = 2003
)

var cocoLabels = []string{
	"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat",
	"traffic light", "fire hydrant", "stop sign", "parking meter", "bench", "bird", "cat",
	"dog", "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe", "backpack",
	"umbrella", "handbag", "tie", "suitcase", "frisbee", "skis", "snowboard", "sports ball",
	"kite", "baseball bat", "baseball glove", "skateboard", "surfboard", "tennis racket",
	"bottle", "wine glass", "cup", "fork", "knife", "spoon", "bowl", "banana", "apple",
	"sandwich", "orange", "broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair",
	"couch", "potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse",
	"remote", "keyboard", "cell phone", "microwave", "oven", "toaster", "sink", "refrigerator",
	"book", "clock", "vase", "scissors", "teddy bear", "hair drier", "toothbrush",
}

// DetectionFlags 与 aiIdentification meta.ts 告警键一致
type DetectionFlags struct {
	NobodyPC         bool `json:"nobodyPC"`
	MultiplePersonPC bool `json:"multiplePersonPC"`
	FindPhonePC      bool `json:"findPhonePC"`
	FindBookPC       bool `json:"findBookPC"`
	LowerHeadPC      bool `json:"lowerHeadPC"`
	TurnheadPC       bool `json:"turnheadPC"`
	RangeTestPC      bool `json:"rangeTestPC"`
	ChangePersonPC   bool `json:"changePersonPC"`
}

type DetectionResult struct {
	Flags DetectionFlags `json:"detection"`
	Codes []int          `json:"codes"`
}

type yoloBox struct {
	x1, y1, x2, y2 float32
	classID        int
	score          float32
}

type faceLandmarks struct {
	x, y, w, h               float32
	rightEyeX, rightEyeY     float32
	leftEyeX, leftEyeY       float32
	noseX, noseY             float32
	rightMouthX, rightMouthY float32
	leftMouthX, leftMouthY   float32
	score                    float32
}

var (
	detectorMu      sync.RWMutex
	detectorReady   bool
	detectorInitErr error
	masterEmbedding []float32
	yoloSession     *ort.DynamicAdvancedSession
	faceDetectSess  *ort.DynamicAdvancedSession
	faceRecSess     *ort.DynamicAdvancedSession
)

// IsDetectorReady 模型会话是否已加载完成
func IsDetectorReady() bool {
	detectorMu.RLock()
	defer detectorMu.RUnlock()
	return detectorReady
}

// DetectorInitError 后台初始化失败原因（仍在加载时返回 nil）
func DetectorInitError() error {
	detectorMu.RLock()
	defer detectorMu.RUnlock()
	return detectorInitErr
}

// DetectorStatus 供健康检查：ready / loading / error / skipped
func DetectorStatus() string {
	if os.Getenv("YKS_SKIP_DETECTOR") == "1" {
		return "skipped"
	}
	detectorMu.RLock()
	defer detectorMu.RUnlock()
	if detectorReady {
		return "ready"
	}
	if detectorInitErr != nil {
		return "error"
	}
	return "loading"
}

// InitDetector 从嵌入资源加载 ONNX 模型（可异步调用）
func InitDetector() error {
	ortPath, err := materializeOrtLib()
	if err != nil {
		setDetectorInitErr(err)
		return err
	}
	ort.SetSharedLibraryPath(ortPath)

	if err := ort.InitializeEnvironment(); err != nil {
		err = fmt.Errorf("onnxruntime init: %w", err)
		setDetectorInitErr(err)
		return err
	}

	yoloData, err := loadModelONNX("yolo11.onnx")
	if err != nil {
		setDetectorInitErr(err)
		return err
	}
	faceDetectData, err := loadModelONNX("face_detect.onnx")
	if err != nil {
		setDetectorInitErr(err)
		return err
	}
	faceRecData, err := loadModelONNX("face_rec.onnx")
	if err != nil {
		setDetectorInitErr(err)
		return err
	}

	yoloSession, err = ort.NewDynamicAdvancedSessionWithONNXData(yoloData, []string{"images"}, []string{"output0"}, nil)
	if err != nil {
		err = fmt.Errorf("yolo session: %w", err)
		setDetectorInitErr(err)
		return err
	}

	faceOutputNames := []string{
		"cls_8", "cls_16", "cls_32",
		"obj_8", "obj_16", "obj_32",
		"bbox_8", "bbox_16", "bbox_32",
		"kps_8", "kps_16", "kps_32",
	}
	faceDetectSess, err = ort.NewDynamicAdvancedSessionWithONNXData(faceDetectData, []string{"input"}, faceOutputNames, nil)
	if err != nil {
		err = fmt.Errorf("face detect session: %w", err)
		setDetectorInitErr(err)
		return err
	}

	faceRecSess, err = ort.NewDynamicAdvancedSessionWithONNXData(faceRecData, []string{"input.1"}, []string{"516"}, nil)
	if err != nil {
		err = fmt.Errorf("face rec session: %w", err)
		setDetectorInitErr(err)
		return err
	}

	detectorMu.Lock()
	detectorReady = true
	detectorInitErr = nil
	detectorMu.Unlock()
	getLogger().Info("detector_initialized", "ort_dll", ortPath, "models", "embedded")
	return nil
}

func setDetectorInitErr(err error) {
	detectorMu.Lock()
	detectorInitErr = err
	detectorReady = false
	detectorMu.Unlock()
}

// SetMasterFace 从图片提取基准人脸 embedding
func SetMasterFace(img image.Image) error {
	if !IsDetectorReady() {
		return fmt.Errorf("detector not initialized")
	}
	face, ok := detectPrimaryFace(img)
	if !ok {
		return fmt.Errorf("no face detected in master image")
	}
	emb, err := extractFaceEmbedding(img, face)
	if err != nil {
		return err
	}
	detectorMu.Lock()
	masterEmbedding = emb
	detectorMu.Unlock()
	getLogger().Info("master_face_set")
	return nil
}

// AnalyzeImage 单帧识别，返回 8 项监考结果
func AnalyzeImage(img image.Image) DetectionResult {
	result := DetectionResult{Codes: []int{}}
	if !IsDetectorReady() {
		return result
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	yoloDet := runYolo(img, width, height)
	result.Flags.FindBookPC = yoloDet.findBook
	result.Flags.FindPhonePC = yoloDet.findPhone
	personCount := len(yoloDet.personBoxes)

	if personCount < 1 {
		result.Flags.NobodyPC = true
	}
	if personCount > 1 {
		result.Flags.MultiplePersonPC = true
	}

	// 单人：用五关键点相对四边 0.2 内框判越界（不用全身框，避免举手挡脸等误报）
	// 越界只报 rangeTest，不跑低头/转头/换人；未检出脸则不做姿态与换人
	if personCount == 1 {
		face, ok := detectPrimaryFace(img)
		if ok {
			if isFaceOutsideFence(face, width, height) {
				result.Flags.RangeTestPC = true
			} else {
				runPortraitChecks(img, face, &result.Flags)
			}
		}
	}

	result.Codes = flagsToCodes(result.Flags)
	return result
}

func flagsToCodes(flags DetectionFlags) []int {
	codes := []int{}
	if flags.NobodyPC {
		codes = append(codes, codeNobody)
	}
	if flags.MultiplePersonPC {
		codes = append(codes, codeMultiPerson)
	}
	if flags.FindPhonePC {
		codes = append(codes, codePhone)
	}
	if flags.FindBookPC {
		codes = append(codes, codeBook)
	}
	if flags.ChangePersonPC {
		codes = append(codes, codeChangePerson)
	}
	if flags.LowerHeadPC {
		codes = append(codes, codeLowerHead)
	}
	if flags.TurnheadPC {
		codes = append(codes, codeTurnHead)
	}
	if flags.RangeTestPC {
		codes = append(codes, codeRange)
	}
	return codes
}

type yoloDetection struct {
	personBoxes []yoloBox
	findBook    bool
	findPhone   bool
}

func runYolo(img image.Image, width, height int) yoloDetection {
	var det yoloDetection
	tensorData, scale, padX, padY := letterboxToTensor(img, width, height, yoloInputSize, true)
	inputShape := ort.NewShape(1, 3, yoloInputSize, yoloInputSize)
	inputTensor, err := ort.NewTensor(inputShape, tensorData)
	if err != nil {
		getLogger().Warn("yolo_input_tensor_failed", "error", err.Error())
		return det
	}
	defer inputTensor.Destroy()

	outputShape := ort.NewShape(1, 84, 8400)
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		getLogger().Warn("yolo_output_tensor_failed", "error", err.Error())
		return det
	}
	defer outputTensor.Destroy()

	if err := yoloSession.Run([]ort.Value{inputTensor}, []ort.Value{outputTensor}); err != nil {
		getLogger().Warn("yolo_run_failed", "error", err.Error())
		return det
	}

	boxes := parseYoloOutput(outputTensor.GetData(), 8400, scale, padX, padY, width, height)
	boxes = nms(boxes, yoloIouThreshold)
	for _, box := range boxes {
		if box.score < yoloScoreThreshold {
			continue
		}
		if box.classID < 0 || box.classID >= len(cocoLabels) {
			continue
		}
		switch cocoLabels[box.classID] {
		case "person":
			det.personBoxes = append(det.personBoxes, box)
		case "book":
			det.findBook = true
		case "cell phone", "remote":
			det.findPhone = true
		}
	}
	return det
}

// isFaceOutsideFence 五关键点（左右眼、鼻、左右嘴）任一点落在四边 0.2 内框外则视为越界
func isFaceOutsideFence(face faceLandmarks, width, height int) bool {
	fenceX := float32(width) * fenceMarginLeft
	fenceY := float32(height) * fenceMarginTop
	fenceRight := float32(width) * (1 - fenceMarginRight)
	fenceBottom := float32(height) * (1 - fenceMarginBottom)
	keypoints := [][2]float32{
		{face.leftEyeX, face.leftEyeY},
		{face.rightEyeX, face.rightEyeY},
		{face.noseX, face.noseY},
		{face.leftMouthX, face.leftMouthY},
		{face.rightMouthX, face.rightMouthY},
	}
	for _, pt := range keypoints {
		if pt[0] < fenceX || pt[0] > fenceRight || pt[1] < fenceY || pt[1] > fenceBottom {
			return true
		}
	}
	return false
}

func letterboxToTensor(img image.Image, width, height, inputSize int, normalize bool) ([]float32, float32, float32, float32) {
	scale := math.Min(float64(inputSize)/float64(width), float64(inputSize)/float64(height))
	newW := int(float64(width) * scale)
	newH := int(float64(height) * scale)
	padX := float32(inputSize-newW) / 2
	padY := float32(inputSize-newH) / 2
	minX := img.Bounds().Min.X
	minY := img.Bounds().Min.Y

	data := make([]float32, 3*inputSize*inputSize)
	for y := 0; y < inputSize; y++ {
		for x := 0; x < inputSize; x++ {
			srcX := int((float32(x) - padX) / float32(scale))
			srcY := int((float32(y) - padY) / float32(scale))
			idx := y*inputSize + x
			if srcX < 0 || srcY < 0 || srcX >= width || srcY >= height {
				continue
			}
			r, g, b, _ := img.At(srcX+minX, srcY+minY).RGBA()
			if normalize {
				data[idx] = float32(r>>8) / 255.0
				data[inputSize*inputSize+idx] = float32(g>>8) / 255.0
				data[2*inputSize*inputSize+idx] = float32(b>>8) / 255.0
			} else {
				data[idx] = float32(r >> 8)
				data[inputSize*inputSize+idx] = float32(g >> 8)
				data[2*inputSize*inputSize+idx] = float32(b >> 8)
			}
		}
	}
	return data, float32(scale), padX, padY
}

func parseYoloOutput(data []float32, numPred int, scale, padX, padY float32, width, height int) []yoloBox {
	boxes := make([]yoloBox, 0, numPred)
	for i := 0; i < numPred; i++ {
		bestScore := float32(0)
		bestClass := 0
		for c := 0; c < 80; c++ {
			score := data[(4+c)*numPred+i]
			if score > bestScore {
				bestScore = score
				bestClass = c
			}
		}
		if bestScore < yoloScoreThreshold {
			continue
		}
		cx := data[0*numPred+i]
		cy := data[1*numPred+i]
		w := data[2*numPred+i]
		h := data[3*numPred+i]

		x1 := (cx - w/2 - padX) / scale
		y1 := (cy - h/2 - padY) / scale
		x2 := (cx + w/2 - padX) / scale
		y2 := (cy + h/2 - padY) / scale

		boxes = append(boxes, yoloBox{
			x1:      clamp32(x1, 0, float32(width)),
			y1:      clamp32(y1, 0, float32(height)),
			x2:      clamp32(x2, 0, float32(width)),
			y2:      clamp32(y2, 0, float32(height)),
			classID: bestClass,
			score:   bestScore,
		})
	}
	return boxes
}

func clamp32(v, minV, maxV float32) float32 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func nms(boxes []yoloBox, iouThreshold float32) []yoloBox {
	sort.Slice(boxes, func(i, j int) bool {
		return boxes[i].score > boxes[j].score
	})
	kept := make([]yoloBox, 0, len(boxes))
	suppressed := make([]bool, len(boxes))
	for i := 0; i < len(boxes); i++ {
		if suppressed[i] {
			continue
		}
		kept = append(kept, boxes[i])
		for j := i + 1; j < len(boxes); j++ {
			if suppressed[j] {
				continue
			}
			if boxes[i].classID != boxes[j].classID {
				continue
			}
			if iou(boxes[i], boxes[j]) > iouThreshold {
				suppressed[j] = true
			}
		}
	}
	return kept
}

func iou(a, b yoloBox) float32 {
	ix1 := math.Max(float64(a.x1), float64(b.x1))
	iy1 := math.Max(float64(a.y1), float64(b.y1))
	ix2 := math.Min(float64(a.x2), float64(b.x2))
	iy2 := math.Min(float64(a.y2), float64(b.y2))
	interW := math.Max(0, ix2-ix1)
	interH := math.Max(0, iy2-iy1)
	inter := float32(interW * interH)
	areaA := (a.x2 - a.x1) * (a.y2 - a.y1)
	areaB := (b.x2 - b.x1) * (b.y2 - b.y1)
	union := areaA + areaB - inter
	if union <= 0 {
		return 0
	}
	return inter / union
}

func detectPrimaryFace(img image.Image) (faceLandmarks, bool) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	tensorData, scale, padX, padY := letterboxToTensor(img, width, height, faceInputSize, false)

	inputShape := ort.NewShape(1, 3, faceInputSize, faceInputSize)
	inputTensor, err := ort.NewTensor(inputShape, tensorData)
	if err != nil {
		return faceLandmarks{}, false
	}
	defer inputTensor.Destroy()

	outputs := make([]ort.Value, 12)
	shapes := []ort.Shape{
		ort.NewShape(1, 6400, 1), ort.NewShape(1, 1600, 1), ort.NewShape(1, 400, 1),
		ort.NewShape(1, 6400, 1), ort.NewShape(1, 1600, 1), ort.NewShape(1, 400, 1),
		ort.NewShape(1, 6400, 4), ort.NewShape(1, 1600, 4), ort.NewShape(1, 400, 4),
		ort.NewShape(1, 6400, 10), ort.NewShape(1, 1600, 10), ort.NewShape(1, 400, 10),
	}
	for i, shape := range shapes {
		t, tensorErr := ort.NewEmptyTensor[float32](shape)
		if tensorErr != nil {
			destroyTensors(outputs[:i])
			return faceLandmarks{}, false
		}
		outputs[i] = t
	}
	defer destroyTensors(outputs)

	if err := faceDetectSess.Run([]ort.Value{inputTensor}, outputs); err != nil {
		getLogger().Warn("face_detect_run_failed", "error", err.Error())
		return faceLandmarks{}, false
	}

	tensors := make([][]float32, 12)
	for i, out := range outputs {
		tensor, ok := out.(*ort.Tensor[float32])
		if !ok {
			return faceLandmarks{}, false
		}
		tensors[i] = tensor.GetData()
	}

	faces := decodeYuNetFaces(tensors, []int{8, 16, 32})
	faces = nmsFaces(faces, faceNmsThreshold)
	if len(faces) == 0 {
		return faceLandmarks{}, false
	}

	best := faces[0]
	for i := 1; i < len(faces); i++ {
		if faces[i].score > best.score {
			best = faces[i]
		}
	}

	invScale := 1 / scale
	mapPoint := func(px, py float32) (float32, float32) {
		return (px - padX) * invScale, (py - padY) * invScale
	}
	x, y := mapPoint(best.x, best.y)
	reX, reY := mapPoint(best.rightEyeX, best.rightEyeY)
	leX, leY := mapPoint(best.leftEyeX, best.leftEyeY)
	nX, nY := mapPoint(best.noseX, best.noseY)
	rmX, rmY := mapPoint(best.rightMouthX, best.rightMouthY)
	lmX, lmY := mapPoint(best.leftMouthX, best.leftMouthY)

	return faceLandmarks{
		x: x, y: y, w: best.w * invScale, h: best.h * invScale,
		rightEyeX: reX, rightEyeY: reY,
		leftEyeX: leX, leftEyeY: leY,
		noseX: nX, noseY: nY,
		rightMouthX: rmX, rightMouthY: rmY,
		leftMouthX: lmX, leftMouthY: lmY,
		score: best.score,
	}, true
}

func destroyTensors(values []ort.Value) {
	for _, v := range values {
		if v != nil {
			_ = v.Destroy()
		}
	}
}

func decodeYuNetFaces(tensors [][]float32, strides []int) []faceLandmarks {
	faces := []faceLandmarks{}
	for scaleIdx, stride := range strides {
		cls := tensors[scaleIdx]
		obj := tensors[scaleIdx+3]
		bbox := tensors[scaleIdx+6]
		kps := tensors[scaleIdx+9]
		featSize := faceInputSize / stride
		count := featSize * featSize
		for idx := 0; idx < count; idx++ {
			score := cls[idx] * obj[idx]
			if score < faceScoreThreshold {
				continue
			}
			row := idx / featSize
			col := idx % featSize
			cx := (float32(col) + 0.5) * float32(stride)
			cy := (float32(row) + 0.5) * float32(stride)
			boff := idx * 4
			x1 := cx - bbox[boff]*float32(stride)
			y1 := cy - bbox[boff+1]*float32(stride)
			x2 := cx + bbox[boff+2]*float32(stride)
			y2 := cy + bbox[boff+3]*float32(stride)
			koff := idx * 10
			face := faceLandmarks{
				x: x1, y: y1, w: x2 - x1, h: y2 - y1,
				rightEyeX:   cx + kps[koff]*float32(stride),
				rightEyeY:   cy + kps[koff+1]*float32(stride),
				leftEyeX:    cx + kps[koff+2]*float32(stride),
				leftEyeY:    cy + kps[koff+3]*float32(stride),
				noseX:       cx + kps[koff+4]*float32(stride),
				noseY:       cy + kps[koff+5]*float32(stride),
				rightMouthX: cx + kps[koff+6]*float32(stride),
				rightMouthY: cy + kps[koff+7]*float32(stride),
				leftMouthX:  cx + kps[koff+8]*float32(stride),
				leftMouthY:  cy + kps[koff+9]*float32(stride),
				score:       score,
			}
			faces = append(faces, face)
		}
	}
	sort.Slice(faces, func(i, j int) bool { return faces[i].score > faces[j].score })
	if len(faces) > 100 {
		faces = faces[:100]
	}
	return faces
}

func nmsFaces(faces []faceLandmarks, threshold float32) []faceLandmarks {
	kept := make([]faceLandmarks, 0, len(faces))
	suppressed := make([]bool, len(faces))
	for i := 0; i < len(faces); i++ {
		if suppressed[i] {
			continue
		}
		kept = append(kept, faces[i])
		for j := i + 1; j < len(faces); j++ {
			if suppressed[j] {
				continue
			}
			if faceIoU(faces[i], faces[j]) > threshold {
				suppressed[j] = true
			}
		}
	}
	return kept
}

func faceIoU(a, b faceLandmarks) float32 {
	ax2 := a.x + a.w
	ay2 := a.y + a.h
	bx2 := b.x + b.w
	by2 := b.y + b.h
	ix1 := math.Max(float64(a.x), float64(b.x))
	iy1 := math.Max(float64(a.y), float64(b.y))
	ix2 := math.Min(float64(ax2), float64(bx2))
	iy2 := math.Min(float64(ay2), float64(by2))
	inter := float32(math.Max(0, ix2-ix1) * math.Max(0, iy2-iy1))
	union := a.w*a.h + b.w*b.h - inter
	if union <= 0 {
		return 0
	}
	return inter / union
}

// runPortraitChecks 低头 / 转头 / 换人（越界已在 AnalyzeImage 用五关键点门禁处理）
// 换人仅在无低头、无转头且人脸质量达标时检测；无人/多人/越界本就不会进入本函数
func runPortraitChecks(img image.Image, face faceLandmarks, flags *DetectionFlags) {
	pitch, yaw := estimateHeadPose(face)
	if pitch > lowerHeadPitchThreshold {
		flags.LowerHeadPC = true
	}
	if yaw < -turnHeadYawThreshold || yaw > turnHeadYawThreshold {
		flags.TurnheadPC = true
	}

	yawTrigger := yaw < -turnHeadYawThreshold || yaw > turnHeadYawThreshold
	getLogger().Info("head_pose",
		"pitch", pitch,
		"yaw", yaw,
		"lower_head_threshold", lowerHeadPitchThreshold,
		"lower_head", flags.LowerHeadPC,
		"turn_head", flags.TurnheadPC,
		"yaw_trigger", yawTrigger,
	)

	// 低头或转头时不做换人比对（侧脸/俯仰会导致 embedding 不稳定）
	if flags.LowerHeadPC || flags.TurnheadPC {
		return
	}

	if face.score < faceChangeQualityThreshold {
		getLogger().Info("change_person_skipped_low_quality", "face_score", face.score, "threshold", faceChangeQualityThreshold)
		return
	}

	detectorMu.RLock()
	master := masterEmbedding
	detectorMu.RUnlock()
	if len(master) == 0 {
		return
	}
	emb, err := extractFaceEmbedding(img, face)
	if err != nil {
		return
	}
	if cosineSimilarity(master, emb) < faceMatchThreshold {
		flags.ChangePersonPC = true
	}
}

func estimateHeadPose(face faceLandmarks) (pitch, yaw float32) {
	eyeMidX := (face.leftEyeX + face.rightEyeX) / 2
	eyeMidY := (face.leftEyeY + face.rightEyeY) / 2
	eyeDistance := float32(math.Abs(float64(face.rightEyeX - face.leftEyeX)))
	if eyeDistance < 1 {
		return 0, 0
	}

	// 对齐 calculate_yaw：水平眼距归一化
	yaw = (face.noseX - eyeMidX) / eyeDistance * 90

	// 对齐 calculate_pitch：嘴-眼垂直距 / 水平眼距
	mouthMidY := (face.rightMouthY + face.leftMouthY) / 2
	normalizedVertical := (mouthMidY - eyeMidY) / eyeDistance
	pitch = (normalizedVertical - 1.1) * 45 * 2
	return pitch, yaw
}

func extractFaceEmbedding(img image.Image, face faceLandmarks) ([]float32, error) {
	crop := alignFaceCrop(img, face)
	tensorData := imageToNCHWNormalized(crop, 112, 112)
	inputShape := ort.NewShape(1, 3, 112, 112)
	inputTensor, err := ort.NewTensor(inputShape, tensorData)
	if err != nil {
		return nil, err
	}
	defer inputTensor.Destroy()

	outputShape := ort.NewShape(1, 512)
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, err
	}
	defer outputTensor.Destroy()

	if err := faceRecSess.Run([]ort.Value{inputTensor}, []ort.Value{outputTensor}); err != nil {
		return nil, err
	}
	emb := outputTensor.GetData()
	normalized := make([]float32, len(emb))
	copy(normalized, emb)
	l2Normalize(normalized)
	return normalized, nil
}

func alignFaceCrop(img image.Image, face faceLandmarks) image.Image {
	size := 112
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	srcBounds := img.Bounds()
	scale := float64(size) / math.Max(float64(face.w), float64(face.h))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			srcX := int(float64(x)/scale + float64(face.x))
			srcY := int(float64(y)/scale + float64(face.y))
			if srcX >= srcBounds.Min.X && srcX < srcBounds.Max.X &&
				srcY >= srcBounds.Min.Y && srcY < srcBounds.Max.Y {
				dst.Set(x, y, img.At(srcX, srcY))
			}
		}
	}
	return dst
}

func imageToNCHWNormalized(img image.Image, width, height int) []float32 {
	data := make([]float32, 3*width*height)
	minX := img.Bounds().Min.X
	minY := img.Bounds().Min.Y
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x+minX, y+minY).RGBA()
			idx := y*width + x
			data[idx] = (float32(r>>8) - 127.5) / 127.5
			data[width*height+idx] = (float32(g>>8) - 127.5) / 127.5
			data[2*width*height+idx] = (float32(b>>8) - 127.5) / 127.5
		}
	}
	return data
}

func l2Normalize(vec []float32) {
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	if sum == 0 {
		return
	}
	norm := float32(math.Sqrt(sum))
	for i := range vec {
		vec[i] /= norm
	}
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return float32(dot)
}
