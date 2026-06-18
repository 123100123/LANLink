package ws

import transferpkg "github.com/123100123/lanlink/internal/transfer"

var transferManager = transferpkg.NewManager(func() string { return "received" })
