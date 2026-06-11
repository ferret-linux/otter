package ui

import "strings"

// TrimImageRef strips the registry and repository prefix from a container
// image reference, returning only the image name and tag.
// Everything up to and including the last '/' is removed.
// If no '/' is present, the ref is returned unchanged.
//
// Examples:
//
//	ghcr.io/ferret-linux/alpine-otr:latest -> alpine-otr:latest
//	docker.io/library/ubuntu:latest        -> ubuntu:latest
//	quay.io/fedora/fedora:44               -> fedora:44
//	alpine:latest                          -> alpine:latest
//	ubi10.1:11-init                        -> ubi10.1:11-init
func TrimImageRef(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}
