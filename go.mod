module fortio.org/progressbar

// Works with any go version really(*), including 1.24.1 or whichever is latest but doesn't require it.
// *: The HumanBytes() function taking either int or float through generics does require 1.18 or later
// but could be split into 2 functions to work with older versions if needed.
go 1.18
