// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package input

import (
	"crypto/rand"
	"math/big"
)

var letterRunes = []rune("1234567890abcdefghijklmnopqrstuvwxyz")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterRunes))))
		if err != nil {
			panic("failed to generate random number")
		}
		b[i] = letterRunes[num.Int64()]
	}
	return string(b)
}

func getTag(namespace, pod, container, containerID string) string {
	return "kubernetes.var.log.containers." + pod + "_" + namespace + "_" + container + "-" + containerID + ".log"
}

func generatePodName(pod string) string {
	return pod + "-" + randStringRunes(10) + "-" + randStringRunes(5)
}

func generateContainerID() string {
	return randStringRunes(64)
}
