package main

import (
	"bufio"
	"fmt"
	"github.com/ncw/swift"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"strings"
)

var (
	VERSION   = "0.1"
	GITCOMMIT = "HEAD"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "swift",
		Short: "swift command line interface",
	}

	flDebug := rootCmd.PersistentFlags().BoolP("debug", "D", false, "Enable debug mode")

	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Print version information and quit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Swift Go Client version %s, build %s\n", VERSION, GITCOMMIT)
			return
		},
	}
	rootCmd.AddCommand(cmdVersion)

	var cmdLs = &cobra.Command{
		Use:   "ls [container[/object]]",
		Short: "list containers or objects",
		Long:  "list containers or objects",
		Run: func(cmd *cobra.Command, args []string) {
			parseDefaultFlags(*flDebug)
			c := connect()

			if len(args) == 0 {
				containers, err := c.ContainerNamesAll(nil)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				for _, name := range containers {
					fmt.Println(name)
				}
				return
			}

			for _, arg := range args {
				if !strings.Contains(arg, "/") {
					objects, err := c.ObjectNamesAll(arg, nil)
					if err != nil {
						fmt.Println(err)
						break
					}
					for _, object := range objects {
						fmt.Println(arg + "/" + object)
					}
				} else {
					fmt.Println("Don't support object lookups yet")
				}
			}
		},
	}
	rootCmd.AddCommand(cmdLs)

	var flConcurrency *int
	var flPartSize *int64
	var flExpireAfter *int64

	var cmdPut = &cobra.Command{
		Use:   "put fromfile container[/object] OR put container[/object] < stream",
		Short: "upload (put) an object",
		Run: func(cmd *cobra.Command, args []string) {
			parseDefaultFlags(*flDebug)
			c := connect()

			switch {
			case len(args) == 0:
				fmt.Println("Need to specify at least put container[/object]")
				os.Exit(1)
			case len(args) > 2:
				fmt.Println("Too many parameters specified, at most put fromfile container[/object]")
				os.Exit(1)
			}

			var r *os.File
			var err error
			var fileOut string

			if len(args) == 1 {
				r = os.Stdin
				defer r.Close()
				fileOut = args[0]
			} else {
				r, err = os.Open(args[0])
				if err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
				defer r.Close()
				fileOut = args[1]
			}

			w, err := NewUploader(c, fileOut, *flConcurrency, *flPartSize, *flExpireAfter)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if _, err = io.Copy(w, r); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if err = w.Close(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	flConcurrency = cmdPut.Flags().IntP("concurrency", "c", 10, "Concurrency of transfers")
	flPartSize = cmdPut.Flags().Int64P("partsize", "s", 20971520, "Initial size of concurrent parts, in bytes")
	flExpireAfter = cmdPut.Flags().Int64P("expire", "e", 0, "Number of seconds to expire document after")
	rootCmd.AddCommand(cmdPut)

	var cmdGet = &cobra.Command{
		Use:   "get container[/object] tofile OR get container[/object] > tofile",
		Short: "download (get) an object",
		Run: func(cmd *cobra.Command, args []string) {
			parseDefaultFlags(*flDebug)
			c := connect()

			switch {
			case len(args) == 0:
				fmt.Println("Need to specify at least get container[/object]")
				os.Exit(1)
			case len(args) > 2:
				fmt.Println("Too many parameters specified, at most get container[/object] tofile")
				os.Exit(1)
			}

			pathParts := strings.SplitN(args[0], "/", 2)
			if len(pathParts) <= 1 {
				fmt.Println("Must specify full object path (container/object)")
				os.Exit(1)
			}

			var w io.Writer
			var bw *bufio.Writer

			switch {
			case len(args) == 1:
				w = os.Stdout
			case len(args) == 2:
				outFile, err := os.Create(args[1])
				if err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
				bw = bufio.NewWriter(outFile)
				w = bw
				defer outFile.Close()
				defer bw.Flush()
			}

			_, err := c.ObjectGet(pathParts[0], pathParts[1], w, false, nil)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

		},
	}
	rootCmd.AddCommand(cmdGet)

	var cmdDelete = &cobra.Command{
		Use:   "delete container[/object]",
		Short: "delete an object",
		Run: func(cmd *cobra.Command, args []string) {
			parseDefaultFlags(*flDebug)
			c := connect()
			if len(args) != 1 {
				fmt.Println("Must specify one object")
				os.Exit(1)
			}
			pathParts := strings.SplitN(args[0], "/", 2)
			if len(pathParts) <= 1 {
				fmt.Println("Must specify full object path (container/object)")
				os.Exit(1)
			}
			_, headers, err := c.Object(pathParts[0], pathParts[1])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			largeObjectPrefix, largeObject := headers["X-Object-Manifest"]
			err = c.ObjectDelete(pathParts[0], pathParts[1])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if largeObject {
				loParts := strings.SplitN(largeObjectPrefix, "/", 2)
				objects, err := c.ObjectNamesAll(loParts[0], nil)
				if err != nil {
					fmt.Println(err)
					return
				}
				for _, object := range objects {
					if strings.HasPrefix(object, loParts[1]) {
						c.ObjectDelete(loParts[0], object)
						if err != nil {
							fmt.Println(err)
						}
					}
				}
			}
		},
	}
	rootCmd.AddCommand(cmdDelete)

	rootCmd.Execute()
}

func parseDefaultFlags(flDebug bool) {
	if flDebug {
		os.Setenv("DEBUG", "1")
		debug("Debug mode on")
	}
}

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
}
func debugf(fmt string, v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Printf(fmt, v...)
	}
}

func connect() *swift.Connection {
	c := swift.Connection{
		// This should be your username
		UserName: os.Getenv("ST_USER"),
		// This should be your api key
		ApiKey: os.Getenv("ST_KEY"),
		// This should be a v1 auth url
		AuthUrl: os.Getenv("ST_AUTH"),
	}

	// Authenticate
	err := c.Authenticate()
	if err != nil {
		fmt.Println("Ensure ST_USER, ST_KEY, and ST_AUTH are set")
		panic(err)
	}
	return &c
}
