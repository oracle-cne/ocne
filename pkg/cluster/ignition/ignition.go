// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ignition

import (
	"encoding/json"
	"fmt"
	"os"

	bututil "github.com/coreos/butane/base/util"
	butconfig "github.com/coreos/butane/config"
	butcommon "github.com/coreos/butane/config/common"
	ignutil "github.com/coreos/ignition/v2/config/util"
	ign34 "github.com/coreos/ignition/v2/config/v3_4"
	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/oracle-cne/ocne/pkg/util"
)

const (
	IgnitionVersion = "3.4.0"
)

type FileContents struct {
	Source string `json:"source"`
}

type File struct {
	Path       string       `json:"path"`
	Filesystem string       `json:"filesystem"`
	Mode       int          `json:"mode"`
	UserId     int          `json:"user"`
	User       string       `json:"username"`
	GroupId    int          `json:"group"`
	Group      string       `json:"groupname"`
	Contents   FileContents `json:"contents"`
	encoded    bool
	Overwrite  bool ` json:"overwrite"`
}

type User struct {
	Name         string   `json:"name"`
	SshKey       string   `json:"sshKey"`
	Password     string   `json:"password"`
	Groups       []string `json:"groups"`
	Shell        string   `json:"shell"`
	System       bool     `json:"system"`
	NoCreateHome bool     `json:"noCreateHome"`
	PrimaryGroup string   `json:"primaryGroup"`
}

type Group struct {
	Name   string `json:"name"`
	System bool   `json:"system"`
}

// AddUser adds a user with the correct variables set, and also checks
// for any conflicts that may have occurred.
func AddUser(ign *igntypes.Config, u *User) error {
	for _, pu := range ign.Passwd.Users {
		if pu.Name == u.Name {
			return fmt.Errorf("A user with name %s is already defined", u.Name)
		}
	}

	var pwh *string
	if u.Password != "" {
		pwh = &u.Password
	}

	var keys []igntypes.SSHAuthorizedKey
	if u.SshKey != "" {
		keys = append(keys, igntypes.SSHAuthorizedKey(u.SshKey))
	}

	var groups []igntypes.Group
	for _, g := range u.Groups {
		groups = append(groups, igntypes.Group(g))
	}

	ign.Passwd.Users = append(ign.Passwd.Users, igntypes.PasswdUser{
		Name:              u.Name,
		PasswordHash:      pwh,
		SSHAuthorizedKeys: keys,
		Groups:            groups,
		Shell:             util.StrPtr(u.Shell),
		System:            util.BoolPtr(u.System),
		NoCreateHome:      util.BoolPtr(u.NoCreateHome),
		PrimaryGroup:      util.StrPtr(u.PrimaryGroup),
	})

	return nil
}

// AddGroup adds a group with the correct variables set, and also checks
// for any conflicts that may have occurred.
func AddGroup(ign *igntypes.Config, g *Group) error {
	for _, pg := range ign.Passwd.Groups {
		if pg.Name == g.Name {
			return fmt.Errorf("A group with name %s is already defined", g.Name)
		}
	}

	ign.Passwd.Groups = append(ign.Passwd.Groups, igntypes.PasswdGroup{
		Name:   g.Name,
		System: util.BoolPtr(g.System),
	})

	return nil
}

// AddFile adds a file with the correct variables set, and also checks for any
// conflicts that may have occurred.
func AddFile(ign *igntypes.Config, f *File) error {
	for _, sf := range ign.Storage.Files {
		if sf.Node.Path == f.Path {
			return fmt.Errorf("A file with path %s is already defined", f.Path)
		}
	}

	data, compressed, err := bututil.MakeDataURL([]byte(f.Contents.Source), nil, true)
	if err != nil {
		return err
	}

	var uidPtr *int
	var gidPtr *int
	var unamePtr *string
	var gnamePtr *string

	if f.UserId != 0 {
		uidPtr = util.IntPtr(f.UserId)
	}
	if f.User != "" {
		unamePtr = util.StrPtr(f.User)
	}
	if f.GroupId != 0 {
		gidPtr = util.IntPtr(f.GroupId)
	}
	if f.Group != "" {
		gnamePtr = util.StrPtr(f.Group)
	}

	ign.Storage.Files = append(ign.Storage.Files, igntypes.File{
		Node: igntypes.Node{
			Path:      f.Path,
			Overwrite: util.BoolPtr(true),
			User: igntypes.NodeUser{
				ID:   uidPtr,
				Name: unamePtr,
			},
			Group: igntypes.NodeGroup{
				ID:   gidPtr,
				Name: gnamePtr,
			},
		},
		FileEmbedded1: igntypes.FileEmbedded1{
			Mode: util.IntPtr(f.Mode),
			Contents: igntypes.Resource{
				Compression: compressed,
				Source:      &data,
			},
		},
	})

	return nil
}

// AddLink adds a link with the correct variables set, and also checks for any
// conflicts that may have occurred.
func AddLink(ign *igntypes.Config, l *igntypes.Link) error {
	for _, sl := range ign.Storage.Links {
		if sl.Node.Path == l.Node.Path {
			return fmt.Errorf("A link with path %s is already defined", l.Node.Path)
		}
	}

	ign.Storage.Links = append(ign.Storage.Links, *l)
	return nil
}

// AddDir adds a directory with the correct variables et, and also checks
// for any conflicts that may have occurred
func AddDir(ign *igntypes.Config, d *igntypes.Directory) error {
	for _, sd := range ign.Storage.Directories {
		if sd.Node.Path == d.Node.Path {
			return fmt.Errorf("A directory with path %s is already defined", d.Node.Path)
		}
	}

	ign.Storage.Directories = append(ign.Storage.Directories, *d)
	return nil
}

// AddUnit adds a unit to an existing ignition config.
func AddUnit(ign *igntypes.Config, unit *igntypes.Unit) *igntypes.Config {
	wrapped := NewIgnition()
	wrapped.Systemd.Units = append(wrapped.Systemd.Units, *unit)
	return Merge(ign, wrapped)
}

// Merge merges two ignition configurations
func Merge(a *igntypes.Config, b *igntypes.Config) *igntypes.Config {
	ret := ign34.Merge(*a, *b)
	return &ret
}

// FromBytes generates an ignition structure from a string.  Both ignition
// and butane formats are accepted.
func FromBytes(in []byte) (*igntypes.Config, error) {
	// First check if this is an ignition string.  Treat the input as
	// ignition if it is well-formed enough to have a reasonable version.
	// Errors are ignored in this check because the only portion of the
	// check that matters is the version string.  It is expected that
	// non-ignition inputs will have errors.
	ver, _, _ := ignutil.GetConfigVersion(in)
	if ver.String() == "0.0.0" {
		// It's not ignition.  Assume the input string is butane.  If
		// it's not then the parser will catch it and complain.
		var err error
		inIgn, report, err := butconfig.TranslateBytes(in, butcommon.TranslateBytesOptions{
			Raw: true,
		})

		// Treat warnings as errors so that it's hard to propagate mistakes
		if len(report.String()) > 0 {
			return nil, fmt.Errorf("could not parse extra ignition: %s", report.String())
		}

		if err != nil {
			return nil, err
		}

		in = inIgn
	}

	ret, report, err := ign34.ParseCompatibleVersion(in)

	if len(report.String()) > 0 {
		return nil, fmt.Errorf("could not parse extra ignition: %s", report.String())
	}

	if err != nil {
		return nil, err
	}
	return &ret, nil
}

// FromString generates an ignition structure from a string.  Both ignition
// and butane formats are accepted.
func FromString(in string) (*igntypes.Config, error) {
	return FromBytes([]byte(in))
}

// FromPath generates an ignition structure from a path.  If the path is
// a directory, it merges together all valid ignition files in the directory.
// If it is a file, it uses just the file.
func FromPath(path string) (*igntypes.Config, error) {
	paths, err := util.FilesFromPath(path)
	if err != nil {
		return nil, err
	}

	// Special case when a path is specifically to a file.  In that
	// case, ignition parsing errors are actually errors
	if len(paths) == 1 && paths[0] == path {
		fileBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return FromBytes(fileBytes)
	}

	ret := NewIgnition()
	for _, p := range paths {
		fileBytes, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}

		ign, err := FromBytes(fileBytes)
		// Ignore errors.  Assume those files are not
		// meant to be included and are just other cruft
		// in the directory.
		if err == nil {
			ret = Merge(ret, ign)
		}
	}

	return ret, nil
}

// NewIgnition initializes an Ingition with no settings.
func NewIgnition() *igntypes.Config {
	return &igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: IgnitionVersion,
		},
	}
}

// MarshalIgnition converts and ignition configuration to a byte array
// containing the json encoding of the configuration.
func MarshalIgnition(ign *igntypes.Config) ([]byte, error) {
	return json.Marshal(ign)
}
