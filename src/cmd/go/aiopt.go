package main

import (
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

const (
	mOptionSize         = 11
	mModeSize           = 6
	sha256sumOutputSize = 64
	catsStringsRow      = 35
	catsStrings1Row     = 1000
	offsetRow           = 6
	scaleRow            = 6
	unityRow            = 1
	coefficientRow      = 1356
	coefficientCol      = 10
	coefficient1Row     = 10
	coefficient1Col     = 1
	interceptsRow       = 10
	intercepts1Row      = 1
)

var (
	catsStrings  [catsStringsRow]string
	catsStrings1 [catsStrings1Row]string
	offset       [offsetRow]float32
	scale        [scaleRow]float32
	unity        [unityRow]float32
	coefficient  [coefficientRow][coefficientCol]float32
	coefficient1 [coefficient1Row][coefficient1Col]float32
	intercepts   [interceptsRow]float32
	intercepts1  [intercepts1Row]float32
)

func fillNode(file *os.File) {
	for i := 0; i < catsStringsRow; i++ {
		var str [sha256sumOutputSize]byte
		for j := 0; j < sha256sumOutputSize; j++ {
			str[j] = readByteFromFile(file)
		}
		catsStrings[i] = string(str[:])
	}
	for i := 0; i < catsStrings1Row; i++ {
		var str [sha256sumOutputSize]byte
		for j := 0; j < sha256sumOutputSize; j++ {
			str[j] = readByteFromFile(file)
		}
		catsStrings1[i] = string(str[:])
	}
	for i := 0; i < offsetRow; i++ {
		offset[i] = readFloatFromFile(file)
	}
	for i := 0; i < scaleRow; i++ {
		scale[i] = readFloatFromFile(file)
	}
	for i := 0; i < unityRow; i++ {
		unity[i] = readFloatFromFile(file)
	}
	for i := 0; i < coefficientRow; i++ {
		for j := 0; j < coefficientCol; j++ {
			coefficient[i][j] = readFloatFromFile(file)
		}
	}
	for i := 0; i < coefficient1Row; i++ {
		for j := 0; j < coefficient1Col; j++ {
			coefficient1[i][j] = readFloatFromFile(file)
		}
	}
	for i := 0; i < interceptsRow; i++ {
		intercepts[i] = readFloatFromFile(file)
	}
	for i := 0; i < intercepts1Row; i++ {
		intercepts1[i] = readFloatFromFile(file)
	}
}

func graphInfer(mops string, modes []int64) int {
	fileName := filepath.Join(runtime.GOROOT(), "src", "cmd", "go", "data", "onnx.fdata")
	file, err := os.Open(fileName)
	if err != nil {
		return -1
	}
	defer file.Close()
	fillNode(file)
	var inModes [mModeSize]int64
	var inOptions [mOptionSize]string
	inOptions[0] = mops

	const concatOutSize = coefficientRow
	var concatResult [concatOutSize]float32
	const encoderOutSize = catsStringsRow
	const encoderLastSize = catsStrings1Row
	concatSize := 0

	for i := 1; i < mOptionSize; i++ {
		var encoderOut [encoderOutSize]float32
		oneHotEncoder(inOptions[i], catsStrings[:], encoderOut[:], encoderOutSize)
		lineConcat(encoderOut[:], encoderOutSize, concatResult[:], concatSize)
		concatSize += encoderOutSize
	}

	var encoderOut2 [encoderLastSize]float32
	oneHotEncoder(inOptions[0], catsStrings1[:], encoderOut2[:], encoderLastSize)
	lineConcat(encoderOut2[:], encoderLastSize, concatResult[:], concatSize)
	concatSize += encoderLastSize

	var variable [mModeSize]float32
	imputer(inModes[:], mModeSize, variable[:])
	var variable1 [mModeSize]float32
	scaler(variable[:], offset[:], scale[:], mModeSize, variable1[:])

	var transformedColumn [concatOutSize + mModeSize]float32
	lineConcat(variable1[:], mModeSize, transformedColumn[:], 0)
	lineConcat(concatResult[:], concatOutSize, transformedColumn[:], mModeSize)

	const (
		m = 1
		k = coefficientRow
		n = coefficientCol
	)
	var mulResult [n]float32
	matmul(transformedColumn[:], unsafe.Slice(&coefficient[0][0], coefficientRow*coefficientCol), m, k, n, mulResult[:])

	var addResult [n]float32
	add(mulResult[:], intercepts[:], n, addResult[:])

	var nextActivations [n]float32
	relu(addResult[:], n, nextActivations[:])

	const (
		m2 = 1
		k2 = 10
		n2 = 1
	)
	var mulResult1 [n2]float32
	matmul(nextActivations[:], unsafe.Slice(&coefficient1[0][0], coefficient1Row*coefficient1Col), m2, k2, n2, mulResult1[:])

	var addResult1 [n2]float32
	add(mulResult1[:], intercepts1[:], n2, addResult1[:])

	var outActivationsResult [n2]float32
	sigmoid(addResult1[:], n2, outActivationsResult[:])

	var negativeClassProba [n2]float32
	sub(unity[:], outActivationsResult[:], n2, negativeClassProba[:])
	const probSize = n2 + n2
	var probabilities [probSize]float32
	lineConcat(negativeClassProba[:], n2, probabilities[:], 0)
	lineConcat(outActivationsResult[:], n2, probabilities[:], n2)

	argmaxOutput := argmax(probabilities[:], probSize)
	return argmaxOutput
}

func getOptimizeDecisionFromAI4C(mops string, modes []int64) int {
	hash := getSha256(mops)
	result := graphInfer(hash, modes)
	return result
}

func GetOptimizeDecision() int {
	mops := getCPUInfo()
	modes := []int64{0, 0, 0, 0, 0, 0}

	return getOptimizeDecisionFromAI4C(mops, modes)
}
