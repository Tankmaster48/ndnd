package main

import "github.com/named-data/ndnd/repo"

// Serves as the entry point for the Named-Data Networking repository command-line interface, executing its configured command set.
func main() {
	repo.CmdRepo.Execute()
}
