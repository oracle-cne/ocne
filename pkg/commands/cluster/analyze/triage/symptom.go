// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package triage

import (
	v1 "k8s.io/api/core/v1"
)

type SymptomAppender interface {
	AddMsg(msg string)
}

type Symptom struct {
	Message string
}

// ResourceSymptomInfo has the symptoms and other info that helps in triage
type ResourceSymptomInfo[T any] struct {
	Resource *T
	Symptoms []*Symptom
}

// ------------
// Interface

func (s *Symptom) AddMsg(msg string) {
	s.Message = msg
}

// ------------
// New functions

func NewEventSymptomInfo(ev *v1.Event) *ResourceSymptomInfo[v1.Event] {
	return &ResourceSymptomInfo[v1.Event]{Resource: ev}
}

func NewPodSymptomInfo(pod *v1.Pod) *ResourceSymptomInfo[v1.Pod] {
	return &ResourceSymptomInfo[v1.Pod]{Resource: pod}
}

func NewNodeSymptomInfo(node *v1.Node) *ResourceSymptomInfo[v1.Node] {
	return &ResourceSymptomInfo[v1.Node]{Resource: node}
}

// -----------
// Methods

func (s *ResourceSymptomInfo[T]) AddSymptom(msg string) {
	s.Symptoms = append(s.Symptoms, &Symptom{Message: msg})
}
