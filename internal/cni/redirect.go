/*
 * Copyright (c) 2022 Cisco and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package cni Defines the redirect object and operations.
package cni

const (
	redirectModeREDIRECT      = "REDIRECT"
	defaultRedirectToPort     = "8554"
	defaultRedirectMode       = redirectModeREDIRECT
	defaultNoRedirectUID      = "1337"
	defaultNoRedirectDestAddr = "127.0.0.0/8"
)

// Redirect is the msm-cni redirect object
type Redirect struct {
	targetPort         string
	redirectMode       string
	noRedirectUID      string
	noRedirectDestAddr string
}

// NewRedirect returns a new Redirect Object constructed from a list of ports and annotations
// For now we are using some default values but that can change in the future to support
// passing values through annotations
func NewRedirect(_ *PodInfo) (*Redirect, error) {
	return &Redirect{
		targetPort:         defaultRedirectToPort,
		redirectMode:       defaultRedirectMode,
		noRedirectUID:      defaultNoRedirectUID,
		noRedirectDestAddr: defaultNoRedirectDestAddr,
	}, nil
}
