/*
   Copyright 2021 The simple-router Authors.
   Licensed under the Apache License, PROJECT_VERSION 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at
       http://www.apache.org/licenses/LICENSE-2.0
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package options

import (
	"bufio"
	"fmt"
	"github.com/l0calh0st/simple-router/pkg/controller/simple"
	"github.com/spf13/pflag"
	"k8s.io/component-base/cli/flag"
	"os"
	"strings"
	"syscall"
)

type Options struct {
	// this is example flags
	ListenAddress string
	// todo write your flags here
	MultiTag []string
	NumTags  int
	WhiteFile string
}

var _ options = new(Options)

// NewOptions create an instance option and return
func NewOptions() *Options {
	// todo write your code or change this code here
	return &Options{}
}

// Validate validates options
func (o *Options) Validate() []error {
	// valid tags valid, tags split must be <= 2
	var retErrors []error
	for _, tag := range o.MultiTag {
		tagSlice := strings.Split(tag, "|")
		if len(tagSlice) > 2 {
			retErrors = append(retErrors, fmt.Errorf("Options validate failed: Invalid tags: %v ", tag))
		}
	}

	if o.WhiteFile != ""{
		_,err := os.Stat(o.WhiteFile)
		if err != nil{
			retErrors = append(retErrors, fmt.Errorf("Options validate failed: White file  %v is not exist", o.WhiteFile))
		}
	}
	return retErrors
}

// Complete fill some default value to options
func (o *Options) Complete() error {
	if o.NumTags == 0 {
		o.NumTags = 20
	}

	return nil
}

//
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ListenAddress, "listen-addr", ":8080", "Address on which to expose metrics and web interfaces")
	fs.StringSliceVar(&o.MultiTag, "multi-tags", []string{"tag14|tag7"}, "All multi tags list")
	fs.IntVar(&o.NumTags, "num-tags", 20, "Number of tag place holder, default 20")
	fs.StringVar(&o.WhiteFile, "white-file", "", "Path of White file")
}

func (o *Options) NamedFlagSets() (fs flag.NamedFlagSets) {
	o.AddFlags(fs.FlagSet("simple-router"))
	return fs
}

// Config conver to Config
func (o *Options) Config() simple.Config {
	config := simple.Config{
		MultiTag:     o.MultiTag,
		NumTags:      o.NumTags,
		WhiteDplList: make([]string, 0),
	}
	if o.WhiteFile != ""{
		fp,err := os.OpenFile(o.WhiteFile, syscall.O_RDONLY, os.ModePerm)
		if err != nil{
			panic(err)
		}
		defer fp.Close()
		scanner := bufio.NewScanner(fp)
		for scanner.Scan(){
			config.WhiteDplList = append(config.WhiteDplList, scanner.Text())
		}
	}

	return config
}
