// Copyright © 2021 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/apex/log"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	mcfs "github.com/materials-commons/gomcfs/fs"
	"github.com/materials-commons/gomcfs/mcapi"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	*fuse.Server
	mountPoint string
}

// mountCmd represents the mount command
var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a Materials Commons project as a file system",
	Long: `The 'mount' command will mount a Materials Commons project as a file system giving access to the
project files as if they were local to the computer.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatalf("No path specified for mount.")
		}

		if projectId == -1 {
			log.Fatalf("No project specified.")
		}

		mcapiUrl := os.Getenv("MC_API_URL")
		mcapiToken := os.Getenv("MC_API_TOKEN")
		client := mcapi.NewClient(mcapiUrl, mcapiToken, projectId)
		rootNode := mcfs.RootNode(client)

		server := mustMount(args[0], rootNode)
		go server.listenForUnmount()
		log.Infof("Mounted project at %q, use ctrl+c to stop", args[0])
		server.Wait()
	},
}

var projectId int

func init() {
	rootCmd.AddCommand(mountCmd)
	mountCmd.PersistentFlags().IntVarP(&projectId, "project-id", "p", -1, "Project Id to mount")
}

var timeout = 10 * time.Second

func mustMount(mountPoint string, root *mcfs.Node) *Server {
	opts := &fs.Options{
		AttrTimeout:  &timeout,
		EntryTimeout: &timeout,
		MountOptions: fuse.MountOptions{
			Debug:  false,
			FsName: "mcfs",
		},
	}

	server, err := fs.Mount(mountPoint, root, opts)
	if err != nil {
		log.Fatalf("Unable to mount project %s", err)
	}

	return &Server{
		Server:     server,
		mountPoint: mountPoint,
	}
}

func (s *Server) listenForUnmount() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	sig := <-c
	log.Infof("Got %s signal, unmounting %q...", sig, s.mountPoint)
	if err := s.Unmount(); err != nil {
		log.Errorf("Failed to unmount: %s, try 'umount %s' manually.", err, s.mountPoint)
	}

	<-c
	log.Warnf("Force exiting...")
	os.Exit(1)
}
