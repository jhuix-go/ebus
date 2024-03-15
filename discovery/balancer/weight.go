/* Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package balancer

const (
	WeightKey = "weight"
)

func GetWeight(addr Address) int {
	if addr.Attributes == nil {
		return 1
	}

	weight, ok := addr.Attributes.Value(WeightKey).(int)
	if ok {
		return weight
	}

	return 1
}
