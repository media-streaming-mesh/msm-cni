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

package main

// Constants used as default values cobra/viper CLI
const (
	redirectModeREDIRECT      = "REDIRECT"
	defaultRedirectMode       = redirectModeREDIRECT
	defaultRedirectToPort     = "8554"
	defaultRTSPPort           = "554"
	defaultNoRedirectUID      = "1337"
	defaultNoRedirectDestAddr = "127.0.0.0/8"
)

// Constants used in cobra/viper CLI
const (
	msmProxyPort         = "msm-proxy-port"
	proxyUID             = "proxy-uid"
	noRedirectDestAddr   = "redir-dest-addr"
	inboundInterceptMode = "inbound-intercept-mode"
)
