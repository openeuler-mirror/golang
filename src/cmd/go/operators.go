// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math"
)

func matmul(lhs []float32, rhs []float32, m int, k int, n int, out []float32) {
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			out[i*n+j] = 0
			for p := 0; p < k; p++ {
				out[i*n+j] += lhs[i*k+p] * rhs[p*n+j]
			}
		}
	}
}

func add(lhs []float32, rhs []float32, length int, out []float32) {
	for i := 0; i < length; i++ {
		out[i] = lhs[i] + rhs[i]
	}
}

func sub(lhs []float32, rhs []float32, length int, out []float32) {
	for i := 0; i < length; i++ {
		out[i] = lhs[i] - rhs[i]
	}
}

func sigmoid(in []float32, length int, out []float32) {
	for i := 0; i < length; i++ {
		out[i] = float32(1 / (1 + math.Exp(float64(-in[i]))))
	}
}

func relu(data []float32, length int, out []float32) {
	for i := 0; i < length; i++ {
		if data[i] < 0 {
			out[i] = 0
		} else {
			out[i] = data[i]
		}
	}
}

func lineConcat(in []float32, inSize int, out []float32, outSize int) {
	for i := 0; i < inSize; i++ {
		out[outSize+i] = in[i]
	}
}

func oneHotEncoder(in string, cats []string, out []float32, outSize int) {
	for i := 0; i < outSize; i++ {
		if i < outSize && cats[i] == in {
			out[i] = 1
		} else {
			out[i] = 0
		}
	}
}

func imputer(in []int64, size int, out []float32) {
	for i := 0; i < size; i++ {
		out[i] = float32(in[i])
	}
}

func scaler(in []float32, offset []float32, scale []float32, size int, out []float32) {
	for i := 0; i < size; i++ {
		out[i] = (in[i] - offset[i]) * scale[i]
	}
}

func argmax(in []float32, inSize int) int {
	var outIdx = 0
	for i := 0; i < inSize; i++ {
		if in[i] > in[outIdx] {
			outIdx = i
		}
	}
	return outIdx
}
