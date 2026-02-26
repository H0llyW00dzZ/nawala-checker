// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

// WithServer adds or replaces a DNS server in the checker's configuration.
// If a server with the same address already exists, it will be replaced.
//
// Deprecated: Use [Checker.SetServers] instead. SetServers provides the
// same add-or-replace behaviour and is safe to call after construction.
func WithServer(server DNSServer) Option {
	return func(c *Checker) {
		for i, s := range c.servers {
			if s.Address == server.Address {
				c.servers[i] = server
				return
			}
		}
		c.servers = append(c.servers, server)
	}
}
