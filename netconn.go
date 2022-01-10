/*
 * Copyright (c) 2013 IBM Corp.
 *
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v1.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v10.html
 *
 * Contributors:
 *    Seth Hoenig
 *    Allan Stockdill-Mander
 *    Mike Robertson
 */

package mqtt

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

//
// This just establishes the network connection; once established the type of connection should be irrelevant
//

// openConnection opens a network connection using the protocol indicated in the URL.
// Does not carry out any MQTT specific handshakes.
func openConnection(uri *url.URL, tlsc *tls.Config, timeout time.Duration, headers http.Header, websocketOptions *WebsocketOptions, customDialer *net.Dialer) (net.Conn, error) {
	switch uri.Scheme {
	case "ws":
		conn, err := NewWebsocket(uri.String(), nil, timeout, headers, websocketOptions)
		return conn, err
	case "wss":
		conn, err := NewWebsocket(uri.String(), tlsc, timeout, headers, websocketOptions)
		return conn, err
	case "mqtt", "tcp":
		// TODO use custom dialer
		allProxy := os.Getenv("all_proxy")
		if len(allProxy) == 0 {
			conn, err := net.DialTimeout("tcp", uri.Host, timeout)
			if err != nil {
				return nil, err
			}
			return conn, nil
		}
		proxyDialer := proxy.FromEnvironment()

		conn, err := proxyDialer.Dial("tcp", uri.Host)
		if err != nil {
			return nil, err
		}
		return conn, nil
	case "unix":
		// TODO use custom dialer
		conn, err := net.DialTimeout("unix", uri.Host, timeout)
		if err != nil {
			return nil, err
		}
		return conn, nil
	case "ssl", "tls", "mqtts", "mqtt+ssl", "tcps":
		allProxy := os.Getenv("all_proxy")
		if len(allProxy) == 0 {
			if customDialer != nil {
				// we only override the timeout of the customDialer to keep existing timeout behavior
				customDialer.Timeout = timeout
				conn, err := tls.DialWithDialer(customDialer, "tcp", uri.Host, tlsc)
				if err != nil {
					return nil, err
				}
				return conn, nil
			}
			conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", uri.Host, tlsc)
			if err != nil {
				return nil, err
			}
			return conn, nil
		}
		proxyDialer := proxy.FromEnvironmentUsing(customDialer)

		conn, err := proxyDialer.Dial("tcp", uri.Host)
		if err != nil {
			return nil, err
		}

		tlsConn := tls.Client(conn, tlsc)

		err = tlsConn.Handshake()
		if err != nil {
			_ = conn.Close()
			return nil, err
		}

		return tlsConn, nil
	}
	return nil, errors.New("unknown protocol")
}
