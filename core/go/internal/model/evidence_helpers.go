package model

func DependencyEvidence(dep DependencyEdge, message string) Evidence {
	target := dep.ToFile
	if target == "" {
		target = dep.Target
	}

	return Evidence{
		File:      dep.FromFile,
		Message:   message,
		FromLayer: dep.FromLayer,
		ToLayer:   dep.ToLayer,
		FromFile:  dep.FromFile,
		ToFile:    target,
	}
}
