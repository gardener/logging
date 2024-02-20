// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package input

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("1234567890abcdefghijklmnopqrstuvwxyz")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
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
