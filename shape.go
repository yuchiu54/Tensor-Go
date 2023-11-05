package main

// This source file contains functions related to manipulating the shape of a tensor.

import (
	"strconv" // <-- used to convert strings to ints
	"strings"
	"sync"
)

// The Partial function is used to retrieve a section out of a Tensor using Python-like slice notation.
// It accepts a Tensor and a string, then returns a pointer to a new tensor.
// Example:
// A := Range_Tensor([]int{3, 4, 9, 2})
// A_Partial := Partial(A, "0:2, 2:, :3, :")
func Partial(A *Tensor, slice string) *Tensor {
	// Remove spaces and split the slice string by commas to handle each dimension separately.
	slice = strings.ReplaceAll(slice, " ", "")
	split := strings.Split(slice, ",")
	if len(split) != len(A.shape) {
		panic("String slice arg must have the same number of dimensions as the tensor")
	}

	// Initialize slices to store the shape of the partial tensor and the start/end indices for each dimension.
	partialShape := make([]int, len(A.shape))
	partialIndices := make([][]int, len(A.shape))

	// Iterate through each dimension of the tensor to parse the slice string and compute the shape and indices of the partial tensor.
	for i, s := range split {
		start, end := 0, A.shape[i] // By default, use the entire dimension.
		if s != ":" {
			parts := strings.Split(s, ":")

			if parts[0] != "" { // If there is a start value, update start.
				start, _ = strconv.Atoi(parts[0])
			}
			if parts[1] != "" { // If there is an end value, update end.
				end, _ = strconv.Atoi(parts[1])
			}
		}
		partialShape[i] = end - start
		partialIndices[i] = []int{start, end}
	}

	// Create a new tensor to store the partial data with the computed shape.
	partialTensor := Zero_Tensor(partialShape)

	// Initialize a slice to store the current multi-dimensional index being processed.
	tempIndex := make([]int, len(partialShape))

	// Define a recursive function to fill the partial tensor.
	// The function takes the current dimension as a parameter.
	var fillPartialTensor func(int)
	fillPartialTensor = func(dim int) {
		if dim == len(partialShape) { // <--- This base case is reached for every element in the partial tensor.

			// Calculate the source index in the original tensor.
			srcIndex := make([]int, len(partialShape))
			for i, indices := range partialIndices {
				srcIndex[i] = tempIndex[i] + indices[0]
			}

			// Convert the multi-dimensional indices to flattened indices and use them to copy the data.
			srcFlattenedIndex := Index(srcIndex, A.shape)
			dstFlattenedIndex := Index(tempIndex, partialTensor.shape)
			partialTensor.data[dstFlattenedIndex] = A.data[srcFlattenedIndex]

			return
		}

		// Recursively process each index in the current dimension.
		for i := 0; i < partialShape[dim]; i++ {
			tempIndex[dim] = i
			fillPartialTensor(dim + 1)
		}
	}

	// Start the recursive process from the first dimension.
	fillPartialTensor(0)

	// Return the filled partial tensor.
	return partialTensor
}

// Reshape()  takes a tensors and a new shape for that tensors, and returns a pointer to a
// new tensors that has the same data as the original tensor, but with the new shape. Reshape
// can be done in this way becauase data for Tensors in stored contigously in memory.
func (A *Tensor) Reshape(shape []int) *Tensor {

	numElements := 1
	for _, v := range shape { // find num elements of shape param
		numElements *= v
	}
	if numElements != len(A.data) {
		panic("Cannot reshape tensor to shape with different number of elements")
	}
	// Create a new tensor to store the reshaped data with the shape param
	reshapedTensor := Zero_Tensor(shape)
	for i := 0; i < len(A.data); i++ {
		reshapedTensor.data[i] = A.data[i] // copy data from A to reshapedTensor
	}
	return reshapedTensor
}

// Transpose returns a new tensor with the axes transposed according to the given specification
// This function is modeled after the NumPy transpose function. It accepts a tensor and an array
// of integers specifying the new order of the axes. For example, if the tensor has shape [2, 3, 4]
// and the axes array is [2, 0, 1], then the resulting tensor will have shape [4, 2, 3].
func (A *Tensor) Transpose(axes []int) *Tensor {

	// Check for invalid axes
	if len(axes) != len(A.shape) {
		panic("The number of axes does not match the number of dimensions of the tensor.")
	}

	// Check for duplicate or out-of-range axes
	seen := make(map[int]bool) // map is like dict in python
	for _, axis := range axes {
		if axis < 0 || axis >= len(A.shape) || seen[axis] {
			panic("Invalid axis specification for transpose.")
		}
		seen[axis] = true
	}

	// Determine the new shape from the reordering in axes
	newShape := make([]int, len(A.shape))
	for i, axis := range axes {
		newShape[i] = A.shape[axis]
	}

	// Allocate the new tensor
	newData := make([]float64, len(A.data))
	B := &Tensor{shape: newShape, data: newData} // <-- B is a pointer to a new tensor

	// Reindex and copy data
	for i := range A.data {
		// Get the multi-dimensional indices for the current element
		originalIndices := UnravelIndex(i, A.shape)

		// Reorder the indices according to the axes array for transpose
		newIndices := make([]int, len(originalIndices))
		for j, axis := range axes {
			newIndices[j] = originalIndices[axis]
		}

		// Convert the reordered multi-dimensional indices back to a flat index
		newIndex := Index(newIndices, newShape)

		// Assign the i'th value of original tensor to the newIndex'th val of new tensor
		B.data[newIndex] = A.data[i]
	}

	return B
}

// The idea behind this algorithm stems from an understanding of how Tensor data is stored in memory.
// Tensors of n dimmension are stored contiguously in memory as a 1D array. The multi-dimensionality
// of the tensor is simulated by indexing the 1D array using a strided index. This means that if you
// are atttemping to index a 5D tensor of shape [3, 3, 3, 3, 3], and you want to move one element up
// the last dimmension, then you must 'stride' over all elements of the 4th dimmension stored in the
// contigous memory to get there. This task is handled by the Index() and Retrieve() functions.
// ---------------------------------------------------------------------------------------------------
// This way of storing data in in memory introduces complexity when concatenating tenosrs along an axis.
// When the axis of concatenation is the 0'th axis, the algorithm is simple. No striding is required, and
// the contigous data from one tensor can just be appended to the other.
// ---------------------------------------------------------------------------------------------------
// However, when the axis of concatenation is not the 0'th axis, the algorithm becomes more complex
// due to the striding. This algorithm handles this complexity by simplifying the cases where the axis
// of concatenation is not zero by first tranpsosing the tensors such that the axis of concatenation
// is the 0'th axis. They can then simply be appended together contiguously and transposed back to the
// original ordering of dimmensions.
func (A *Tensor) Concat(B *Tensor, axis_cat int) *Tensor {

	// Ensure that the number of dimensions of the tensors are the same
	if len(A.shape) != len(B.shape) {
		panic("The number of dimensions of the tensors must be the same.")
	}

	// Check that axis_cat is within the valid range
	if axis_cat < 0 || axis_cat >= len(A.shape) {
		panic("axis_cat is out of bounds for the shape of the tensors.")
	}

	// Ensure that the shape of the tensors are the same except for the axis of concatenation
	for i := 0; i < len(A.shape); i++ {
		if i != axis_cat && A.shape[i] != B.shape[i] {
			panic("The shapes of the tensors must be the same except for the axis of concatenation.")
		}
	}

	// Define the concatTensor variable outside of the if-else blocks to use it in the entire function scope
	var concatTensor *Tensor

	// conditional to check if the axis of concat is 0 or not
	if axis_cat == 0 {

		// Determine the shape of the concatenated tensor
		concatShape := make([]int, len(A.shape))
		for i := 0; i < len(A.shape); i++ {
			if i == axis_cat {
				concatShape[i] = A.shape[i] + B.shape[i] // <--- concatenation extends this dimension
			} else {
				concatShape[i] = A.shape[i]
			}
		}

		// concatenate data contiguously into new slice
		concatData := append(A.data, B.data...)

		// create new tensor to store concatenated data for return
		concatTensor = &Tensor{shape: concatShape, data: concatData}
	} else if axis_cat != 0 {

		// determine the reordering of the axes for transpose to make axis_cat the 0'th axis the slice
		// will be a permutation of the numbers 0 through len(A.shape) - 1 with axis cat and 0 swapped
		axes_reordering := make([]int, len(A.shape))

		// set axis cat to 0'th axis
		axes_reordering[0] = axis_cat

		// Now fill in the rest of the axes.
		for i, count := 1, 0; count < len(A.shape); count++ {
			// exclude axis_cat from the reordering, its already at 0
			if count != axis_cat {
				axes_reordering[i] = count
				i++
			}
		}

		// transpose A and B to make axis_cat the 0'th axis
		A_T := A.Transpose(axes_reordering)
		B_T := B.Transpose(axes_reordering)

		// concatenate data contiguously into new slice
		concatData_Transposed := append(A_T.data, B_T.data...)

		// We now have a slice of contigous data that is the concatenation of A_T and B_T, in order to use
		// this data to create a new tensor, we must first determine the shape of the new tensor in this
		// Trasnposed form. This can be done by copying A_T.shape and adding B_T.shape[0] to it.
		concatShape_Transposed := make([]int, len(A_T.shape))
		for i := 0; i < len(A_T.shape); i++ {
			if i == 0 {
				concatShape_Transposed[i] = A_T.shape[i] + B_T.shape[i]
			} else {
				concatShape_Transposed[i] = A_T.shape[i]
			}
		}

		// create new tensor to store the transposed concatenated data
		concatTensor_Transposed := &Tensor{shape: concatShape_Transposed, data: concatData_Transposed}

		// transpose the concatenated tensor back to the original ordering of axes. Because we only swapped
		// two axes, we can just reuse the same axe_reordering array from the originbal transpose.
		concatTensor = concatTensor_Transposed.Transpose(axes_reordering)
	}

	return concatTensor
}

// The Extend() method is used to add a new dimmension to the tensor. The new dimmension each element
// across the new dimmension contains a state of the pre extended tensor with all other dimmension elements
// copied into it. The new dimmension is added to the end of the shape of the tensor. The Extend() method
// returns a pointer to a new tensor with the extended shape and zeroed data.
func (A *Tensor) Extend(num_elements int) *Tensor {
	// Check that the number of elements is valid
	if num_elements < 1 {
		panic("The number of elements must be positive.")
	}

	// Create a new shape with the additional dimension
	newShape := make([]int, len(A.shape)+1)
	copy(newShape, A.shape)               // <--- Copy the original shape
	newShape[len(A.shape)] = num_elements // <---  Add the new dimension at the end

	// Create a new tensor with the extended shape and zeroed data
	extendedTensor := Zero_Tensor(newShape)

	// WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Recursive function to fill the extended tensor
	var fillExtendedTensor func(int, []int)
	fillExtendedTensor = func(dim int, tempIndex []int) {
		defer wg.Done()

		if dim == len(A.shape) { // If we reached the last original dimension
			srcFlattenedIndex := Index(tempIndex[:len(tempIndex)-1], A.shape)
			for i := 0; i < num_elements; i++ {
				tempIndex[len(tempIndex)-1] = i
				dstFlattenedIndex := Index(tempIndex, newShape)
				extendedTensor.data[dstFlattenedIndex] = A.data[srcFlattenedIndex]
			}
			return
		}

		// Recursively process each index in the current dimension
		for i := 0; i < A.shape[dim]; i++ {
			// Make a copy of tempIndex for concurrent use
			newTempIndex := make([]int, len(tempIndex))
			copy(newTempIndex, tempIndex)
			newTempIndex[dim] = i

			wg.Add(1)
			// Start a new goroutine for each recursive call
			go fillExtendedTensor(dim+1, newTempIndex)
		}
	}

	// Add to the WaitGroup and start the first call to fillExtendedTensor
	wg.Add(1)
	go fillExtendedTensor(0, make([]int, len(newShape)))

	// Wait for all goroutines to finish
	wg.Wait()

	// Return the filled extended tensor
	return extendedTensor
}
