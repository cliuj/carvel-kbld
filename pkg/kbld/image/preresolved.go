// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"github.com/k14s/kbld/pkg/kbld/config"
)

type PreresolvedImage struct {
	url   string
	metas []config.ImageMeta
}

type PreresolvedImageSourceURL struct {
	Type string // always set to 'preresolved'
	URL  string
	Tag  string `json:",omitempty" yaml:",omitempty"`
}

func (PreresolvedImageSourceURL) meta() {}

func NewPreresolvedImage(url string, metas []config.ImageMeta) PreresolvedImage {
	return PreresolvedImage{url, metas}
}

func (i PreresolvedImage) URL() (string, []Meta, error) {
	if len(i.metas) > 0 {
		var givenMetas []Meta
		for _, m := range i.metas {
			givenMetas = append(givenMetas, PreresolvedImageSourceURL{Type: "preresolved", URL: m.URL, Tag: m.Tag})
		}
		return i.url, givenMetas, nil
	}
	return i.url, []Meta{PreresolvedImageSourceURL{Type: "preresolved", URL: i.url}}, nil
}
