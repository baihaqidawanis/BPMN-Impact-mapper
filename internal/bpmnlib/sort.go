package bpmnlib

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"sort"
)

func sortFindings(values []bpmn.Finding) {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i], values[j]
		return a.Scope+"\x00"+a.ElementID+"\x00"+a.Code+"\x00"+a.Message < b.Scope+"\x00"+b.ElementID+"\x00"+b.Code+"\x00"+b.Message
	})
}
